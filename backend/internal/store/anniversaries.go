package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/Liao-Jun-Tao/couple-daily/internal/apierr"
	"github.com/Liao-Jun-Tao/couple-daily/internal/authz"
	"github.com/jackc/pgx/v5"
)

// ListAnniversaries returns anniversaries newest first.
func (db *DB) ListAnniversaries(ctx context.Context, spaceID string, limit int, before Page) ([]Anniversary, error) {
	rows, err := db.Pool.Query(ctx, anniversarySelect+`
		WHERE a.space_id = $1 AND a.deleted_at IS NULL
		  AND ($2::bool = false OR (a.created_at, a.id) < ($3::timestamptz, $4::uuid))
		ORDER BY a.created_at DESC, a.id DESC
		LIMIT $5
	`, spaceID, before.Has, before.Time, before.ID, limit)
	if err != nil {
		return nil, fmt.Errorf("list anniversaries: %w", err)
	}
	defer rows.Close()
	return scanAnniversaries(rows)
}

// CreateAnniversary inserts a reminder configuration.
func (db *DB) CreateAnniversary(ctx context.Context, spaceID, creatorID string, a Anniversary) (Anniversary, error) {
	var id string
	err := db.Pool.QueryRow(ctx, `
		INSERT INTO anniversaries (
			space_id, creator_user_id, title, event_date, recurrence, remind,
			remind_days_before, remind_local_time, timezone
		) VALUES ($1, $2, $3, $4::date, $5, $6, $7, $8::time, $9)
		RETURNING id::text
	`, spaceID, creatorID, a.Title, a.EventDate, a.Recurrence, a.Remind, a.RemindDaysBefore, a.RemindLocalTime, a.Timezone).Scan(&id)
	if err != nil {
		return Anniversary{}, fmt.Errorf("create anniversary: %w", err)
	}
	return db.getAnniversary(ctx, spaceID, id)
}

// UpdateAnniversary replaces the editable fields. Any space member may update.
func (db *DB) UpdateAnniversary(ctx context.Context, spaceID, id string, a Anniversary) (Anniversary, error) {
	tag, err := db.Pool.Exec(ctx, `
		UPDATE anniversaries SET
			title = $3,
			event_date = $4::date,
			recurrence = $5,
			remind = $6,
			remind_days_before = $7,
			remind_local_time = $8::time,
			timezone = $9,
			updated_at = now()
		WHERE id = $1 AND space_id = $2 AND deleted_at IS NULL
	`, id, spaceID, a.Title, a.EventDate, a.Recurrence, a.Remind, a.RemindDaysBefore, a.RemindLocalTime, a.Timezone)
	if err != nil {
		return Anniversary{}, fmt.Errorf("update anniversary: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return Anniversary{}, apierr.ErrNotFound
	}
	return db.getAnniversary(ctx, spaceID, id)
}

// GetAnniversary loads one live anniversary in the space.
func (db *DB) GetAnniversary(ctx context.Context, spaceID, id string) (Anniversary, error) {
	return db.getAnniversary(ctx, spaceID, id)
}

func (db *DB) getAnniversary(ctx context.Context, spaceID, id string) (Anniversary, error) {
	rows, err := db.Pool.Query(ctx, anniversarySelect+`
		WHERE a.id = $1 AND a.space_id = $2 AND a.deleted_at IS NULL
	`, id, spaceID)
	if err != nil {
		return Anniversary{}, err
	}
	defer rows.Close()
	items, err := scanAnniversaries(rows)
	if err != nil {
		return Anniversary{}, err
	}
	if len(items) == 0 {
		return Anniversary{}, apierr.ErrNotFound
	}
	return items[0], nil
}

// DeleteAnniversary soft-deletes an anniversary. Only the author may delete it.
func (db *DB) DeleteAnniversary(ctx context.Context, spaceID, id, actorID string) error {
	item, err := db.getAnniversary(ctx, spaceID, id)
	if err != nil {
		return err
	}
	if !authz.AuthorOnly(actorID, item.CreatorID) {
		return apierr.ErrForbidden
	}
	_, err = db.Pool.Exec(ctx, `UPDATE anniversaries SET deleted_at = now(), updated_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	return err
}

// ListReminders returns anniversaries that may produce a push.
func (db *DB) ListReminders(ctx context.Context) ([]ReminderRow, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id::text, space_id::text, title, event_date::text, recurrence,
		       remind_days_before::int, to_char(remind_local_time, 'HH24:MI'), timezone
		FROM anniversaries
		WHERE remind = true AND deleted_at IS NULL
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ReminderRow
	for rows.Next() {
		var row ReminderRow
		if err := rows.Scan(&row.ID, &row.SpaceID, &row.Title, &row.EventDate, &row.Recurrence, &row.DaysBefore, &row.LocalTime, &row.Timezone); err != nil {
			return nil, err
		}
		items = append(items, row)
	}
	return items, rows.Err()
}

// DevicesForSpace returns every registered token belonging to the space.
func (db *DB) DevicesForSpace(ctx context.Context, spaceID string) ([]DeviceRef, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT d.platform, d.token
		FROM device_tokens d
		JOIN space_members sm ON sm.user_id = d.user_id
		WHERE sm.space_id = $1
	`, spaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []DeviceRef
	for rows.Next() {
		var d DeviceRef
		if err := rows.Scan(&d.Platform, &d.Token); err != nil {
			return nil, err
		}
		items = append(items, d)
	}
	return items, rows.Err()
}

const anniversarySelect = `
	SELECT a.id::text, a.space_id::text, a.creator_user_id::text, a.title, a.event_date::text,
	       a.recurrence, a.remind, a.remind_days_before::int,
	       to_char(a.remind_local_time, 'HH24:MI'), a.timezone,
	       a.created_at, a.updated_at,
	       u.id::text, u.email::text, u.display_name, u.created_at
	FROM anniversaries a
	JOIN users u ON u.id = a.creator_user_id
`

func scanAnniversaries(rows pgx.Rows) ([]Anniversary, error) {
	var items []Anniversary
	for rows.Next() {
		var a Anniversary
		if err := rows.Scan(
			&a.ID, &a.SpaceID, &a.CreatorID, &a.Title, &a.EventDate,
			&a.Recurrence, &a.Remind, &a.RemindDaysBefore,
			&a.RemindLocalTime, &a.Timezone,
			&a.CreatedAt, &a.UpdatedAt,
			&a.Creator.ID, &a.Creator.Email, &a.Creator.DisplayName, &a.Creator.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, a)
	}
	if items == nil {
		items = []Anniversary{}
	}
	if err := rows.Err(); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	return items, nil
}
