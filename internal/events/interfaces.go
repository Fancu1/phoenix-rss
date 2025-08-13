package events

import (
	"context"
)

// Producer define capabilities to publish domain events
type Producer interface {
	PublishFeedFetch(ctx context.Context, feedID uint) error
}

// Consumer define capabilities to consume domain events
type Consumer interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// EventType define supported event types
type EventType string

const (
	EventFeedFetch EventType = "feed:fetch"
)

// FeedFetchEvent is the payload for feed fetch requests
type FeedFetchEvent struct {
	FeedID uint `json:"feed_id"`
}
