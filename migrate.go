package main

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"sort"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func runMigrations(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT        PRIMARY KEY,
			applied_at TIMESTAMPTZ DEFAULT NOW()
		)
	`)
	if err != nil {
		return err
	}

	entries, err := fs.Glob(migrationsFS, "migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(entries)

	for _, path := range entries {
		version := path[len("migrations/"):]

		var count int
		db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = $1`, version).Scan(&count)
		if count > 0 {
			continue
		}

		content, err := migrationsFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}

		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("apply %s: %w", version, err)
		}

		db.Exec(`INSERT INTO schema_migrations (version) VALUES ($1)`, version)
		log.Printf("applied migration: %s", version)
	}
	return nil
}
