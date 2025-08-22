package client

import (
	"context"
	"fmt"
	"log/slog"

	"google.golang.org/grpc"

	"github.com/Fancu1/phoenix-rss/internal/scheduler-service/models"
	"github.com/Fancu1/phoenix-rss/pkg/logger"
	feedpb "github.com/Fancu1/phoenix-rss/protos/gen/go/feed"
)

type FeedServiceClient struct {
	client feedpb.FeedServiceClient
	logger *slog.Logger
}

func NewFeedServiceClient(conn *grpc.ClientConn, logger *slog.Logger) *FeedServiceClient {
	return &FeedServiceClient{
		client: feedpb.NewFeedServiceClient(conn),
		logger: logger,
	}
}

// GetAllFeeds retrieve all feeds from the feed service
func (c *FeedServiceClient) GetAllFeeds(ctx context.Context) ([]*models.Feed, error) {
	log := logger.FromContext(ctx)
	log.Debug("fetching all feeds from feed service")

	req := &feedpb.ListAllFeedsRequest{}

	resp, err := c.client.ListAllFeeds(ctx, req)
	if err != nil {
		log.Error("failed to list all feeds", "error", err.Error())
		return nil, fmt.Errorf("failed to list all feeds: %w", err)
	}

	feeds := make([]*models.Feed, len(resp.Feeds))
	for i, pbFeed := range resp.Feeds {
		feeds[i] = &models.Feed{
			ID:          uint(pbFeed.Id),
			Title:       pbFeed.Title,
			URL:         pbFeed.Url,
			Description: pbFeed.Description,
		}
	}

	log.Debug("successfully fetched feeds", "count", len(feeds))
	return feeds, nil
}
