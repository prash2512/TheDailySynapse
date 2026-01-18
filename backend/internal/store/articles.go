package store

import (
	"context"
	"fmt"
	"time"

	"dailysynapse/backend/internal/core"
)

func (q *Queries) CreateArticle(ctx context.Context, article core.Article) (int64, error) {
	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	queryMeta := `
		INSERT INTO articles (feed_id, title, url, published_at, summary)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(url) DO NOTHING;
	`
	res, err := tx.ExecContext(ctx, queryMeta,
		article.FeedID,
		article.Title,
		article.URL,
		article.PublishedAt,
		article.Summary,
	)
	if err != nil {
		return 0, fmt.Errorf("executing create article meta: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("getting rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return 0, nil
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("getting last insert id: %w", err)
	}

	queryContent := `INSERT INTO article_content (article_id, content, judge_model) VALUES (?, ?, ?)`
	_, err = tx.ExecContext(ctx, queryContent, id, article.Content, article.JudgeModel)
	if err != nil {
		return 0, fmt.Errorf("executing create article content: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("committing transaction: %w", err)
	}

	return id, nil
}

func (q *Queries) DeleteOldArticles(ctx context.Context, horizon time.Time) (int64, error) {
	query := `DELETE FROM articles WHERE published_at < ? AND read_later = 0`

	res, err := q.db.ExecContext(ctx, query, horizon)
	if err != nil {
		return 0, fmt.Errorf("executing delete old articles: %w", err)
	}

	return res.RowsAffected()
}

func (q *Queries) DeleteArticlesByFeedID(ctx context.Context, feedID int64) error {
	query := `DELETE FROM articles WHERE feed_id = ?`
	_, err := q.db.ExecContext(ctx, query, feedID)
	if err != nil {
		return fmt.Errorf("executing delete articles by feed id: %w", err)
	}
	return nil
}
