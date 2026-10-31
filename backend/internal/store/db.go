// Package store is the PostgreSQL persistence layer.
package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNoSpace means the user is not a member of a couple space.
var ErrNoSpace = errors.New("no space")

// ErrInviteCodeConflict means a generated invite code collided.
var ErrInviteCodeConflict = errors.New("invite code conflict")

// DB is a connection pool.
type DB struct {
	Pool *pgxpool.Pool
}

type queryer interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Connect opens a pool and checks that the database answers.
func Connect(ctx context.Context, databaseURL string) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 10
	cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, "SET TIME ZONE 'UTC'")
		return err
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, err
	}
	return &DB{Pool: pool}, nil
}

// Close releases the pool.
func (db *DB) Close() {
	if db != nil && db.Pool != nil {
		db.Pool.Close()
	}
}

func (db *DB) withTx(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func uniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// Page is a cursor position. Has is false on the first page; Time and ID stay populated so SQL casts stay typed.
type Page struct {
	Has  bool
	Time time.Time
	ID   string
}

// PageFrom builds a page bound. A missing cursor still carries a dummy timestamp and UUID.
func PageFrom(t *time.Time, id string) Page {
	if t == nil || id == "" {
		return Page{
			Time: time.Unix(0, 0).UTC(),
			ID:   "00000000-0000-0000-0000-000000000000",
		}
	}
	return Page{Has: true, Time: t.UTC(), ID: id}
}
