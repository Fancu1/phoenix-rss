package core

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/Fancu1/phoenix-rss/internal/models"
	feedpb "github.com/Fancu1/phoenix-rss/protos/gen/go/feed"
)

// ArticleServiceInterface define the interface for article operations
type ArticleServiceInterface interface {
	FetchAndSaveArticles(ctx context.Context, feedID uint) ([]*models.Article, error)
	ListArticlesByFeedID(ctx context.Context, userID, feedID uint) ([]*models.Article, error)
	TriggerFetch(ctx context.Context, userID, feedID uint) error
}

// ArticleServiceClient implement ArticleServiceInterface using gRPC
type ArticleServiceClient struct {
	client feedpb.FeedServiceClient
	conn   *grpc.ClientConn
}

// NewArticleServiceClient create a new gRPC client for Article operations
func NewArticleServiceClient(address string) (*ArticleServiceClient, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Feed Service at %s: %w", address, err)
	}

	client := feedpb.NewFeedServiceClient(conn)
	return &ArticleServiceClient{
		client: client,
		conn:   conn,
	}, nil
}

// Close the gRPC connection
func (c *ArticleServiceClient) Close() error {
	return c.conn.Close()
}

// FetchAndSaveArticles is not available via gRPC as it's an internal operation
func (c *ArticleServiceClient) FetchAndSaveArticles(ctx context.Context, feedID uint) ([]*models.Article, error) {
	return nil, fmt.Errorf("FetchAndSaveArticles not available via gRPC client")
}

// ListArticlesByFeedID return articles for a specific feed
func (c *ArticleServiceClient) ListArticlesByFeedID(ctx context.Context, userID, feedID uint) ([]*models.Article, error) {
	req := &feedpb.ListArticlesRequest{
		UserId: uint64(userID),
		FeedId: uint64(feedID),
	}

	resp, err := c.client.ListArticles(ctx, req)
	if err != nil {
		return nil, MapGRPCError(err)
	}

	articles := make([]*models.Article, len(resp.Articles))
	for i, pbArticle := range resp.Articles {
		article, err := c.convertPbToArticle(pbArticle)
		if err != nil {
			return nil, fmt.Errorf("failed to convert article %d: %w", pbArticle.Id, err)
		}
		articles[i] = article
	}

	return articles, nil
}

// TriggerFetch trigger a manual fetch for a specific feed
func (c *ArticleServiceClient) TriggerFetch(ctx context.Context, userID, feedID uint) error {
	req := &feedpb.TriggerFetchRequest{
		UserId: uint64(userID),
		FeedId: uint64(feedID),
	}

	_, err := c.client.TriggerFetch(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to trigger fetch: %w", err)
	}

	return nil
}

// Helper method for protobuf conversion
func (c *ArticleServiceClient) convertPbToArticle(pbArticle *feedpb.Article) (*models.Article, error) {
	createdAt, err := time.Parse(time.RFC3339, pbArticle.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse created_at: %w", err)
	}

	updatedAt, err := time.Parse(time.RFC3339, pbArticle.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse updated_at: %w", err)
	}

	publishedAt, err := time.Parse(time.RFC3339, pbArticle.PublishedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse published_at: %w", err)
	}

	return &models.Article{
		ID:          uint(pbArticle.Id),
		FeedID:      uint(pbArticle.FeedId),
		Title:       pbArticle.Title,
		URL:         pbArticle.Url,
		Description: pbArticle.Description,
		Content:     pbArticle.Content,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
		Read:        pbArticle.Read,
		Starred:     pbArticle.Starred,
		PublishedAt: publishedAt,
	}, nil
}
