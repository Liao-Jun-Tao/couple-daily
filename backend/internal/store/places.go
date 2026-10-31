package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Liao-Jun-Tao/couple-daily/internal/apierr"
	"github.com/Liao-Jun-Tao/couple-daily/internal/authz"
	"github.com/jackc/pgx/v5"
)

// ListCheckins returns check-ins newest first.
func (db *DB) ListCheckins(ctx context.Context, spaceID string, limit int, before Page) ([]Checkin, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT c.id::text, c.space_id::text, c.name, c.address, c.lat, c.lng, c.place_id,
		       c.checked_in_at, c.created_at,
		       u.id::text, u.email::text, u.display_name, u.created_at
		FROM restaurant_checkins c
		JOIN users u ON u.id = c.creator_user_id
		WHERE c.space_id = $1 AND c.deleted_at IS NULL
		  AND ($2::bool = false OR (c.checked_in_at, c.id) < ($3::timestamptz, $4::uuid))
		ORDER BY c.checked_in_at DESC, c.id DESC
		LIMIT $5
	`, spaceID, before.Has, before.Time, before.ID, limit)
	if err != nil {
		return nil, fmt.Errorf("list checkins: %w", err)
	}
	defer rows.Close()
	return scanCheckins(rows)
}

// CreateCheckin inserts a hand-entered or searched restaurant visit.
func (db *DB) CreateCheckin(ctx context.Context, spaceID, creatorID, name string, address, placeID *string, lat, lng *float64, checkedInAt time.Time) (Checkin, error) {
	var id string
	err := db.Pool.QueryRow(ctx, `
		INSERT INTO restaurant_checkins (
			space_id, creator_user_id, name, address, lat, lng, place_id, checked_in_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id::text
	`, spaceID, creatorID, name, address, lat, lng, placeID, checkedInAt).Scan(&id)
	if err != nil {
		return Checkin{}, fmt.Errorf("create checkin: %w", err)
	}
	return db.getCheckin(ctx, spaceID, id)
}

func (db *DB) getCheckin(ctx context.Context, spaceID, id string) (Checkin, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT c.id::text, c.space_id::text, c.name, c.address, c.lat, c.lng, c.place_id,
		       c.checked_in_at, c.created_at,
		       u.id::text, u.email::text, u.display_name, u.created_at
		FROM restaurant_checkins c
		JOIN users u ON u.id = c.creator_user_id
		WHERE c.id = $1 AND c.space_id = $2 AND c.deleted_at IS NULL
	`, id, spaceID)
	if err != nil {
		return Checkin{}, err
	}
	defer rows.Close()
	items, err := scanCheckins(rows)
	if err != nil {
		return Checkin{}, err
	}
	if len(items) == 0 {
		return Checkin{}, apierr.ErrNotFound
	}
	return items[0], nil
}

func scanCheckins(rows pgx.Rows) ([]Checkin, error) {
	var items []Checkin
	for rows.Next() {
		var c Checkin
		if err := rows.Scan(
			&c.ID, &c.SpaceID, &c.Name, &c.Address, &c.Lat, &c.Lng, &c.PlaceID,
			&c.CheckedInAt, &c.CreatedAt,
			&c.Creator.ID, &c.Creator.Email, &c.Creator.DisplayName, &c.Creator.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	if items == nil {
		items = []Checkin{}
	}
	return items, rows.Err()
}

// DeleteCheckin soft-deletes a check-in. Only the author may delete it.
func (db *DB) DeleteCheckin(ctx context.Context, spaceID, id, actorID string) error {
	return db.deleteAuthored(ctx, "restaurant_checkins", "creator_user_id", spaceID, id, actorID)
}

// ListFavorites returns saved restaurants newest first.
func (db *DB) ListFavorites(ctx context.Context, spaceID string, limit int, before Page) ([]Favorite, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT f.id::text, f.space_id::text, f.name, f.address, f.lat, f.lng, f.place_id, f.note, f.created_at,
		       u.id::text, u.email::text, u.display_name, u.created_at
		FROM restaurant_favorites f
		JOIN users u ON u.id = f.creator_user_id
		WHERE f.space_id = $1 AND f.deleted_at IS NULL
		  AND ($2::bool = false OR (f.created_at, f.id) < ($3::timestamptz, $4::uuid))
		ORDER BY f.created_at DESC, f.id DESC
		LIMIT $5
	`, spaceID, before.Has, before.Time, before.ID, limit)
	if err != nil {
		return nil, fmt.Errorf("list favorites: %w", err)
	}
	defer rows.Close()
	return scanFavorites(rows)
}

// CreateFavorite saves a restaurant. A repeated place_id in the space conflicts.
func (db *DB) CreateFavorite(ctx context.Context, spaceID, creatorID, name string, address, placeID, note *string, lat, lng *float64) (Favorite, error) {
	var id string
	err := db.Pool.QueryRow(ctx, `
		INSERT INTO restaurant_favorites (
			space_id, creator_user_id, name, address, lat, lng, place_id, note
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id::text
	`, spaceID, creatorID, name, address, lat, lng, placeID, note).Scan(&id)
	if uniqueViolation(err) {
		return Favorite{}, apierr.ErrFavoriteExists
	}
	if err != nil {
		return Favorite{}, fmt.Errorf("create favorite: %w", err)
	}
	items, err := db.listFavoriteID(ctx, spaceID, id)
	if err != nil {
		return Favorite{}, err
	}
	if len(items) == 0 {
		return Favorite{}, apierr.ErrNotFound
	}
	return items[0], nil
}

func (db *DB) listFavoriteID(ctx context.Context, spaceID, id string) ([]Favorite, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT f.id::text, f.space_id::text, f.name, f.address, f.lat, f.lng, f.place_id, f.note, f.created_at,
		       u.id::text, u.email::text, u.display_name, u.created_at
		FROM restaurant_favorites f
		JOIN users u ON u.id = f.creator_user_id
		WHERE f.id = $1 AND f.space_id = $2 AND f.deleted_at IS NULL
	`, id, spaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanFavorites(rows)
}

func scanFavorites(rows pgx.Rows) ([]Favorite, error) {
	var items []Favorite
	for rows.Next() {
		var f Favorite
		if err := rows.Scan(
			&f.ID, &f.SpaceID, &f.Name, &f.Address, &f.Lat, &f.Lng, &f.PlaceID, &f.Note, &f.CreatedAt,
			&f.Creator.ID, &f.Creator.Email, &f.Creator.DisplayName, &f.Creator.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, f)
	}
	if items == nil {
		items = []Favorite{}
	}
	return items, rows.Err()
}

// DeleteFavorite soft-deletes a favorite. Only the author may delete it.
func (db *DB) DeleteFavorite(ctx context.Context, spaceID, id, actorID string) error {
	return db.deleteAuthored(ctx, "restaurant_favorites", "creator_user_id", spaceID, id, actorID)
}

func (db *DB) deleteAuthored(ctx context.Context, table, authorCol, spaceID, id, actorID string) error {
	switch table {
	case "restaurant_checkins", "restaurant_favorites", "anniversaries":
	default:
		return fmt.Errorf("unsupported table %s", table)
	}
	var author, gotSpace string
	var deleted bool
	query := fmt.Sprintf(`
		SELECT space_id::text, %s::text, deleted_at IS NOT NULL
		FROM %s WHERE id = $1
	`, authorCol, table)
	err := db.Pool.QueryRow(ctx, query, id).Scan(&gotSpace, &author, &deleted)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && (gotSpace != spaceID || deleted)) {
		return apierr.ErrNotFound
	}
	if err != nil {
		return err
	}
	if !authz.AuthorOnly(actorID, author) {
		return apierr.ErrForbidden
	}
	_, err = db.Pool.Exec(ctx, fmt.Sprintf(`UPDATE %s SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, table), id)
	return err
}
