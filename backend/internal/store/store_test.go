package store

import (
	"database/sql"
	"os"
	"testing"

	_ "modernc.org/sqlite"
)

// setupTestDB creates a temporary database for testing with schema
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	tmpfile, err := os.CreateTemp("", "test_*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpfile.Close()

	db, err := sql.Open("sqlite", tmpfile.Name())
	if err != nil {
		os.Remove(tmpfile.Name())
		t.Fatalf("failed to open test database: %v", err)
	}

	// Set up WAL mode
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		os.Remove(tmpfile.Name())
		t.Fatalf("failed to enable WAL mode: %v", err)
	}

	// Create schema directly (simplified for tests)
	schema := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER NOT NULL PRIMARY KEY,
			applied_at DATETIME NOT NULL
		);

		CREATE TABLE IF NOT EXISTS feeds (
			id INTEGER PRIMARY KEY,
			url TEXT UNIQUE NOT NULL,
			name TEXT,
			etag TEXT,
			last_modified TEXT,
			status TEXT DEFAULT 'active',
			last_synced_at DATETIME
		);

		CREATE TABLE IF NOT EXISTS articles (
			id INTEGER PRIMARY KEY,
			feed_id INTEGER REFERENCES feeds(id),
			title TEXT NOT NULL,
			url TEXT UNIQUE NOT NULL,
			published_at DATETIME,
			quality_rank INTEGER,
			summary TEXT,
			justification TEXT,
			is_read BOOLEAN DEFAULT 0,
			read_later BOOLEAN DEFAULT 0
		);

		CREATE TABLE IF NOT EXISTS tags (
			id INTEGER PRIMARY KEY,
			name TEXT UNIQUE NOT NULL
		);

		CREATE TABLE IF NOT EXISTS article_tags (
			article_id INTEGER NOT NULL REFERENCES articles(id),
			tag_id INTEGER NOT NULL REFERENCES tags(id),
			PRIMARY KEY (article_id, tag_id)
		);

		CREATE INDEX IF NOT EXISTS idx_articles_quality_rank ON articles (quality_rank DESC);
		CREATE INDEX IF NOT EXISTS idx_tags_name ON tags (name);
		CREATE INDEX IF NOT EXISTS idx_article_tags_tag_id ON article_tags (tag_id);
	`

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		os.Remove(tmpfile.Name())
		t.Fatalf("failed to create schema: %v", err)
	}

	db.SetMaxOpenConns(1)

	cleanup := func() {
		db.Close()
		os.Remove(tmpfile.Name())
	}

	return db, cleanup
}

// setupTestQueries creates a Queries instance with a test database
func setupTestQueries(t *testing.T) (*Queries, func()) {
	t.Helper()

	db, cleanup := setupTestDB(t)
	return NewQueries(db), cleanup
}

