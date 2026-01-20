package store

import (
	"context"
	"testing"
	"time"

	"dailysynapse/backend/internal/core"
)

func TestCreateFeed(t *testing.T) {
	q, cleanup := setupTestQueries(t)
	defer cleanup()

	ctx := context.Background()

	feed, err := q.CreateFeed(ctx, "https://example.com/feed", "Test Feed")
	if err != nil {
		t.Fatalf("CreateFeed() error = %v", err)
	}

	if feed.ID == 0 {
		t.Error("CreateFeed() returned ID = 0")
	}
	if feed.URL != "https://example.com/feed" {
		t.Errorf("CreateFeed() URL = %v, want 'https://example.com/feed'", feed.URL)
	}
	if feed.Name != "Test Feed" {
		t.Errorf("CreateFeed() Name = %v, want 'Test Feed'", feed.Name)
	}
	if feed.Status != "active" {
		t.Errorf("CreateFeed() Status = %v, want 'active'", feed.Status)
	}
}

func TestGetAllFeeds(t *testing.T) {
	q, cleanup := setupTestQueries(t)
	defer cleanup()

	ctx := context.Background()

	// Create multiple feeds
	feed1, err := q.CreateFeed(ctx, "https://example.com/feed1", "Feed 1")
	if err != nil {
		t.Fatalf("CreateFeed() error = %v", err)
	}

	feed2, err := q.CreateFeed(ctx, "https://example.com/feed2", "Feed 2")
	if err != nil {
		t.Fatalf("CreateFeed() error = %v", err)
	}

	feeds, err := q.GetAllFeeds(ctx)
	if err != nil {
		t.Fatalf("GetAllFeeds() error = %v", err)
	}

	if len(feeds) < 2 {
		t.Fatalf("GetAllFeeds() returned %d feeds, want at least 2", len(feeds))
	}

	// Verify feeds are present
	found1, found2 := false, false
	for _, f := range feeds {
		if f.ID == feed1.ID {
			found1 = true
		}
		if f.ID == feed2.ID {
			found2 = true
		}
	}

	if !found1 {
		t.Error("GetAllFeeds() did not return feed1")
	}
	if !found2 {
		t.Error("GetAllFeeds() did not return feed2")
	}
}

func TestGetFeedsToSync(t *testing.T) {
	q, cleanup := setupTestQueries(t)
	defer cleanup()

	ctx := context.Background()

	// Create feed
	feed, err := q.CreateFeed(ctx, "https://example.com/feed", "Test Feed")
	if err != nil {
		t.Fatalf("CreateFeed() error = %v", err)
	}

	feeds, err := q.GetFeedsToSync(ctx, 10)
	if err != nil {
		t.Fatalf("GetFeedsToSync() error = %v", err)
	}

	if len(feeds) == 0 {
		t.Error("GetFeedsToSync() returned 0 feeds, want at least 1")
	}

	found := false
	for _, f := range feeds {
		if f.ID == feed.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("GetFeedsToSync() did not return created feed")
	}
}

func TestUpdateFeedHeaders(t *testing.T) {
	q, cleanup := setupTestQueries(t)
	defer cleanup()

	ctx := context.Background()

	feed, err := q.CreateFeed(ctx, "https://example.com/feed", "Test Feed")
	if err != nil {
		t.Fatalf("CreateFeed() error = %v", err)
	}

	etag := "test-etag"
	lastModified := "Mon, 01 Jan 2024 00:00:00 GMT"
	lastSynced := time.Now()

	err = q.UpdateFeedHeaders(ctx, feed.ID, etag, lastModified, lastSynced)
	if err != nil {
		t.Fatalf("UpdateFeedHeaders() error = %v", err)
	}

	// Verify headers were updated
	feeds, err := q.GetAllFeeds(ctx)
	if err != nil {
		t.Fatalf("GetAllFeeds() error = %v", err)
	}

	var updatedFeed *core.Feed
	for _, f := range feeds {
		if f.ID == feed.ID {
			updatedFeed = &f
			break
		}
	}

	if updatedFeed == nil {
		t.Fatal("Feed not found after update")
	}
	if updatedFeed.Etag != etag {
		t.Errorf("Etag = %v, want %v", updatedFeed.Etag, etag)
	}
	if updatedFeed.LastModified != lastModified {
		t.Errorf("LastModified = %v, want %v", updatedFeed.LastModified, lastModified)
	}
}

func TestUpdateFeedName(t *testing.T) {
	q, cleanup := setupTestQueries(t)
	defer cleanup()

	ctx := context.Background()

	feed, err := q.CreateFeed(ctx, "https://example.com/feed", "Original Name")
	if err != nil {
		t.Fatalf("CreateFeed() error = %v", err)
	}

	err = q.UpdateFeedName(ctx, feed.ID, "Updated Name")
	if err != nil {
		t.Fatalf("UpdateFeedName() error = %v", err)
	}

	// Verify name was updated
	feeds, err := q.GetAllFeeds(ctx)
	if err != nil {
		t.Fatalf("GetAllFeeds() error = %v", err)
	}

	var updatedFeed *core.Feed
	for _, f := range feeds {
		if f.ID == feed.ID {
			updatedFeed = &f
			break
		}
	}

	if updatedFeed == nil {
		t.Fatal("Feed not found after update")
	}
	if updatedFeed.Name != "Updated Name" {
		t.Errorf("Name = %v, want 'Updated Name'", updatedFeed.Name)
	}
}

func TestMarkFeedForDeletion(t *testing.T) {
	q, cleanup := setupTestQueries(t)
	defer cleanup()

	ctx := context.Background()

	feed, err := q.CreateFeed(ctx, "https://example.com/feed", "Test Feed")
	if err != nil {
		t.Fatalf("CreateFeed() error = %v", err)
	}

	err = q.MarkFeedForDeletion(ctx, feed.ID)
	if err != nil {
		t.Fatalf("MarkFeedForDeletion() error = %v", err)
	}

	// Verify feed is marked for deletion
	pendingFeeds, err := q.GetFeedsPendingDeletion(ctx)
	if err != nil {
		t.Fatalf("GetFeedsPendingDeletion() error = %v", err)
	}

	found := false
	for _, f := range pendingFeeds {
		if f.ID == feed.ID {
			found = true
			if f.Status != "pending_deletion" {
				t.Errorf("Status = %v, want 'pending_deletion'", f.Status)
			}
			break
		}
	}
	if !found {
		t.Error("Feed not found in pending deletion list")
	}

	// Verify feed is not in GetAllFeeds (which excludes pending_deletion)
	allFeeds, err := q.GetAllFeeds(ctx)
	if err != nil {
		t.Fatalf("GetAllFeeds() error = %v", err)
	}

	for _, f := range allFeeds {
		if f.ID == feed.ID {
			t.Error("Feed found in GetAllFeeds() after marking for deletion")
		}
	}
}

func TestDeleteFeed(t *testing.T) {
	q, cleanup := setupTestQueries(t)
	defer cleanup()

	ctx := context.Background()

	feed, err := q.CreateFeed(ctx, "https://example.com/feed", "Test Feed")
	if err != nil {
		t.Fatalf("CreateFeed() error = %v", err)
	}

	err = q.DeleteFeed(ctx, feed.ID)
	if err != nil {
		t.Fatalf("DeleteFeed() error = %v", err)
	}

	// Verify feed is deleted
	feeds, err := q.GetAllFeeds(ctx)
	if err != nil {
		t.Fatalf("GetAllFeeds() error = %v", err)
	}

	for _, f := range feeds {
		if f.ID == feed.ID {
			t.Error("Feed found in GetAllFeeds() after deletion")
		}
	}
}

func TestDeleteFeed_NotFound(t *testing.T) {
	q, cleanup := setupTestQueries(t)
	defer cleanup()

	ctx := context.Background()

	err := q.DeleteFeed(ctx, 99999)
	if err != core.ErrNotFound {
		t.Errorf("DeleteFeed() error = %v, want ErrNotFound", err)
	}
}

