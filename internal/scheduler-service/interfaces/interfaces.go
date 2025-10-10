package interfaces

import (
	"context"

	"github.com/Fancu1/phoenix-rss/internal/events"
	"github.com/Fancu1/phoenix-rss/internal/scheduler-service/models"
)

// FeedServiceClientInterface define the interface for feed service communication
type FeedServiceClientInterface interface {
	GetAllFeeds(ctx context.Context) ([]*models.Feed, error)
	ListArticlesToCheck(ctx context.Context, timeRange models.ArticleCheckWindow, pageSize int, pageToken string) (*models.ArticleCheckPage, error)
}

// ProducerInterface define the interface for event publishing
type ProducerInterface interface {
	PublishFeedFetch(ctx context.Context, feedID uint) error
}

type ArticleCheckProducerInterface interface {
	PublishArticleCheck(ctx context.Context, event events.ArticleCheckEvent) error
}
