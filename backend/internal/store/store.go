package store

import (
	"context"
	"time"

	"dailysynapse/backend/internal/core"
)

type FeedStore interface {
	CreateFeed(ctx context.Context, url, name string) (core.Feed, error)
	DeleteFeed(ctx context.Context, id int64) error
	MarkFeedForDeletion(ctx context.Context, id int64) error
	GetFeedsPendingDeletion(ctx context.Context) ([]core.Feed, error)
	GetAllFeeds(ctx context.Context) ([]core.Feed, error)
	GetFeedsToSync(ctx context.Context, limit int) ([]core.Feed, error)
	UpdateFeedHeaders(ctx context.Context, id int64, etag, lastModified string, lastSyncedAt time.Time) error
}

type ArticleStore interface {
	CreateArticle(ctx context.Context, article core.Article) (int64, error)
	DeleteOldArticles(ctx context.Context, horizon time.Time) (int64, error)
	DeleteArticlesByFeedID(ctx context.Context, feedID int64) error
	GetUnscoredArticles(ctx context.Context, limit int) ([]core.Article, error)
	UpdateArticleScore(ctx context.Context, id int64, rank int, summary, justification, model string, tags []string) error
	GetTopArticles(ctx context.Context, limit int) ([]core.Article, error)
	GetArticleByID(ctx context.Context, id int64) (*core.Article, error)
}

type Store interface {
	FeedStore
	ArticleStore
}
