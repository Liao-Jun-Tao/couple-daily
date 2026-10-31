// Package dbmigrate applies the embedded SQL migrations.
package dbmigrate

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"regexp"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
)

var upName = regexp.MustCompile(`^([0-9]{6}_[a-z0-9_]+)\.up\.sql$`)

// Apply runs each *.up.sql that is not yet recorded in schema_migrations.
// If the initial schema was applied by hand, that version is stamped and skipped.
func Apply(ctx context.Context, databaseURL string, files fs.FS) error {
	cfg, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		return err
	}
	cfg.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	if _, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version text PRIMARY KEY,
		applied_at timestamptz NOT NULL DEFAULT now()
	)`); err != nil {
		return fmt.Errorf("schema_migrations: %w", err)
	}

	entries, err := fs.ReadDir(files, ".")
	if err != nil {
		return err
	}
	var names []string
	for _, entry := range entries {
		if upName.MatchString(entry.Name()) {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		version := upName.FindStringSubmatch(name)[1]
		var applied bool
		if err := conn.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = `+quote(version)+`)`).Scan(&applied); err != nil {
			return err
		}
		if applied {
			continue
		}
		if version == "000001_init" {
			var exists bool
			if err := conn.QueryRow(ctx, `SELECT EXISTS(
				SELECT 1 FROM information_schema.tables
				WHERE table_schema = 'public' AND table_name = 'users'
			)`).Scan(&exists); err != nil {
				return err
			}
			if exists {
				if err := stamp(ctx, conn, version); err != nil {
					return err
				}
				log.Printf("migrate: stamped existing schema as %s", version)
				continue
			}
		}
		body, err := fs.ReadFile(files, name)
		if err != nil {
			return err
		}
		script := "BEGIN;\n" + string(body) + "\nINSERT INTO schema_migrations(version) VALUES (" + quote(version) + ");\nCOMMIT;"
		if _, err := conn.Exec(ctx, script); err != nil {
			return fmt.Errorf("apply %s: %w", name, err)
		}
		log.Printf("migrate: applied %s", version)
	}
	return nil
}

func stamp(ctx context.Context, conn *pgx.Conn, version string) error {
	_, err := conn.Exec(ctx, `INSERT INTO schema_migrations(version) VALUES (`+quote(version)+`) ON CONFLICT DO NOTHING`)
	return err
}

func quote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}
