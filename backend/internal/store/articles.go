package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
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

func (q *Queries) GetUnscoredArticles(ctx context.Context, limit int) ([]core.Article, error) {
	query := `
		SELECT a.id, a.feed_id, a.title, a.url, a.published_at, ac.content
		FROM articles a
		JOIN article_content ac ON a.id = ac.article_id
		WHERE a.quality_rank IS NULL
		ORDER BY a.published_at DESC
		LIMIT ?
	`
	rows, err := q.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("querying unscored articles: %w", err)
	}
	defer rows.Close()

	var articles []core.Article
	for rows.Next() {
		var a core.Article
		if err := rows.Scan(&a.ID, &a.FeedID, &a.Title, &a.URL, &a.PublishedAt, &a.Content); err != nil {
			return nil, fmt.Errorf("scanning article: %w", err)
		}
		articles = append(articles, a)
	}
	return articles, nil
}

func (q *Queries) UpdateArticleScore(ctx context.Context, id int64, rank int, summary, justification, model string, tags []string) error {
	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	query := `
		UPDATE articles
		SET quality_rank = ?, summary = ?, justification = ?
		WHERE id = ?
	`
	if _, err := tx.ExecContext(ctx, query, rank, summary, justification, id); err != nil {
		return fmt.Errorf("updating article score: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `UPDATE article_content SET judge_model = ? WHERE article_id = ?`, model, id); err != nil {
		return fmt.Errorf("updating judge model: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM article_tags WHERE article_id = ?`, id); err != nil {
		return fmt.Errorf("clearing tags: %w", err)
	}

	insertTag := `INSERT INTO tags (name) VALUES (?) ON CONFLICT(name) DO UPDATE SET id=id RETURNING id`
	linkTag := `INSERT INTO article_tags (article_id, tag_id) VALUES (?, ?)`

	stmtTag, err := tx.PrepareContext(ctx, insertTag)
	if err != nil {
		return err
	}
	defer stmtTag.Close()

	stmtLink, err := tx.PrepareContext(ctx, linkTag)
	if err != nil {
		return err
	}
	defer stmtLink.Close()

	for _, tag := range tags {
		var tagID int64
		if err := stmtTag.QueryRowContext(ctx, tag).Scan(&tagID); err != nil {
			return fmt.Errorf("processing tag %s: %w", tag, err)
		}
		if _, err := stmtLink.ExecContext(ctx, id, tagID); err != nil {
			return fmt.Errorf("linking tag %s: %w", tag, err)
		}
	}

	return tx.Commit()
}

func (q *Queries) GetTopArticles(ctx context.Context, limit int) ([]core.Article, error) {
	query := `
		SELECT a.id, a.feed_id, a.title, a.url, a.published_at, 
		       a.quality_rank, a.summary, a.justification,
		       f.name as feed_name, a.is_read, a.read_later
		FROM articles a
		JOIN feeds f ON a.feed_id = f.id
		WHERE a.quality_rank IS NOT NULL AND a.is_read = 0
		ORDER BY a.quality_rank DESC, a.published_at DESC
		LIMIT ?
	`
	rows, err := q.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("querying top articles: %w", err)
	}
	defer rows.Close()

	var articles []core.Article
	for rows.Next() {
		var a core.Article
		var feedName string
		if err := rows.Scan(&a.ID, &a.FeedID, &a.Title, &a.URL, &a.PublishedAt,
			&a.QualityRank, &a.Summary, &a.Justification, &feedName, &a.IsRead, &a.ReadLater); err != nil {
			return nil, fmt.Errorf("scanning article: %w", err)
		}
		a.FeedName = feedName
		articles = append(articles, a)
	}
	return articles, nil
}

func (q *Queries) GetArticleByID(ctx context.Context, id int64) (*core.Article, error) {
	query := `
		SELECT a.id, a.feed_id, a.title, a.url, a.published_at,
		       a.quality_rank, a.summary, a.justification,
		       ac.content, f.name as feed_name, a.is_read, a.read_later
		FROM articles a
		JOIN article_content ac ON a.id = ac.article_id
		JOIN feeds f ON a.feed_id = f.id
		WHERE a.id = ?
	`
	var a core.Article
	var feedName string
	err := q.db.QueryRowContext(ctx, query, id).Scan(
		&a.ID, &a.FeedID, &a.Title, &a.URL, &a.PublishedAt,
		&a.QualityRank, &a.Summary, &a.Justification,
		&a.Content, &feedName, &a.IsRead, &a.ReadLater,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, core.ErrNotFound
		}
		return nil, fmt.Errorf("querying article: %w", err)
	}
	a.FeedName = feedName
	return &a, nil
}

func (q *Queries) GetArticlesByTags(ctx context.Context, tags []string, limit int) ([]core.Article, error) {
	if len(tags) == 0 {
		return q.GetTopArticles(ctx, limit)
	}

	placeholders := make([]string, len(tags))
	args := make([]any, len(tags)+1)
	for i, tag := range tags {
		placeholders[i] = "?"
		args[i] = tag
	}
	args[len(tags)] = limit

	query := fmt.Sprintf(`
		SELECT DISTINCT a.id, a.feed_id, a.title, a.url, a.published_at,
		       a.quality_rank, a.summary, a.justification,
		       f.name as feed_name
		FROM articles a
		JOIN feeds f ON a.feed_id = f.id
		JOIN article_tags at ON a.id = at.article_id
		JOIN tags t ON at.tag_id = t.id
		WHERE a.quality_rank IS NOT NULL
		  AND t.name IN (%s)
		ORDER BY a.quality_rank DESC, a.published_at DESC
		LIMIT ?
	`, strings.Join(placeholders, ","))

	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying articles by tags: %w", err)
	}
	defer rows.Close()

	var articles []core.Article
	for rows.Next() {
		var a core.Article
		var feedName string
		if err := rows.Scan(&a.ID, &a.FeedID, &a.Title, &a.URL, &a.PublishedAt,
			&a.QualityRank, &a.Summary, &a.Justification, &feedName); err != nil {
			return nil, fmt.Errorf("scanning article: %w", err)
		}
		a.FeedName = feedName
		articles = append(articles, a)
	}
	return articles, nil
}

func (q *Queries) GetAllTags(ctx context.Context) ([]core.TagCount, error) {
	query := `
		SELECT t.name, COUNT(at.article_id) as count
		FROM tags t
		JOIN article_tags at ON t.id = at.tag_id
		JOIN articles a ON at.article_id = a.id
		WHERE a.quality_rank IS NOT NULL
		GROUP BY t.name
		ORDER BY count DESC, t.name ASC
	`
	rows, err := q.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying tags: %w", err)
	}
	defer rows.Close()

	var tags []core.TagCount
	for rows.Next() {
		var t core.TagCount
		if err := rows.Scan(&t.Name, &t.Count); err != nil {
			return nil, fmt.Errorf("scanning tag: %w", err)
		}
		tags = append(tags, t)
	}
	return tags, nil
}

func (q *Queries) GetArticleTags(ctx context.Context, articleID int64) ([]string, error) {
	query := `
		SELECT t.name
		FROM tags t
		JOIN article_tags at ON t.id = at.tag_id
		WHERE at.article_id = ?
		ORDER BY t.name
	`
	rows, err := q.db.QueryContext(ctx, query, articleID)
	if err != nil {
		return nil, fmt.Errorf("querying article tags: %w", err)
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scanning tag: %w", err)
		}
		tags = append(tags, name)
	}
	return tags, nil
}

func (q *Queries) MarkArticleRead(ctx context.Context, id int64) error {
	_, err := q.db.ExecContext(ctx, `UPDATE articles SET is_read = 1 WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("marking article read: %w", err)
	}
	return nil
}

func (q *Queries) ToggleArticleSaved(ctx context.Context, id int64) (bool, error) {
	var currentState bool
	err := q.db.QueryRowContext(ctx, `SELECT read_later FROM articles WHERE id = ?`, id).Scan(&currentState)
	if err != nil {
		return false, fmt.Errorf("getting current state: %w", err)
	}

	newState := !currentState
	_, err = q.db.ExecContext(ctx, `UPDATE articles SET read_later = ? WHERE id = ?`, newState, id)
	if err != nil {
		return false, fmt.Errorf("toggling saved state: %w", err)
	}
	return newState, nil
}

func (q *Queries) GetSavedArticles(ctx context.Context) ([]core.Article, error) {
	query := `
		SELECT a.id, a.feed_id, a.title, a.url, a.published_at,
		       a.quality_rank, a.summary, a.justification,
		       f.name as feed_name
		FROM articles a
		JOIN feeds f ON a.feed_id = f.id
		WHERE a.read_later = 1
		ORDER BY a.published_at DESC
	`
	rows, err := q.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying saved articles: %w", err)
	}
	defer rows.Close()

	var articles []core.Article
	for rows.Next() {
		var a core.Article
		var feedName string
		if err := rows.Scan(&a.ID, &a.FeedID, &a.Title, &a.URL, &a.PublishedAt,
			&a.QualityRank, &a.Summary, &a.Justification, &feedName); err != nil {
			return nil, fmt.Errorf("scanning article: %w", err)
		}
		a.FeedName = feedName
		articles = append(articles, a)
	}
	return articles, nil
}

func (q *Queries) DeleteArticle(ctx context.Context, id int64) error {
	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM article_tags WHERE article_id = ?`, id); err != nil {
		return fmt.Errorf("deleting article tags: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM article_content WHERE article_id = ?`, id); err != nil {
		return fmt.Errorf("deleting article content: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM articles WHERE id = ?`, id); err != nil {
		return fmt.Errorf("deleting article: %w", err)
	}

	return tx.Commit()
}
