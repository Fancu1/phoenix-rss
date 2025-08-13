package events

import (
	"context"
	"log/slog"
)

// MemoryBus is a simple in-process implementation for tests
type MemoryBus struct {
	logger  *slog.Logger
	handler func(ctx context.Context, evt FeedFetchEvent) error
	ch      chan FeedFetchEvent
}

func NewMemoryBus(logger *slog.Logger, handler func(ctx context.Context, evt FeedFetchEvent) error) *MemoryBus {
	return &MemoryBus{
		logger:  logger,
		handler: handler,
		ch:      make(chan FeedFetchEvent, 1024),
	}
}

func (b *MemoryBus) PublishFeedFetch(ctx context.Context, feedID uint) error {
	b.ch <- FeedFetchEvent{FeedID: feedID}
	return nil
}

func (b *MemoryBus) Start(ctx context.Context) error {
	b.logger.Info("starting memory event bus")
	for {
		select {
		case <-ctx.Done():
			return nil
		case evt := <-b.ch:
			if b.handler != nil {
				if err := b.handler(ctx, evt); err != nil {
					b.logger.Error("memory handler error", "error", err)
				}
			}
		}
	}
}

func (b *MemoryBus) Stop(ctx context.Context) error {
	b.logger.Info("stopping memory event bus")
	return nil
}
