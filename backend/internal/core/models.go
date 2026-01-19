package core

import "time"

type Feed struct {
	ID           int64
	URL          string
	Name         string
	Status       string
	Etag         string
	LastModified string
	LastSyncedAt time.Time
}

type Article struct {
	ID            int64
	FeedID        int64
	FeedName      string
	Title         string
	URL           string
	PublishedAt   time.Time
	Content       string
	QualityRank   int
	Summary       string
	Justification string
	JudgeModel    string
	IsRead        bool
	ReadLater     bool
}

type Tag struct {
	ID   int64
	Name string
}

type TagCount struct {
	Name  string
	Count int
}

type ArticleTag struct {
	ArticleID int64
	TagID     int64
}
