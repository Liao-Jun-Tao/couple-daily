package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Liao-Jun-Tao/couple-daily/internal/apierr"
	"github.com/Liao-Jun-Tao/couple-daily/internal/invite"
	"github.com/jackc/pgx/v5"
)

// Membership returns the caller's space or ErrNoSpace.
func (db *DB) Membership(ctx context.Context, userID string) (Membership, error) {
	var m Membership
	err := db.Pool.QueryRow(ctx, `
		SELECT s.id::text, s.member_count::int, s.created_at
		FROM space_members sm
		JOIN couple_spaces s ON s.id = sm.space_id
		WHERE sm.user_id = $1
	`, userID).Scan(&m.SpaceID, &m.MemberCount, &m.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Membership{}, ErrNoSpace
	}
	if err != nil {
		return Membership{}, fmt.Errorf("membership: %w", err)
	}
	return m, nil
}

// SpaceForUser loads the caller's space and its members.
func (db *DB) SpaceForUser(ctx context.Context, userID string) (Space, error) {
	m, err := db.Membership(ctx, userID)
	if err != nil {
		return Space{}, err
	}
	return db.loadSpace(ctx, db.Pool, m.SpaceID)
}

// CreateInvite creates a one-member space and a pending invite.
func (db *DB) CreateInvite(ctx context.Context, userID, code string, expires time.Time) (Space, error) {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return Space{}, err
	}
	defer tx.Rollback(ctx)

	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM space_members WHERE user_id = $1)`, userID).Scan(&exists); err != nil {
		return Space{}, err
	}
	if exists {
		return Space{}, apierr.ErrAlreadyPaired
	}

	var spaceID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO couple_spaces (member_count) VALUES (1) RETURNING id::text
	`).Scan(&spaceID); err != nil {
		return Space{}, fmt.Errorf("create space: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO space_members (space_id, user_id, role) VALUES ($1, $2, 'owner')
	`, spaceID, userID); err != nil {
		if uniqueViolation(err) {
			return Space{}, apierr.ErrAlreadyPaired
		}
		return Space{}, fmt.Errorf("create member: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO invite_codes (code, inviter_user_id, space_id, status, expires_at)
		VALUES ($1, $2, $3, 'pending', $4)
	`, code, userID, spaceID, expires); err != nil {
		if uniqueViolation(err) {
			return Space{}, ErrInviteCodeConflict
		}
		return Space{}, fmt.Errorf("create invite: %w", err)
	}
	space, err := db.loadSpace(ctx, tx, spaceID)
	if err != nil {
		return Space{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Space{}, err
	}
	return space, nil
}

// AcceptInvite joins the acceptor to the inviter's space.
func (db *DB) AcceptInvite(ctx context.Context, acceptorID, code string, now time.Time) (Space, error) {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return Space{}, err
	}
	defer tx.Rollback(ctx)

	var inviterID, spaceID, status string
	var expires time.Time
	err = tx.QueryRow(ctx, `
		SELECT inviter_user_id::text, space_id::text, status, expires_at
		FROM invite_codes WHERE code = $1
		FOR UPDATE
	`, code).Scan(&inviterID, &spaceID, &status, &expires)
	if errors.Is(err, pgx.ErrNoRows) {
		return Space{}, apierr.ErrInviteNotFound
	}
	if err != nil {
		return Space{}, fmt.Errorf("load invite: %w", err)
	}

	var memberCount int
	if err := tx.QueryRow(ctx, `
		SELECT member_count::int FROM couple_spaces WHERE id = $1 FOR UPDATE
	`, spaceID).Scan(&memberCount); err != nil {
		return Space{}, fmt.Errorf("lock space: %w", err)
	}
	var acceptorInSpace bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM space_members WHERE user_id = $1)
	`, acceptorID).Scan(&acceptorInSpace); err != nil {
		return Space{}, err
	}

	decision := invite.Decide(invite.Input{
		Status:          status,
		ExpiresAt:       expires,
		MemberCount:     memberCount,
		InviterID:       inviterID,
		AcceptorID:      acceptorID,
		AcceptorInSpace: acceptorInSpace,
		Now:             now,
	})
	if errors.Is(decision, apierr.ErrInviteExpired) && status == "pending" {
		if _, err := tx.Exec(ctx, `UPDATE invite_codes SET status = 'expired' WHERE code = $1`, code); err != nil {
			return Space{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return Space{}, err
		}
		return Space{}, decision
	}
	if decision != nil {
		return Space{}, decision
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO space_members (space_id, user_id, role) VALUES ($1, $2, 'member')
	`, spaceID, acceptorID); err != nil {
		if uniqueViolation(err) {
			return Space{}, apierr.ErrAlreadyPaired
		}
		return Space{}, fmt.Errorf("join space: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE couple_spaces SET member_count = 2, updated_at = $2 WHERE id = $1
	`, spaceID, now); err != nil {
		return Space{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE invite_codes
		SET status = 'accepted', accepted_by_user_id = $2, accepted_at = $3
		WHERE code = $1
	`, code, acceptorID, now); err != nil {
		return Space{}, err
	}
	space, err := db.loadSpace(ctx, tx, spaceID)
	if err != nil {
		return Space{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Space{}, err
	}
	return space, nil
}

func (db *DB) loadSpace(ctx context.Context, q queryer, spaceID string) (Space, error) {
	rows, err := q.Query(ctx, `
		SELECT s.id::text, s.member_count::int, s.created_at,
		       u.id::text, u.display_name, sm.joined_at
		FROM couple_spaces s
		JOIN space_members sm ON sm.space_id = s.id
		JOIN users u ON u.id = sm.user_id
		WHERE s.id = $1
		ORDER BY sm.joined_at ASC, u.id
	`, spaceID)
	if err != nil {
		return Space{}, fmt.Errorf("load space: %w", err)
	}
	defer rows.Close()
	var space Space
	for rows.Next() {
		var member Member
		if err := rows.Scan(&space.ID, &space.MemberCount, &space.CreatedAt, &member.UserID, &member.DisplayName, &member.JoinedAt); err != nil {
			return Space{}, err
		}
		space.Members = append(space.Members, member)
	}
	if err := rows.Err(); err != nil {
		return Space{}, err
	}
	if space.ID == "" {
		return Space{}, apierr.ErrNotFound
	}
	if space.Members == nil {
		space.Members = []Member{}
	}
	return space, nil
}
