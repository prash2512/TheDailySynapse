package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

func Open(dsn string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(dsn), 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

func runMigrations(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER NOT NULL PRIMARY KEY,
			applied_at DATETIME NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("could not create schema_migrations table: %w", err)
	}

	var currentVersion int
	err = db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&currentVersion)
	if err != nil {
		return fmt.Errorf("could not get current schema version: %w", err)
	}

	files, err := filepath.Glob(filepath.Join("backend", "scripts", "*.sql"))
	if err != nil {
		return fmt.Errorf("could not list migration files: %w", err)
	}

	// We need to sort by VERSION number, not filename string
	// "10_x.sql" comes before "2_x.sql" in string sort, but after in number sort.
	type migration struct {
		version int
		file    string
	}
	var migrations []migration

	for _, file := range files {
		filename := filepath.Base(file)
		parts := strings.SplitN(filename, "_", 2)
		if len(parts) < 2 {
			continue
		}

		v, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		migrations = append(migrations, migration{version: v, file: file})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].version < migrations[j].version
	})

	for _, m := range migrations {
		if m.version > currentVersion {
			fmt.Printf("Applying migration %d (%s)...\n", m.version, filepath.Base(m.file))

			script, err := os.ReadFile(m.file)
			if err != nil {
				return fmt.Errorf("could not read migration file %s: %w", m.file, err)
			}

			tx, err := db.Begin()
			if err != nil {
				return fmt.Errorf("could not begin transaction: %w", err)
			}

			if _, err := tx.Exec(string(script)); err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to execute migration script %s: %w", m.file, err)
			}

			_, err = tx.Exec("INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)", m.version, time.Now())
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to update schema version: %w", err)
			}

			if err := tx.Commit(); err != nil {
				return fmt.Errorf("could not commit transaction: %w", err)
			}
			fmt.Printf("Migration %d applied successfully.\n", m.version)
		}
	}

	return nil
}
