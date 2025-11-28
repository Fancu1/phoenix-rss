package core

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	feedpb "github.com/Fancu1/phoenix-rss/protos/gen/go/feed"
)

type ArticleServiceInterface interface {
	TriggerFetch(ctx context.Context, userID, feedID uint) error
}

type ArticleServiceClient struct {
	client feedpb.FeedServiceClient
	conn   *grpc.ClientConn
}

func NewArticleServiceClient(address string) (*ArticleServiceClient, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Feed Service at %s: %w", address, err)
	}

	return &ArticleServiceClient{
		client: feedpb.NewFeedServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *ArticleServiceClient) Close() error {
	return c.conn.Close()
}

func (c *ArticleServiceClient) TriggerFetch(ctx context.Context, userID, feedID uint) error {
	_, err := c.client.TriggerFetch(ctx, &feedpb.TriggerFetchRequest{
		UserId: uint64(userID),
		FeedId: uint64(feedID),
	})
	if err != nil {
		return fmt.Errorf("failed to trigger fetch: %w", err)
	}
	return nil
}
