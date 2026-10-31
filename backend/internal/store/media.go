package store

import (
	"context"
	"fmt"

	"github.com/Liao-Jun-Tao/couple-daily/internal/apierr"
	"github.com/Liao-Jun-Tao/couple-daily/internal/idutil"
	"github.com/jackc/pgx/v5"
)

// CreateMedia records an upload intent owned by the caller.
func (db *DB) CreateMedia(ctx context.Context, userID, spaceID, contentType string, byteSize int64) (id, storageKey string, err error) {
	id = idutil.New()
	storageKey = "media/" + spaceID + "/" + id
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO media_assets (id, uploader_user_id, space_id, content_type, byte_size, storage_key, status)
		VALUES ($1, $2, $3, $4, $5, $6, 'pending_upload')
	`, id, userID, spaceID, contentType, byteSize, storageKey)
	if err != nil {
		return "", "", fmt.Errorf("create media: %w", err)
	}
	return id, storageKey, nil
}

func lockMedia(ctx context.Context, tx pgx.Tx, spaceID, authorID, currentPostID string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	rows, err := tx.Query(ctx, `
		SELECT m.id::text,
		       m.uploader_user_id::text,
		       COALESCE(m.space_id::text, ''),
		       m.status,
		       EXISTS (
		         SELECT 1 FROM post_media pm
		         WHERE pm.media_id = m.id
		           AND ($2::text = '' OR pm.post_id::text <> $2)
		       )
		FROM media_assets m
		WHERE m.id::text = ANY($1::text[])
		FOR UPDATE
	`, ids, currentPostID)
	if err != nil {
		return fmt.Errorf("lock media: %w", err)
	}
	defer rows.Close()
	type row struct {
		uploader string
		space    string
		status   string
		linked   bool
	}
	found := map[string]row{}
	for rows.Next() {
		var id string
		var item row
		if err := rows.Scan(&id, &item.uploader, &item.space, &item.status, &item.linked); err != nil {
			return err
		}
		found[id] = item
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(found) != len(ids) {
		return apierr.ErrMediaInvalid
	}
	for _, id := range ids {
		item := found[id]
		if item.uploader != authorID || item.status == "failed" || item.linked {
			return apierr.ErrMediaInvalid
		}
		if item.space != "" && item.space != spaceID {
			return apierr.ErrMediaInvalid
		}
	}
	return nil
}

func linkMedia(ctx context.Context, tx pgx.Tx, spaceID, postID string, ids []string) error {
	for i, id := range ids {
		if _, err := tx.Exec(ctx, `
			INSERT INTO post_media (post_id, media_id, sort_order) VALUES ($1, $2, $3)
		`, postID, id, i); err != nil {
			return fmt.Errorf("link media: %w", err)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	if _, err := tx.Exec(ctx, `
		UPDATE media_assets SET space_id = $1 WHERE id::text = ANY($2::text[])
	`, spaceID, ids); err != nil {
		return fmt.Errorf("bind media space: %w", err)
	}
	return nil
}
