package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
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

	migrationVersion := 1
	if currentVersion < migrationVersion {
		fmt.Println("Applying migration 1...")

		migrationFile := filepath.Join("backend", "scripts", "001_init.sql")
		script, err := os.ReadFile(migrationFile)
		if err != nil {
			return fmt.Errorf("could not read migration file %s: %w", migrationFile, err)
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("could not begin transaction: %w", err)
		}
		defer tx.Rollback()

		if _, err := tx.Exec(string(script)); err != nil {
			return fmt.Errorf("failed to execute migration script: %w", err)
		}

		_, err = tx.Exec("INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)", migrationVersion, time.Now())
		if err != nil {
			return fmt.Errorf("failed to update schema version: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("could not commit transaction: %w", err)
		}
		fmt.Println("Migration 1 applied successfully.")
	}

	return nil
}
