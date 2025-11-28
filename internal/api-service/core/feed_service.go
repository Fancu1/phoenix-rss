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

// BatchSubscribeResult represents the result of a single feed subscription attempt
type BatchSubscribeResult struct {
	URL     string
	Success bool
	Error   string
	Feed    *models.Feed
}

// FeedServiceInterface define the interface for feed operations
type FeedServiceInterface interface {
	ListAllFeeds(ctx context.Context) ([]*models.Feed, error)
	SubscribeToFeed(ctx context.Context, userID uint, url string) (*models.Feed, error)
	BatchSubscribeToFeeds(ctx context.Context, userID uint, urls []string) (results []BatchSubscribeResult, imported, failed int, err error)
	ListUserFeeds(ctx context.Context, userID uint) ([]*models.UserFeed, error)
	UnsubscribeFromFeed(ctx context.Context, userID, feedID uint) error
	IsUserSubscribed(ctx context.Context, userID, feedID uint) (bool, error)
	UpdateFeedCustomTitle(ctx context.Context, userID, feedID uint, customTitle *string) (*models.UserFeed, error)
}

// FeedServiceClient implement FeedServiceInterface using gRPC
type FeedServiceClient struct {
	client feedpb.FeedServiceClient
	conn   *grpc.ClientConn
}

// NewFeedServiceClient create a new gRPC client for Feed Service
func NewFeedServiceClient(address string) (*FeedServiceClient, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Feed Service at %s: %w", address, err)
	}

	client := feedpb.NewFeedServiceClient(conn)
	return &FeedServiceClient{
		client: client,
		conn:   conn,
	}, nil
}

// Close the gRPC connection
func (c *FeedServiceClient) Close() error {
	return c.conn.Close()
}

// ListAllFeeds return all feeds in the system
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

// SubscribeToFeed create a subscription between user and feed
func (c *FeedServiceClient) SubscribeToFeed(ctx context.Context, userID uint, url string) (*models.Feed, error) {
	req := &feedpb.SubscribeToFeedRequest{
		UserId:  uint64(userID),
		FeedUrl: url,
	}

	resp, err := c.client.SubscribeToFeed(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to feed: %w", err)
	}

	return c.convertPbToFeed(resp.Feed)
}

// ListUserFeeds return all feeds subscribed by a specific user with custom titles
func (c *FeedServiceClient) ListUserFeeds(ctx context.Context, userID uint) ([]*models.UserFeed, error) {
	req := &feedpb.ListUserFeedsRequest{
		UserId: uint64(userID),
	}

	resp, err := c.client.ListUserFeeds(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to list user feeds: %w", err)
	}

	feeds := make([]*models.UserFeed, len(resp.Feeds))
	for i, pbFeed := range resp.Feeds {
		feed, err := c.convertPbToUserFeed(pbFeed)
		if err != nil {
			return nil, fmt.Errorf("failed to convert feed %d: %w", pbFeed.Id, err)
		}
		feeds[i] = feed
	}

	return feeds, nil
}

// UnsubscribeFromFeed remove a subscription between user and feed
func (c *FeedServiceClient) UnsubscribeFromFeed(ctx context.Context, userID, feedID uint) error {
	req := &feedpb.UnsubscribeFromFeedRequest{
		UserId: uint64(userID),
		FeedId: uint64(feedID),
	}

	_, err := c.client.UnsubscribeFromFeed(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to unsubscribe from feed: %w", err)
	}

	return nil
}

// IsUserSubscribed check if a user is subscribed to a feed
func (c *FeedServiceClient) IsUserSubscribed(ctx context.Context, userID, feedID uint) (bool, error) {
	req := &feedpb.CheckSubscriptionRequest{
		UserId: uint64(userID),
		FeedId: uint64(feedID),
	}

	resp, err := c.client.CheckSubscription(ctx, req)
	if err != nil {
		return false, fmt.Errorf("failed to check subscription: %w", err)
	}

	return resp.IsSubscribed, nil
}

func (c *FeedServiceClient) BatchSubscribeToFeeds(ctx context.Context, userID uint, urls []string) ([]BatchSubscribeResult, int, int, error) {
	req := &feedpb.BatchSubscribeToFeedsRequest{
		UserId:   uint64(userID),
		FeedUrls: urls,
	}

	resp, err := c.client.BatchSubscribeToFeeds(ctx, req)
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
			feed, err := c.convertPbToFeed(pbResult.Feed)
			if err == nil {
				result.Feed = feed
			}
		}
		results[i] = result
	}

	return results, int(resp.Imported), int(resp.Failed), nil
}

// UpdateFeedCustomTitle updates the custom title for a user's feed subscription
func (c *FeedServiceClient) UpdateFeedCustomTitle(ctx context.Context, userID, feedID uint, customTitle *string) (*models.UserFeed, error) {
	req := &feedpb.UpdateSubscriptionRequest{
		UserId:      uint64(userID),
		FeedId:      uint64(feedID),
		CustomTitle: customTitle,
	}

	resp, err := c.client.UpdateSubscription(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update subscription: %w", err)
	}

	return c.convertPbToUserFeed(resp.Feed)
}

// Helper method for protobuf conversion
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

// Helper method for protobuf conversion with custom title
func (c *FeedServiceClient) convertPbToUserFeed(pbFeed *feedpb.Feed) (*models.UserFeed, error) {
	feed, err := c.convertPbToFeed(pbFeed)
	if err != nil {
		return nil, err
	}

	return &models.UserFeed{
		Feed:        *feed,
		CustomTitle: pbFeed.CustomTitle,
	}, nil
}
