package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/Liao-Jun-Tao/couple-daily/internal/apierr"
	"github.com/Liao-Jun-Tao/couple-daily/internal/authz"
	"github.com/jackc/pgx/v5"
)

func (db *DB) ensurePost(ctx context.Context, spaceID, postID string) error {
	var exists bool
	err := db.Pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM posts
			WHERE id = $1 AND space_id = $2 AND deleted_at IS NULL
		)
	`, postID, spaceID).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return apierr.ErrNotFound
	}
	return nil
}

// SetLike adds or removes the caller's like and returns the new status.
func (db *DB) SetLike(ctx context.Context, spaceID, postID, userID string, like bool) (liked bool, count int, err error) {
	if err = db.ensurePost(ctx, spaceID, postID); err != nil {
		return false, 0, err
	}
	if like {
		_, err = db.Pool.Exec(ctx, `
			INSERT INTO likes (space_id, post_id, user_id) VALUES ($1, $2, $3)
			ON CONFLICT (post_id, user_id) DO NOTHING
		`, spaceID, postID, userID)
	} else {
		_, err = db.Pool.Exec(ctx, `DELETE FROM likes WHERE post_id = $1 AND user_id = $2`, postID, userID)
	}
	if err != nil {
		return false, 0, fmt.Errorf("set like: %w", err)
	}
	var n int64
	err = db.Pool.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM likes WHERE post_id = $1),
		       EXISTS(SELECT 1 FROM likes WHERE post_id = $1 AND user_id = $2)
	`, postID, userID).Scan(&n, &liked)
	if err != nil {
		return false, 0, err
	}
	return liked, int(n), nil
}

// ListComments returns comments oldest first.
func (db *DB) ListComments(ctx context.Context, spaceID, postID string, limit int, after Page) ([]Comment, error) {
	if err := db.ensurePost(ctx, spaceID, postID); err != nil {
		return nil, err
	}
	rows, err := db.Pool.Query(ctx, `
		SELECT c.id::text, c.post_id::text, c.body, c.created_at,
		       u.id::text, u.email::text, u.display_name, u.created_at
		FROM comments c
		JOIN users u ON u.id = c.author_user_id
		WHERE c.post_id = $1 AND c.deleted_at IS NULL
		  AND ($2::bool = false OR (c.created_at, c.id) > ($3::timestamptz, $4::uuid))
		ORDER BY c.created_at ASC, c.id ASC
		LIMIT $5
	`, postID, after.Has, after.Time, after.ID, limit)
	if err != nil {
		return nil, fmt.Errorf("list comments: %w", err)
	}
	defer rows.Close()
	var items []Comment
	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.ID, &c.PostID, &c.Body, &c.CreatedAt, &c.Author.ID, &c.Author.Email, &c.Author.DisplayName, &c.Author.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	if items == nil {
		items = []Comment{}
	}
	return items, rows.Err()
}

// CreateComment inserts a comment on a live post in the space.
func (db *DB) CreateComment(ctx context.Context, spaceID, postID, authorID, body string) (Comment, error) {
	if err := db.ensurePost(ctx, spaceID, postID); err != nil {
		return Comment{}, err
	}
	var c Comment
	err := db.Pool.QueryRow(ctx, `
		WITH inserted AS (
			INSERT INTO comments (space_id, post_id, author_user_id, body)
			VALUES ($1, $2, $3, $4)
			RETURNING id, post_id, author_user_id, body, created_at
		)
		SELECT i.id::text, i.post_id::text, i.body, i.created_at,
		       u.id::text, u.email::text, u.display_name, u.created_at
		FROM inserted i
		JOIN users u ON u.id = i.author_user_id
	`, spaceID, postID, authorID, body).Scan(
		&c.ID, &c.PostID, &c.Body, &c.CreatedAt,
		&c.Author.ID, &c.Author.Email, &c.Author.DisplayName, &c.Author.CreatedAt,
	)
	if err != nil {
		return Comment{}, fmt.Errorf("create comment: %w", err)
	}
	return c, nil
}

// DeleteComment soft-deletes a comment. Only the author may delete it.
func (db *DB) DeleteComment(ctx context.Context, spaceID, commentID, actorID string) error {
	var author, gotSpace string
	var deleted bool
	err := db.Pool.QueryRow(ctx, `
		SELECT space_id::text, author_user_id::text, deleted_at IS NOT NULL
		FROM comments WHERE id = $1
	`, commentID).Scan(&gotSpace, &author, &deleted)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && (gotSpace != spaceID || deleted)) {
		return apierr.ErrNotFound
	}
	if err != nil {
		return err
	}
	if !authz.AuthorOnly(actorID, author) {
		return apierr.ErrForbidden
	}
	_, err = db.Pool.Exec(ctx, `UPDATE comments SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, commentID)
	return err
}
