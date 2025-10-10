package models

import "time"

type ArticleToCheck struct {
	ArticleID        uint
	FeedID           uint
	URL              string
	PrevETag         string
	PrevLastModified string
}

type ArticleCheckPage struct {
	Items         []*ArticleToCheck
	NextPageToken string
}

type ArticleCheckWindow struct {
	PublishedSince    time.Time
	LastCheckedBefore time.Time
}
