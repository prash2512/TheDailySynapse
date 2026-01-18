package syncer

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"dailysynapse/backend/internal/core"
	"dailysynapse/backend/internal/store"
	"dailysynapse/backend/pkg/readability"
	"github.com/mmcdole/gofeed"
)

type Syncer struct {
	store    *store.Queries
	fp       *gofeed.Parser
	feedChan chan core.Feed
}

func New(s *store.Queries) *Syncer {
	fp := gofeed.NewParser()
	fp.Client = &http.Client{
		Timeout: 10 * time.Second,
	}
	return &Syncer{
		store:    s,
		fp:       fp,
		feedChan: make(chan core.Feed, 100),
	}
}

func (s *Syncer) StartBackgroundWorkers(ctx context.Context, numWorkers int, interval time.Duration) {
	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for feed := range s.feedChan {
				if err := s.syncFeed(ctx, feed); err != nil {
					log.Printf("[Worker %d] Error syncing %s: %v", workerID, feed.Name, err)
				}
			}
		}(i)
	}

	ticker := time.NewTicker(interval)
	cleanupTicker := time.NewTicker(24 * time.Hour)
	purgeTicker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	defer cleanupTicker.Stop()
	defer purgeTicker.Stop()

	s.TriggerSync(ctx, 20)

	for {
		select {
		case <-ctx.Done():
			close(s.feedChan)
			wg.Wait()
			return
		case <-ticker.C:
			s.TriggerSync(ctx, 20)
		case <-purgeTicker.C:
			s.purgeDeletedFeeds(ctx)
		case <-cleanupTicker.C:
			s.runCleanup(ctx)
		}
	}
}

func (s *Syncer) TriggerSync(ctx context.Context, limit int) error {
	feeds, err := s.store.GetFeedsToSync(limit)
	if err != nil {
		return fmt.Errorf("failed to fetch feeds: %w", err)
	}

	go func() {
		for _, feed := range feeds {
			select {
			case s.feedChan <- feed:
			default:
				log.Printf("Worker queue full, skipping feed %s", feed.Name)
			}
		}
	}()

	return nil
}

func (s *Syncer) purgeDeletedFeeds(ctx context.Context) {
	feeds, err := s.store.GetFeedsPendingDeletion()
	if err != nil {
		log.Printf("Failed to fetch feeds pending deletion: %v", err)
		return
	}

	for _, feed := range feeds {
		log.Printf("Purging feed %s (ID: %d)...", feed.Name, feed.ID)

		if err := s.store.DeleteArticlesByFeedID(ctx, feed.ID); err != nil {
			log.Printf("Failed to delete articles for feed %d: %v", feed.ID, err)
			continue
		}

		if err := s.store.DeleteFeed(feed.ID); err != nil {
			log.Printf("Failed to delete feed %d: %v", feed.ID, err)
		} else {
			log.Printf("Successfully purged feed %s", feed.Name)
		}
	}
}

func (s *Syncer) runCleanup(ctx context.Context) {
	s.purgeDeletedFeeds(ctx)

	horizon := time.Now().AddDate(0, 0, -30)

	count, err := s.store.DeleteOldArticles(ctx, horizon)
	if err != nil {
		log.Printf("Cleanup failed: %v", err)
		return
	}
	if count > 0 {
		log.Printf("Cleanup: Deleted %d old articles", count)
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
		return s.store.UpdateFeedHeaders(feed.ID, feed.Etag, feed.LastModified, time.Now())
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned error: %s", resp.Status)
	}

	parsed, err := s.fp.Parse(resp.Body)
	if err != nil {
		return err
	}

	newEtag := resp.Header.Get("ETag")
	newLastMod := resp.Header.Get("Last-Modified")

	horizon := time.Now().AddDate(0, 0, -7)

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

		content, err := readability.Extract(item.Link)
		if err != nil {
			log.Printf("Failed to extract content for %s: %v", item.Link, err)
		} else {
			article.Content = content
		}

		if _, err := s.store.CreateArticle(ctx, article); err != nil {
			log.Printf("Failed to save article %s: %v", item.Title, err)
		}
	}

	return s.store.UpdateFeedHeaders(feed.ID, newEtag, newLastMod, time.Now())
}
