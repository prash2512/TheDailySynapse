package syncer

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"dailysynapse/backend/internal/config"
	"dailysynapse/backend/internal/core"
	"dailysynapse/backend/internal/store"
	"dailysynapse/backend/pkg/readability"

	"github.com/mmcdole/gofeed"
)

type Syncer struct {
	store     store.Store
	extractor readability.Extractor
	fp        *gofeed.Parser
	feedChan  chan core.Feed
	cfg       *config.Config
	logger    *slog.Logger
}

func New(s store.Store, cfg *config.Config, logger *slog.Logger) *Syncer {
	fp := gofeed.NewParser()
	fp.Client = &http.Client{Timeout: cfg.HTTPTimeout}

	return &Syncer{
		store:     s,
		extractor: readability.NewExtractor(cfg.HTTPTimeout),
		fp:        fp,
		feedChan:  make(chan core.Feed, 100),
		cfg:       cfg,
		logger:    logger,
	}
}

func (s *Syncer) StartBackgroundWorkers(ctx context.Context) {
	var wg sync.WaitGroup

	for i := 0; i < s.cfg.SyncWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for feed := range s.feedChan {
				if err := s.syncFeed(ctx, feed); err != nil {
					s.logger.Error("sync failed",
						slog.Int("worker", workerID),
						slog.String("feed", feed.Name),
						slog.String("error", err.Error()),
					)
				}
			}
		}(i)
	}

	ticker := time.NewTicker(s.cfg.SyncInterval)
	cleanupTicker := time.NewTicker(24 * time.Hour)
	purgeTicker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	defer cleanupTicker.Stop()
	defer purgeTicker.Stop()

	s.TriggerSync(ctx)

	for {
		select {
		case <-ctx.Done():
			close(s.feedChan)
			wg.Wait()
			return
		case <-ticker.C:
			s.TriggerSync(ctx)
		case <-purgeTicker.C:
			s.purgeDeletedFeeds(ctx)
		case <-cleanupTicker.C:
			s.runCleanup(ctx)
		}
	}
}

func (s *Syncer) TriggerSync(ctx context.Context) error {
	feeds, err := s.store.GetFeedsToSync(ctx, s.cfg.SyncBatchSize)
	if err != nil {
		return fmt.Errorf("failed to fetch feeds: %w", err)
	}

	go func() {
		for _, feed := range feeds {
			select {
			case s.feedChan <- feed:
			default:
				s.logger.Warn("worker queue full", slog.String("feed", feed.Name))
			}
		}
	}()

	return nil
}

func (s *Syncer) purgeDeletedFeeds(ctx context.Context) {
	feeds, err := s.store.GetFeedsPendingDeletion(ctx)
	if err != nil {
		s.logger.Error("failed to fetch feeds pending deletion", slog.String("error", err.Error()))
		return
	}

	for _, feed := range feeds {
		s.logger.Info("purging feed", slog.String("name", feed.Name), slog.Int64("id", feed.ID))

		if err := s.store.DeleteArticlesByFeedID(ctx, feed.ID); err != nil {
			s.logger.Error("failed to delete articles", slog.Int64("feed_id", feed.ID), slog.String("error", err.Error()))
			continue
		}

		if err := s.store.DeleteFeed(ctx, feed.ID); err != nil {
			s.logger.Error("failed to delete feed", slog.Int64("feed_id", feed.ID), slog.String("error", err.Error()))
		} else {
			s.logger.Info("purged feed", slog.String("name", feed.Name))
		}
	}
}

func (s *Syncer) runCleanup(ctx context.Context) {
	s.purgeDeletedFeeds(ctx)

	horizon := time.Now().AddDate(0, 0, -s.cfg.RetentionDays)

	count, err := s.store.DeleteOldArticles(ctx, horizon)
	if err != nil {
		s.logger.Error("cleanup failed", slog.String("error", err.Error()))
		return
	}
	if count > 0 {
		s.logger.Info("cleanup complete", slog.Int64("deleted", count))
	}
}

func (s *Syncer) syncFeed(ctx context.Context, feed core.Feed) error {
	req, err := http.NewRequestWithContext(ctx, "GET", feed.URL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "TheDailySynapse/1.0")
	if feed.Etag != "" {
		req.Header.Set("If-None-Match", feed.Etag)
	}
	if feed.LastModified != "" {
		req.Header.Set("If-Modified-Since", feed.LastModified)
	}

	resp, err := s.fp.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		return s.store.UpdateFeedHeaders(ctx, feed.ID, feed.Etag, feed.LastModified, time.Now())
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned error: %s", resp.Status)
	}

	parsed, err := s.fp.Parse(resp.Body)
	if err != nil {
		return err
	}

	if parsed.Title != "" && feed.Name == "" {
		s.store.UpdateFeedName(ctx, feed.ID, parsed.Title)
	}

	newEtag := resp.Header.Get("ETag")
	newLastMod := resp.Header.Get("Last-Modified")

	horizon := time.Now().AddDate(0, 0, -s.cfg.ArticleHorizonDays)

	for _, item := range parsed.Items {
		published := item.PublishedParsed
		if published == nil {
			now := time.Now()
			published = &now
		}

		if published.Before(horizon) {
			continue
		}

		article := core.Article{
			FeedID:      feed.ID,
			Title:       item.Title,
			URL:         item.Link,
			PublishedAt: *published,
			Summary:     item.Description,
		}

		content, err := s.extractor.Extract(ctx, item.Link)
		if err != nil {
			s.logger.Warn("content extraction failed",
				slog.String("url", item.Link),
				slog.String("error", err.Error()),
			)
		} else {
			article.Content = content
		}

		if _, err := s.store.CreateArticle(ctx, article); err != nil {
			s.logger.Error("failed to save article",
				slog.String("title", item.Title),
				slog.String("error", err.Error()),
			)
		}
	}

	return s.store.UpdateFeedHeaders(ctx, feed.ID, newEtag, newLastMod, time.Now())
}
