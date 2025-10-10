package client

import (
	"context"
	"fmt"
	"log/slog"
	"time"

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

func (c *FeedServiceClient) ListArticlesToCheck(ctx context.Context, timeRange models.ArticleCheckWindow, pageSize int, pageToken string) (*models.ArticleCheckPage, error) {
	log := logger.FromContext(ctx)
	log.Debug("fetching articles to check",
		"published_since", timeRange.PublishedSince,
		"last_checked_before", timeRange.LastCheckedBefore,
		"page_size", pageSize,
	)

	if pageSize <= 0 {
		return nil, fmt.Errorf("page size must be positive")
	}

	req := &feedpb.ListArticlesToCheckRequest{
		PublishedSince:    timeRange.PublishedSince.UTC().Format(time.RFC3339),
		LastCheckedBefore: timeRange.LastCheckedBefore.UTC().Format(time.RFC3339),
		PageSize:          uint32(pageSize),
		PageToken:         pageToken,
	}

	resp, err := c.client.ListArticlesToCheck(ctx, req)
	if err != nil {
		log.Error("failed to list articles to check", "error", err)
		return nil, fmt.Errorf("failed to list articles to check: %w", err)
	}

	items := make([]*models.ArticleToCheck, len(resp.Items))
	for i, item := range resp.Items {
		items[i] = &models.ArticleToCheck{
			ArticleID:        uint(item.ArticleId),
			FeedID:           uint(item.FeedId),
			URL:              item.Url,
			PrevETag:         item.PrevEtag,
			PrevLastModified: item.PrevLastModified,
		}
	}

	log.Debug("received articles to check", "count", len(items), "has_next", resp.NextPageToken != "")

	return &models.ArticleCheckPage{
		Items:         items,
		NextPageToken: resp.NextPageToken,
	}, nil
}
