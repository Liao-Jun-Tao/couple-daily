package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Liao-Jun-Tao/couple-daily/internal/apierr"
	"github.com/jackc/pgx/v5"
)

// CreateUser inserts an account. The email must already be canonical.
func (db *DB) CreateUser(ctx context.Context, email, passwordHash, displayName string) (User, error) {
	var u User
	err := db.Pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, display_name)
		VALUES ($1, $2, $3)
		RETURNING id::text, email::text, display_name, created_at
	`, email, passwordHash, displayName).Scan(&u.ID, &u.Email, &u.DisplayName, &u.CreatedAt)
	if uniqueViolation(err) {
		return User{}, apierr.ErrEmailTaken
	}
	if err != nil {
		return User{}, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

// UserByEmail loads a user for login, including the password hash.
func (db *DB) UserByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := db.Pool.QueryRow(ctx, `
		SELECT id::text, email::text, password_hash, display_name, created_at
		FROM users WHERE email = $1
	`, email).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, apierr.ErrInvalidCredentials
	}
	if err != nil {
		return User{}, fmt.Errorf("user by email: %w", err)
	}
	return u, nil
}

// UserByID loads the public account fields.
func (db *DB) UserByID(ctx context.Context, id string) (User, error) {
	var u User
	err := db.Pool.QueryRow(ctx, `
		SELECT id::text, email::text, display_name, created_at
		FROM users WHERE id = $1
	`, id).Scan(&u.ID, &u.Email, &u.DisplayName, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, apierr.ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("user by id: %w", err)
	}
	return u, nil
}

// InsertRefresh stores a new refresh token hash.
func (db *DB) InsertRefresh(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, userID, tokenHash, expiresAt)
	if err != nil {
		return fmt.Errorf("insert refresh: %w", err)
	}
	return nil
}

// RotateRefresh revokes oldHash and inserts newHash for the same user.
func (db *DB) RotateRefresh(ctx context.Context, oldHash, newHash string, newExp time.Time, now time.Time) (string, error) {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var userID string
	var expires time.Time
	var revoked *time.Time
	err = tx.QueryRow(ctx, `
		SELECT user_id::text, expires_at, revoked_at
		FROM refresh_tokens WHERE token_hash = $1
		FOR UPDATE
	`, oldHash).Scan(&userID, &expires, &revoked)
	if errors.Is(err, pgx.ErrNoRows) || revoked != nil || !expires.After(now) {
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("load refresh: %w", err)
		}
		return "", apierr.ErrTokenInvalid
	}
	if _, err := tx.Exec(ctx, `UPDATE refresh_tokens SET revoked_at = $2 WHERE token_hash = $1`, oldHash, now); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)
	`, userID, newHash, newExp); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return userID, nil
}
