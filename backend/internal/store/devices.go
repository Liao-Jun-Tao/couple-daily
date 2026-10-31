package store

import (
	"context"
	"fmt"
)

// UpsertDevice registers or refreshes a push token for the user.
func (db *DB) UpsertDevice(ctx context.Context, userID, platform, token string) (Device, error) {
	var d Device
	err := db.Pool.QueryRow(ctx, `
		INSERT INTO device_tokens (user_id, platform, token)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, token) DO UPDATE
		SET platform = EXCLUDED.platform,
		    updated_at = now(),
		    last_seen_at = now()
		RETURNING id::text, platform, token, updated_at
	`, userID, platform, token).Scan(&d.ID, &d.Platform, &d.Token, &d.UpdatedAt)
	if err != nil {
		return Device{}, fmt.Errorf("upsert device: %w", err)
	}
	return d, nil
}

// DeleteDevice removes one of the caller's tokens. Missing tokens are not an error.
func (db *DB) DeleteDevice(ctx context.Context, userID, token string) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM device_tokens WHERE user_id = $1 AND token = $2`, userID, token)
	return err
}
