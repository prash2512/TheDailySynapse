package store

import (
	"database/sql"
	"fmt"
	"time"

	"dailysynapse/backend/internal/core"
)

type Queries struct {
	db *sql.DB
}

func NewQueries(db *sql.DB) *Queries {
	return &Queries{db: db}
}

func (q *Queries) CreateFeed(url string, name string) (core.Feed, error) {
	stmt, err := q.db.Prepare(`
		INSERT INTO feeds (url, name, last_synced_at)
		VALUES (?, ?, ?);
	`)
	if err != nil {
		return core.Feed{}, fmt.Errorf("could not prepare statement: %w", err)
	}
	defer stmt.Close()

	initialSyncTime := time.Time{}
	res, err := stmt.Exec(url, name, initialSyncTime)
	if err != nil {
		return core.Feed{}, fmt.Errorf("could not execute statement: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return core.Feed{}, fmt.Errorf("could not get last insert ID: %w", err)
	}

	feed := core.Feed{
		ID:           id,
		URL:          url,
		Name:         name,
		LastSyncedAt: initialSyncTime,
	}

	return feed, nil
}

func (q *Queries) DeleteFeed(id int64) error {
	stmt, err := q.db.Prepare("DELETE FROM feeds WHERE id = ?;")
	if err != nil {
		return fmt.Errorf("could not prepare statement: %w", err)
	}
	defer stmt.Close()

	res, err := stmt.Exec(id)
	if err != nil {
		return fmt.Errorf("could not execute statement: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("no feed with id %d found", id)
	}

	return nil
}

func (q *Queries) GetAllFeeds() ([]core.Feed, error) {
	rows, err := q.db.Query("SELECT id, url, name, etag, last_modified, last_synced_at FROM feeds ORDER BY name;")
	if err != nil {
		return nil, fmt.Errorf("could not query feeds: %w", err)
	}
	defer rows.Close()

	var feeds []core.Feed
	for rows.Next() {
		var feed core.Feed
		var etag, lastMod sql.NullString
		
		if err := rows.Scan(&feed.ID, &feed.URL, &feed.Name, &etag, &lastMod, &feed.LastSyncedAt); err != nil {
			return nil, fmt.Errorf("could not scan feed row: %w", err)
		}
		
		feed.Etag = etag.String
		feed.LastModified = lastMod.String
		feeds = append(feeds, feed)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during rows iteration: %w", err)
	}

	return feeds, nil
}

func (q *Queries) UpdateFeedHeaders(id int64, etag, lastModified string, lastSyncedAt time.Time) error {
	query := `UPDATE feeds SET etag = ?, last_modified = ?, last_synced_at = ? WHERE id = ?`
	_, err := q.db.Exec(query, etag, lastModified, lastSyncedAt, id)
	if err != nil {
		return fmt.Errorf("failed to update feed headers: %w", err)
	}
	return nil
}