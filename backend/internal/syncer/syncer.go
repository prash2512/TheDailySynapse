package syncer

import (
	"context"
	"fmt"
	"log"
	"time"

	"dailysynapse/backend/internal/core"
	"dailysynapse/backend/internal/store"
	"github.com/mmcdole/gofeed"
)

type Syncer struct {
	store *store.Queries
	fp    *gofeed.Parser
}

func New(s *store.Queries) *Syncer {
	return &Syncer{
		store: s,
		fp:    gofeed.NewParser(),
	}
}

func (s *Syncer) SyncAll(ctx context.Context) error {
	feeds, err := s.store.GetAllFeeds()
	if err != nil {
		return fmt.Errorf("fetching feeds: %w", err)
	}

	for _, feed := range feeds {
		if err := s.syncFeed(ctx, feed); err != nil {
			log.Printf("error syncing feed %s: %v", feed.URL, err)
			continue
		}
	}

	return nil
}

func (s *Syncer) syncFeed(ctx context.Context, feed core.Feed) error {
	parsed, err := s.fp.ParseURLWithContext(feed.URL, ctx)
	if err != nil {
		return fmt.Errorf("parsing feed: %w", err)
	}

	for _, item := range parsed.Items {
		published := item.PublishedParsed
		if published == nil {
			now := time.Now()
			published = &now
		}

		article := core.Article{
			FeedID:      feed.ID,
			Title:       item.Title,
			URL:         item.Link,
			PublishedAt: *published,
			Summary:     item.Description,
		}

		if _, err := s.store.CreateArticle(ctx, article); err != nil {
			log.Printf("failed to save article %s: %v", item.Title, err)
		}
	}

	return nil
}
