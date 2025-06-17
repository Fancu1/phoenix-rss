package repository

import (
	"sync"
	"time"

	"github.com/Fancu1/phoenix-rss/internal/models"
)

type FeedRepository struct {
	mu      sync.RWMutex
	feeds   map[uint]*models.Feed
	counter uint
}

func NewFeedRepository() *FeedRepository {
	return &FeedRepository{
		feeds:   make(map[uint]*models.Feed),
		counter: 0,
	}
}

func (r *FeedRepository) Create(feed *models.Feed) (*models.Feed, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.counter++
	feed.ID = r.counter
	feed.CreatedAt = time.Now()
	feed.UpdatedAt = time.Now()
	r.feeds[feed.ID] = feed

	return feed, nil
}
