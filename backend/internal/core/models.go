package core

import "time"

type Feed struct {
	ID           int64
	URL          string
	Name         string
	Etag         string
	LastModified string
	LastSyncedAt time.Time
}

type Article struct {
	ID          int64
	FeedID      int64
	Title       string
	URL         string
	PublishedAt time.Time
	QualityRank int
	Summary     string
	IsRead      bool
	ReadLater   bool
}

type Tag struct {
	ID   int64
	Name string
}

type ArticleTag struct {
	ArticleID int64
	TagID     int64
}