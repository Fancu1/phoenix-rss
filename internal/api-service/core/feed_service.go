package core

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/Fancu1/phoenix-rss/internal/feed-service/models"
	feedpb "github.com/Fancu1/phoenix-rss/protos/gen/go/feed"
)

type BatchSubscribeResult struct {
	URL     string
	Success bool
	Error   string
	Feed    *models.Feed
}

type FeedServiceInterface interface {
	ListAllFeeds(ctx context.Context) ([]*models.Feed, error)
	SubscribeToFeed(ctx context.Context, userID uint, url string) (*models.Feed, error)
	BatchSubscribeToFeeds(ctx context.Context, userID uint, urls []string) (results []BatchSubscribeResult, imported, failed int, err error)
}

type FeedServiceClient struct {
	client feedpb.FeedServiceClient
	conn   *grpc.ClientConn
}

func NewFeedServiceClient(address string) (*FeedServiceClient, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Feed Service at %s: %w", address, err)
	}

	return &FeedServiceClient{
		client: feedpb.NewFeedServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *FeedServiceClient) Close() error {
	return c.conn.Close()
}

func (c *FeedServiceClient) ListAllFeeds(ctx context.Context) ([]*models.Feed, error) {
	resp, err := c.client.ListAllFeeds(ctx, &feedpb.ListAllFeedsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to list all feeds: %w", err)
	}

	feeds := make([]*models.Feed, len(resp.Feeds))
	for i, pbFeed := range resp.Feeds {
		feed, err := c.convertPbToFeed(pbFeed)
		if err != nil {
			return nil, fmt.Errorf("failed to convert feed %d: %w", pbFeed.Id, err)
		}
		feeds[i] = feed
	}

	return feeds, nil
}

func (c *FeedServiceClient) SubscribeToFeed(ctx context.Context, userID uint, url string) (*models.Feed, error) {
	resp, err := c.client.SubscribeToFeed(ctx, &feedpb.SubscribeToFeedRequest{
		UserId:  uint64(userID),
		FeedUrl: url,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to feed: %w", err)
	}

	return c.convertPbToFeed(resp.Feed)
}

func (c *FeedServiceClient) BatchSubscribeToFeeds(ctx context.Context, userID uint, urls []string) ([]BatchSubscribeResult, int, int, error) {
	resp, err := c.client.BatchSubscribeToFeeds(ctx, &feedpb.BatchSubscribeToFeedsRequest{
		UserId:   uint64(userID),
		FeedUrls: urls,
	})
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to batch subscribe to feeds: %w", err)
	}

	results := make([]BatchSubscribeResult, len(resp.Results))
	for i, pbResult := range resp.Results {
		result := BatchSubscribeResult{
			URL:     pbResult.Url,
			Success: pbResult.Success,
			Error:   pbResult.Error,
		}
		if pbResult.Feed != nil {
			if feed, err := c.convertPbToFeed(pbResult.Feed); err == nil {
				result.Feed = feed
			}
		}
		results[i] = result
	}

	return results, int(resp.Imported), int(resp.Failed), nil
}

func (c *FeedServiceClient) convertPbToFeed(pbFeed *feedpb.Feed) (*models.Feed, error) {
	createdAt, err := time.Parse(time.RFC3339, pbFeed.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse created_at: %w", err)
	}

	updatedAt, err := time.Parse(time.RFC3339, pbFeed.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse updated_at: %w", err)
	}

	return &models.Feed{
		ID:          uint(pbFeed.Id),
		Title:       pbFeed.Title,
		URL:         pbFeed.Url,
		Description: pbFeed.Description,
		Status:      models.FeedStatus(pbFeed.Status),
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}, nil
}
