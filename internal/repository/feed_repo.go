package repository

import (
	"fmt"
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

func (r *FeedRepository) ListAll() ([]*models.Feed, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.feeds) == 0 {
		return []*models.Feed{}, nil
	}

	feeds := make([]*models.Feed, 0, len(r.feeds))
	for _, feed := range r.feeds {
		feeds = append(feeds, feed)
	}
	return feeds, nil
}

func (r *FeedRepository) GetByID(id uint) (*models.Feed, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	feed, ok := r.feeds[id]
	if !ok {
		return nil, fmt.Errorf("feed not found")
	}
	return feed, nil
}
