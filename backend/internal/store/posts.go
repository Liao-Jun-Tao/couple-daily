package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/Liao-Jun-Tao/couple-daily/internal/apierr"
	"github.com/Liao-Jun-Tao/couple-daily/internal/authz"
	"github.com/jackc/pgx/v5"
)

// ListPosts returns up to limit posts, newest first.
func (db *DB) ListPosts(ctx context.Context, spaceID, viewerID string, limit int, before Page) ([]Post, error) {
	return loadPosts(ctx, db.Pool, spaceID, viewerID, "", before, limit)
}

// GetPost loads one live post in the space.
func (db *DB) GetPost(ctx context.Context, spaceID, viewerID, postID string) (Post, error) {
	items, err := loadPosts(ctx, db.Pool, spaceID, viewerID, postID, PageFrom(nil, ""), 1)
	if err != nil {
		return Post{}, err
	}
	if len(items) == 0 {
		return Post{}, apierr.ErrNotFound
	}
	return items[0], nil
}

// CreatePost inserts a post and its media.
func (db *DB) CreatePost(ctx context.Context, spaceID, authorID, body string, loc *Geo, mediaIDs []string) (Post, error) {
	var out Post
	err := db.withTx(ctx, func(tx pgx.Tx) error {
		if err := lockMedia(ctx, tx, spaceID, authorID, "", mediaIDs); err != nil {
			return err
		}
		postID, err := insertPost(ctx, tx, spaceID, authorID, body, loc)
		if err != nil {
			return err
		}
		if err := linkMedia(ctx, tx, spaceID, postID, mediaIDs); err != nil {
			return err
		}
		items, err := loadPosts(ctx, tx, spaceID, authorID, postID, PageFrom(nil, ""), 1)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return apierr.ErrNotFound
		}
		out = items[0]
		return nil
	})
	return out, err
}

// UpdatePost replaces body, location, and the media list. Only the author may update.
func (db *DB) UpdatePost(ctx context.Context, spaceID, postID, actorID, body string, loc *Geo, mediaIDs []string) (Post, error) {
	var out Post
	err := db.withTx(ctx, func(tx pgx.Tx) error {
		author, err := lockLivePost(ctx, tx, spaceID, postID)
		if err != nil {
			return err
		}
		if !authz.AuthorOnly(actorID, author) {
			return apierr.ErrForbidden
		}
		if _, err := tx.Exec(ctx, `DELETE FROM post_media WHERE post_id = $1`, postID); err != nil {
			return err
		}
		if err := lockMedia(ctx, tx, spaceID, actorID, postID, mediaIDs); err != nil {
			return err
		}
		if err := updatePostRow(ctx, tx, postID, body, loc); err != nil {
			return err
		}
		if err := linkMedia(ctx, tx, spaceID, postID, mediaIDs); err != nil {
			return err
		}
		items, err := loadPosts(ctx, tx, spaceID, actorID, postID, PageFrom(nil, ""), 1)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return apierr.ErrNotFound
		}
		out = items[0]
		return nil
	})
	return out, err
}

// DeletePost soft-deletes a post. Only the author may delete it.
func (db *DB) DeletePost(ctx context.Context, spaceID, postID, actorID string) error {
	author, err := db.liveAuthor(ctx, "posts", spaceID, postID)
	if err != nil {
		return err
	}
	if !authz.AuthorOnly(actorID, author) {
		return apierr.ErrForbidden
	}
	_, err = db.Pool.Exec(ctx, `
		UPDATE posts SET deleted_at = now(), updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
	`, postID)
	return err
}

func (db *DB) liveAuthor(ctx context.Context, table, spaceID, id string) (string, error) {
	var author, gotSpace string
	var deleted bool
	query := fmt.Sprintf(`
		SELECT space_id::text, author_user_id::text, deleted_at IS NOT NULL
		FROM %s WHERE id = $1
	`, table)
	if table != "posts" && table != "comments" {
		return "", fmt.Errorf("unsupported table %s", table)
	}
	err := db.Pool.QueryRow(ctx, query, id).Scan(&gotSpace, &author, &deleted)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && (gotSpace != spaceID || deleted)) {
		return "", apierr.ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return author, nil
}

func lockLivePost(ctx context.Context, tx pgx.Tx, spaceID, postID string) (string, error) {
	var author, gotSpace string
	var deleted bool
	err := tx.QueryRow(ctx, `
		SELECT space_id::text, author_user_id::text, deleted_at IS NOT NULL
		FROM posts WHERE id = $1 FOR UPDATE
	`, postID).Scan(&gotSpace, &author, &deleted)
	if errors.Is(err, pgx.ErrNoRows) || gotSpace != spaceID || deleted {
		return "", apierr.ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return author, nil
}

func insertPost(ctx context.Context, tx pgx.Tx, spaceID, authorID, body string, loc *Geo) (string, error) {
	lat, lng, name, place, vis := locationArgs(loc)
	var id string
	err := tx.QueryRow(ctx, `
		INSERT INTO posts (
			space_id, author_user_id, body,
			location_lat, location_lng, location_name, location_place_id, location_visibility
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id::text
	`, spaceID, authorID, body, lat, lng, name, place, vis).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("insert post: %w", err)
	}
	return id, nil
}

func updatePostRow(ctx context.Context, tx pgx.Tx, postID, body string, loc *Geo) error {
	lat, lng, name, place, vis := locationArgs(loc)
	tag, err := tx.Exec(ctx, `
		UPDATE posts SET
			body = $2,
			location_lat = $3,
			location_lng = $4,
			location_name = $5,
			location_place_id = $6,
			location_visibility = $7,
			updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
	`, postID, body, lat, lng, name, place, vis)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apierr.ErrNotFound
	}
	return nil
}

func locationArgs(loc *Geo) (lat, lng, name, place any, vis string) {
	vis = "exact"
	if loc == nil {
		return nil, nil, nil, nil, vis
	}
	vis = loc.Visibility
	if vis == "" {
		vis = "exact"
	}
	return loc.Lat, loc.Lng, loc.Name, loc.PlaceID, vis
}

func loadPosts(ctx context.Context, q queryer, spaceID, viewerID, onlyID string, before Page, limit int) ([]Post, error) {
	if onlyID == "" {
		onlyID = ""
	}
	rows, err := q.Query(ctx, `
		SELECT p.id::text,
		       p.space_id::text,
		       p.body,
		       p.location_lat,
		       p.location_lng,
		       p.location_name,
		       p.location_place_id,
		       p.location_visibility,
		       p.created_at,
		       p.updated_at,
		       u.id::text,
		       u.email::text,
		       u.display_name,
		       u.created_at,
		       (SELECT count(*) FROM likes l WHERE l.post_id = p.id),
		       EXISTS (SELECT 1 FROM likes l WHERE l.post_id = p.id AND l.user_id = $2::uuid),
		       (SELECT count(*) FROM comments c WHERE c.post_id = p.id AND c.deleted_at IS NULL)
		FROM posts p
		JOIN users u ON u.id = p.author_user_id
		WHERE p.space_id = $1::uuid
		  AND p.deleted_at IS NULL
		  AND ($3::text = '' OR p.id::text = $3)
		  AND ($4::bool = false OR (p.created_at, p.id) < ($5::timestamptz, $6::uuid))
		ORDER BY p.created_at DESC, p.id DESC
		LIMIT $7
	`, spaceID, viewerID, onlyID, before.Has, before.Time, before.ID, limit)
	if err != nil {
		return nil, fmt.Errorf("list posts: %w", err)
	}
	defer rows.Close()
	var posts []Post
	for rows.Next() {
		var p Post
		var lat, lng *float64
		var name, place *string
		var vis string
		var likes, comments int64
		if err := rows.Scan(
			&p.ID, &p.SpaceID, &p.Body,
			&lat, &lng, &name, &place, &vis,
			&p.CreatedAt, &p.UpdatedAt,
			&p.Author.ID, &p.Author.Email, &p.Author.DisplayName, &p.Author.CreatedAt,
			&likes, &p.LikedByMe, &comments,
		); err != nil {
			return nil, err
		}
		p.LikeCount = int(likes)
		p.CommentCount = int(comments)
		p.Media = []PostMedia{}
		if lat != nil && lng != nil {
			p.Location = &Geo{Lat: *lat, Lng: *lng, Name: name, PlaceID: place, Visibility: vis}
		}
		posts = append(posts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(posts) == 0 {
		return []Post{}, nil
	}
	ids := make([]string, len(posts))
	index := map[string]int{}
	for i, p := range posts {
		ids[i] = p.ID
		index[p.ID] = i
	}
	mediaRows, err := q.Query(ctx, `
		SELECT pm.post_id::text, m.id::text, m.content_type, m.storage_key, pm.sort_order::int
		FROM post_media pm
		JOIN media_assets m ON m.id = pm.media_id
		WHERE pm.post_id::text = ANY($1::text[])
		ORDER BY pm.sort_order ASC, m.id
	`, ids)
	if err != nil {
		return nil, fmt.Errorf("post media: %w", err)
	}
	defer mediaRows.Close()
	for mediaRows.Next() {
		var postID string
		var media PostMedia
		if err := mediaRows.Scan(&postID, &media.ID, &media.ContentType, &media.StorageKey, &media.SortOrder); err != nil {
			return nil, err
		}
		i, ok := index[postID]
		if ok {
			posts[i].Media = append(posts[i].Media, media)
		}
	}
	if err := mediaRows.Err(); err != nil {
		return nil, err
	}
	return posts, nil
}
