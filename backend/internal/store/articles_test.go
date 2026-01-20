package store

import (
	"context"
	"testing"
	"time"

	"dailysynapse/backend/internal/core"
)

func TestCreateArticle(t *testing.T) {
	q, cleanup := setupTestQueries(t)
	defer cleanup()

	ctx := context.Background()

	// Create a feed first
	feed, err := q.CreateFeed(ctx, "https://example.com/feed", "Test Feed")
	if err != nil {
		t.Fatalf("CreateFeed() error = %v", err)
	}

	article := core.Article{
		FeedID:      feed.ID,
		Title:       "Test Article",
		URL:         "https://example.com/article1",
		PublishedAt: time.Now(),
		Summary:     "This is a test article summary that is long enough to pass validation",
	}

	id, err := q.CreateArticle(ctx, article)
	if err != nil {
		t.Fatalf("CreateArticle() error = %v", err)
	}
	if id == 0 {
		t.Error("CreateArticle() returned 0, want non-zero ID")
	}

	// Verify article was created
	created, err := q.GetArticleByID(ctx, id)
	if err != nil {
		t.Fatalf("GetArticleByID() error = %v", err)
	}
	if created.Title != article.Title {
		t.Errorf("GetArticleByID() Title = %v, want %v", created.Title, article.Title)
	}
	if created.URL != article.URL {
		t.Errorf("GetArticleByID() URL = %v, want %v", created.URL, article.URL)
	}
}

func TestCreateArticle_DuplicateURL(t *testing.T) {
	q, cleanup := setupTestQueries(t)
	defer cleanup()

	ctx := context.Background()

	feed, err := q.CreateFeed(ctx, "https://example.com/feed", "Test Feed")
	if err != nil {
		t.Fatalf("CreateFeed() error = %v", err)
	}

	article := core.Article{
		FeedID:      feed.ID,
		Title:       "Test Article",
		URL:         "https://example.com/article1",
		PublishedAt: time.Now(),
		Summary:     "This is a test article summary that is long enough to pass validation",
	}

	id1, err := q.CreateArticle(ctx, article)
	if err != nil {
		t.Fatalf("CreateArticle() error = %v", err)
	}

	// Try to create same article again
	id2, err := q.CreateArticle(ctx, article)
	if err != nil {
		t.Fatalf("CreateArticle() error = %v", err)
	}
	if id2 != 0 {
		t.Errorf("CreateArticle() returned %v, want 0 (duplicate)", id2)
	}
	if id1 == 0 {
		t.Error("First CreateArticle() returned 0")
	}
}

func TestGetUnscoredArticles(t *testing.T) {
	q, cleanup := setupTestQueries(t)
	defer cleanup()

	ctx := context.Background()

	feed, err := q.CreateFeed(ctx, "https://example.com/feed", "Test Feed")
	if err != nil {
		t.Fatalf("CreateFeed() error = %v", err)
	}

	// Create unscored article
	article := core.Article{
		FeedID:      feed.ID,
		Title:       "Unscored Article",
		URL:         "https://example.com/unscored",
		PublishedAt: time.Now(),
		Summary:     "This is a test article summary that is long enough to pass validation",
	}
	_, err = q.CreateArticle(ctx, article)
	if err != nil {
		t.Fatalf("CreateArticle() error = %v", err)
	}

	// Create scored article
	scoredArticle := core.Article{
		FeedID:      feed.ID,
		Title:       "Scored Article",
		URL:         "https://example.com/scored",
		PublishedAt: time.Now(),
		Summary:     "This is a test article summary that is long enough to pass validation",
	}
	scoredID, err := q.CreateArticle(ctx, scoredArticle)
	if err != nil {
		t.Fatalf("CreateArticle() error = %v", err)
	}
	err = q.UpdateArticleScore(ctx, scoredID, 80, "Summary", "Justification", "model", []string{"test"})
	if err != nil {
		t.Fatalf("UpdateArticleScore() error = %v", err)
	}

	// Get unscored articles
	articles, err := q.GetUnscoredArticles(ctx, 10)
	if err != nil {
		t.Fatalf("GetUnscoredArticles() error = %v", err)
	}

	if len(articles) == 0 {
		t.Error("GetUnscoredArticles() returned 0 articles, want at least 1")
	}

	found := false
	for _, a := range articles {
		if a.Title == "Unscored Article" {
			found = true
			break
		}
	}
	if !found {
		t.Error("GetUnscoredArticles() did not return unscored article")
	}

	// Verify scored article is not included
	for _, a := range articles {
		if a.Title == "Scored Article" {
			t.Error("GetUnscoredArticles() returned scored article")
		}
	}
}

func TestUpdateArticleScore(t *testing.T) {
	q, cleanup := setupTestQueries(t)
	defer cleanup()

	ctx := context.Background()

	feed, err := q.CreateFeed(ctx, "https://example.com/feed", "Test Feed")
	if err != nil {
		t.Fatalf("CreateFeed() error = %v", err)
	}

	article := core.Article{
		FeedID:      feed.ID,
		Title:       "Test Article",
		URL:         "https://example.com/article1",
		PublishedAt: time.Now(),
		Summary:     "This is a test article summary that is long enough to pass validation",
	}
	id, err := q.CreateArticle(ctx, article)
	if err != nil {
		t.Fatalf("CreateArticle() error = %v", err)
	}

	tags := []string{"Go", "Performance", "Testing"}
	err = q.UpdateArticleScore(ctx, id, 85, "Updated summary", "Great article", "gemini-2.5-pro", tags)
	if err != nil {
		t.Fatalf("UpdateArticleScore() error = %v", err)
	}

	// Verify score was updated
	updated, err := q.GetArticleByID(ctx, id)
	if err != nil {
		t.Fatalf("GetArticleByID() error = %v", err)
	}
	if updated.QualityRank != 85 {
		t.Errorf("QualityRank = %v, want 85", updated.QualityRank)
	}
	if updated.Summary != "Updated summary" {
		t.Errorf("Summary = %v, want 'Updated summary'", updated.Summary)
	}
	if updated.Justification != "Great article" {
		t.Errorf("Justification = %v, want 'Great article'", updated.Justification)
	}

	// Verify tags were created
	articleTags, err := q.GetArticleTags(ctx, id)
	if err != nil {
		t.Fatalf("GetArticleTags() error = %v", err)
	}
	if len(articleTags) != len(tags) {
		t.Errorf("GetArticleTags() returned %d tags, want %d", len(articleTags), len(tags))
	}
}

func TestMarkArticleRead(t *testing.T) {
	q, cleanup := setupTestQueries(t)
	defer cleanup()

	ctx := context.Background()

	feed, err := q.CreateFeed(ctx, "https://example.com/feed", "Test Feed")
	if err != nil {
		t.Fatalf("CreateFeed() error = %v", err)
	}

	article := core.Article{
		FeedID:      feed.ID,
		Title:       "Test Article",
		URL:         "https://example.com/article1",
		PublishedAt: time.Now(),
		Summary:     "This is a test article summary that is long enough to pass validation",
	}
	id, err := q.CreateArticle(ctx, article)
	if err != nil {
		t.Fatalf("CreateArticle() error = %v", err)
	}

	err = q.MarkArticleRead(ctx, id)
	if err != nil {
		t.Fatalf("MarkArticleRead() error = %v", err)
	}

	// Verify article is marked as read
	read, err := q.GetArticleByID(ctx, id)
	if err != nil {
		t.Fatalf("GetArticleByID() error = %v", err)
	}
	if !read.IsRead {
		t.Error("IsRead = false, want true")
	}
}

func TestMarkArticleUnread(t *testing.T) {
	q, cleanup := setupTestQueries(t)
	defer cleanup()

	ctx := context.Background()

	feed, err := q.CreateFeed(ctx, "https://example.com/feed", "Test Feed")
	if err != nil {
		t.Fatalf("CreateFeed() error = %v", err)
	}

	article := core.Article{
		FeedID:      feed.ID,
		Title:       "Test Article",
		URL:         "https://example.com/article1",
		PublishedAt: time.Now(),
		Summary:     "This is a test article summary that is long enough to pass validation",
	}
	id, err := q.CreateArticle(ctx, article)
	if err != nil {
		t.Fatalf("CreateArticle() error = %v", err)
	}

	// Mark as read first
	err = q.MarkArticleRead(ctx, id)
	if err != nil {
		t.Fatalf("MarkArticleRead() error = %v", err)
	}

	// Mark as unread
	err = q.MarkArticleUnread(ctx, id)
	if err != nil {
		t.Fatalf("MarkArticleUnread() error = %v", err)
	}

	// Verify article is marked as unread
	unread, err := q.GetArticleByID(ctx, id)
	if err != nil {
		t.Fatalf("GetArticleByID() error = %v", err)
	}
	if unread.IsRead {
		t.Error("IsRead = true, want false")
	}
}

func TestToggleArticleSaved(t *testing.T) {
	q, cleanup := setupTestQueries(t)
	defer cleanup()

	ctx := context.Background()

	feed, err := q.CreateFeed(ctx, "https://example.com/feed", "Test Feed")
	if err != nil {
		t.Fatalf("CreateFeed() error = %v", err)
	}

	article := core.Article{
		FeedID:      feed.ID,
		Title:       "Test Article",
		URL:         "https://example.com/article1",
		PublishedAt: time.Now(),
		Summary:     "This is a test article summary that is long enough to pass validation",
	}
	id, err := q.CreateArticle(ctx, article)
	if err != nil {
		t.Fatalf("CreateArticle() error = %v", err)
	}

	// Toggle to saved
	saved, err := q.ToggleArticleSaved(ctx, id)
	if err != nil {
		t.Fatalf("ToggleArticleSaved() error = %v", err)
	}
	if !saved {
		t.Error("ToggleArticleSaved() returned false, want true")
	}

	// Verify article is saved
	article2, err := q.GetArticleByID(ctx, id)
	if err != nil {
		t.Fatalf("GetArticleByID() error = %v", err)
	}
	if !article2.ReadLater {
		t.Error("ReadLater = false, want true")
	}

	// Toggle to unsaved
	saved2, err := q.ToggleArticleSaved(ctx, id)
	if err != nil {
		t.Fatalf("ToggleArticleSaved() error = %v", err)
	}
	if saved2 {
		t.Error("ToggleArticleSaved() returned true, want false")
	}

	// Verify article is not saved
	article3, err := q.GetArticleByID(ctx, id)
	if err != nil {
		t.Fatalf("GetArticleByID() error = %v", err)
	}
	if article3.ReadLater {
		t.Error("ReadLater = true, want false")
	}
}

func TestDeleteArticle(t *testing.T) {
	q, cleanup := setupTestQueries(t)
	defer cleanup()

	ctx := context.Background()

	feed, err := q.CreateFeed(ctx, "https://example.com/feed", "Test Feed")
	if err != nil {
		t.Fatalf("CreateFeed() error = %v", err)
	}

	article := core.Article{
		FeedID:      feed.ID,
		Title:       "Test Article",
		URL:         "https://example.com/article1",
		PublishedAt: time.Now(),
		Summary:     "This is a test article summary that is long enough to pass validation",
	}
	id, err := q.CreateArticle(ctx, article)
	if err != nil {
		t.Fatalf("CreateArticle() error = %v", err)
	}

	// Add tags
	err = q.UpdateArticleScore(ctx, id, 80, "Summary", "Justification", "model", []string{"test"})
	if err != nil {
		t.Fatalf("UpdateArticleScore() error = %v", err)
	}

	// Delete article
	err = q.DeleteArticle(ctx, id)
	if err != nil {
		t.Fatalf("DeleteArticle() error = %v", err)
	}

	// Verify article is deleted
	_, err = q.GetArticleByID(ctx, id)
	if err != core.ErrNotFound {
		t.Errorf("GetArticleByID() error = %v, want ErrNotFound", err)
	}
}

func TestGetTopArticles_Ordering(t *testing.T) {
	q, cleanup := setupTestQueries(t)
	defer cleanup()

	ctx := context.Background()

	feed, err := q.CreateFeed(ctx, "https://example.com/feed", "Test Feed")
	if err != nil {
		t.Fatalf("CreateFeed() error = %v", err)
	}

	// Create articles with different scores and read status
	articles := []struct {
		title      string
		url        string
		score      int
		isRead     bool
		published  time.Time
	}{
		{"High Score Unread", "https://example.com/1", 90, false, time.Now()},
		{"Low Score Unread", "https://example.com/2", 60, false, time.Now().Add(-time.Hour)},
		{"High Score Read", "https://example.com/3", 85, true, time.Now().Add(-2 * time.Hour)},
		{"Low Score Read", "https://example.com/4", 55, true, time.Now().Add(-3 * time.Hour)},
	}

	for i, a := range articles {
		article := core.Article{
			FeedID:      feed.ID,
			Title:       a.title,
			URL:         a.url,
			PublishedAt: a.published,
			Summary:     "This is a test article summary that is long enough to pass validation",
		}
		id, err := q.CreateArticle(ctx, article)
		if err != nil {
			t.Fatalf("CreateArticle() error = %v", err)
		}

		err = q.UpdateArticleScore(ctx, id, a.score, "Summary", "Justification", "model", []string{})
		if err != nil {
			t.Fatalf("UpdateArticleScore() error = %v", err)
		}

		if a.isRead {
			err = q.MarkArticleRead(ctx, id)
			if err != nil {
				t.Fatalf("MarkArticleRead() error = %v", err)
			}
		}
		articles[i].title = a.title // Keep for later reference
	}

	// Get top articles
	topArticles, _, err := q.GetTopArticles(ctx, 10, 0)
	if err != nil {
		t.Fatalf("GetTopArticles() error = %v", err)
	}

	if len(topArticles) != 4 {
		t.Fatalf("GetTopArticles() returned %d articles, want 4", len(topArticles))
	}

	// Verify ordering: unread articles first, then by score
	// First should be "High Score Unread" (unread, score 90)
	if topArticles[0].Title != "High Score Unread" {
		t.Errorf("GetTopArticles()[0].Title = %v, want 'High Score Unread'", topArticles[0].Title)
	}
	// Second should be "Low Score Unread" (unread, score 60)
	if topArticles[1].Title != "Low Score Unread" {
		t.Errorf("GetTopArticles()[1].Title = %v, want 'Low Score Unread'", topArticles[1].Title)
	}
	// Then read articles
	if topArticles[2].Title != "High Score Read" {
		t.Errorf("GetTopArticles()[2].Title = %v, want 'High Score Read'", topArticles[2].Title)
	}
}

