package store

import (
	"context"
	"fmt"

	"dailysynapse/backend/internal/core"
)

func (q *Queries) CreateArticle(ctx context.Context, article core.Article) (int64, error) {
	query := `
		INSERT INTO articles (feed_id, title, url, published_at, summary)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(url) DO NOTHING;
	`
	
	stmt, err := q.db.PrepareContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("preparing create article statement: %w", err)
	}
	defer stmt.Close()

	res, err := stmt.ExecContext(ctx,
		article.FeedID,
		article.Title,
		article.URL,
		article.PublishedAt,
		article.Summary,
	)
	if err != nil {
		return 0, fmt.Errorf("executing create article: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("getting last insert id: %w", err)
	}

	return id, nil
}
