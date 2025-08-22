package interfaces

import (
	"context"

	"github.com/Fancu1/phoenix-rss/internal/scheduler-service/models"
)

// FeedServiceClientInterface define the interface for feed service communication
type FeedServiceClientInterface interface {
	GetAllFeeds(ctx context.Context) ([]*models.Feed, error)
}

// ProducerInterface define the interface for event publishing
type ProducerInterface interface {
	PublishFeedFetch(ctx context.Context, feedID uint) error
}
