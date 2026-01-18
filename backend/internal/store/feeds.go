package store

import (
	"context"
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

func (q *Queries) CreateFeed(ctx context.Context, url string, name string) (core.Feed, error) {
	query := `
		INSERT INTO feeds (url, name, status, last_synced_at)
		VALUES (?, ?, 'active', ?);
	`
	initialSyncTime := time.Time{}
	res, err := q.db.ExecContext(ctx, query, url, name, initialSyncTime)
	if err != nil {
		return core.Feed{}, fmt.Errorf("executing statement: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return core.Feed{}, fmt.Errorf("getting last insert ID: %w", err)
	}

	return core.Feed{
		ID:           id,
		URL:          url,
		Name:         name,
		Status:       "active",
		LastSyncedAt: initialSyncTime,
	}, nil
}

func (q *Queries) DeleteFeed(ctx context.Context, id int64) error {
	res, err := q.db.ExecContext(ctx, "DELETE FROM feeds WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("executing statement: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("getting rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return core.ErrNotFound
	}

	return nil
}

func (q *Queries) MarkFeedForDeletion(ctx context.Context, id int64) error {
	res, err := q.db.ExecContext(ctx, "UPDATE feeds SET status = 'pending_deletion' WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("executing statement: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("getting rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return core.ErrNotFound
	}

	return nil
}

func (q *Queries) GetFeedsPendingDeletion(ctx context.Context) ([]core.Feed, error) {
	rows, err := q.db.QueryContext(ctx, "SELECT id, url, name, status, etag, last_modified, last_synced_at FROM feeds WHERE status = 'pending_deletion'")
	if err != nil {
		return nil, fmt.Errorf("querying feeds pending deletion: %w", err)
	}
	defer rows.Close()

	return q.scanFeeds(rows)
}

func (q *Queries) GetAllFeeds(ctx context.Context) ([]core.Feed, error) {
	rows, err := q.db.QueryContext(ctx, "SELECT id, url, name, status, etag, last_modified, last_synced_at FROM feeds ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("querying feeds: %w", err)
	}
	defer rows.Close()

	return q.scanFeeds(rows)
}

func (q *Queries) GetFeedsToSync(ctx context.Context, limit int) ([]core.Feed, error) {
	rows, err := q.db.QueryContext(ctx, "SELECT id, url, name, status, etag, last_modified, last_synced_at FROM feeds WHERE status = 'active' ORDER BY last_synced_at ASC LIMIT ?", limit)
	if err != nil {
		return nil, fmt.Errorf("querying feeds to sync: %w", err)
	}
	defer rows.Close()

	return q.scanFeeds(rows)
}

func (q *Queries) scanFeeds(rows *sql.Rows) ([]core.Feed, error) {
	var feeds []core.Feed
	for rows.Next() {
		var feed core.Feed
		var etag, lastMod sql.NullString

		if err := rows.Scan(&feed.ID, &feed.URL, &feed.Name, &feed.Status, &etag, &lastMod, &feed.LastSyncedAt); err != nil {
			return nil, fmt.Errorf("scanning feed row: %w", err)
		}

		feed.Etag = etag.String
		feed.LastModified = lastMod.String
		feeds = append(feeds, feed)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating rows: %w", err)
	}

	return feeds, nil
}

func (q *Queries) UpdateFeedHeaders(ctx context.Context, id int64, etag, lastModified string, lastSyncedAt time.Time) error {
	_, err := q.db.ExecContext(ctx, `UPDATE feeds SET etag = ?, last_modified = ?, last_synced_at = ? WHERE id = ?`, etag, lastModified, lastSyncedAt, id)
	if err != nil {
		return fmt.Errorf("updating feed headers: %w", err)
	}
	return nil
}
