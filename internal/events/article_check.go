package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"
)

type ArticleCheckEvent struct {
	ArticleID        uint      `json:"article_id"`
	FeedID           uint      `json:"feed_id"`
	URL              string    `json:"url"`
	PrevETag         string    `json:"prev_etag,omitempty"`
	PrevLastModified string    `json:"prev_last_modified,omitempty"`
	RequestID        string    `json:"request_id"`
	Attempt          int       `json:"attempt"`
	ScheduledAt      time.Time `json:"scheduled_at"`
	Reason           string    `json:"reason"`
}

type ArticleCheckProducer interface {
	PublishArticleCheck(ctx context.Context, event ArticleCheckEvent) error
	Close() error
}

type ArticleCheckConsumer interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type KafkaArticleCheckProducer struct {
	logger *slog.Logger
	writer *kafka.Writer
}

func NewKafkaArticleCheckProducer(logger *slog.Logger, cfg KafkaConfig) *KafkaArticleCheckProducer {
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers: cfg.Brokers,
		Topic:   cfg.Topic,
	})

	return &KafkaArticleCheckProducer{logger: logger, writer: writer}
}

func (p *KafkaArticleCheckProducer) PublishArticleCheck(ctx context.Context, event ArticleCheckEvent) error {
	if event.Attempt <= 0 {
		event.Attempt = 1
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal article check event: %w", err)
	}

	key := fmt.Sprintf("%d", event.ArticleID)
	message := kafka.Message{Key: []byte(key), Value: payload}

	if err := p.writer.WriteMessages(ctx, message); err != nil {
		return fmt.Errorf("failed to write article check message: %w", err)
	}

	p.logger.Info("published article check event", "article_id", event.ArticleID, "topic", p.writer.Topic, "request_id", event.RequestID)
	return nil
}

func (p *KafkaArticleCheckProducer) Close() error {
	p.logger.Info("closing article check producer")
	return p.writer.Close()
}

type KafkaArticleCheckConsumer struct {
	logger  *slog.Logger
	reader  *kafka.Reader
	handler func(ctx context.Context, event ArticleCheckEvent) error
}

func NewKafkaArticleCheckConsumer(logger *slog.Logger, cfg KafkaConfig, handler func(ctx context.Context, event ArticleCheckEvent) error) *KafkaArticleCheckConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Brokers,
		GroupID:        cfg.GroupID,
		Topic:          cfg.Topic,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: 0,
	})

	return &KafkaArticleCheckConsumer{logger: logger, reader: reader, handler: handler}
}

func (c *KafkaArticleCheckConsumer) Start(ctx context.Context) error {
	c.logger.Info("starting article check consumer", "topic", c.reader.Config().Topic, "group", c.reader.Config().GroupID)

	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			c.logger.Error("failed to fetch article check message", "error", err)
			continue
		}

		var event ArticleCheckEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			c.logger.Error("failed to unmarshal article check event", "error", err)
			if commitErr := c.reader.CommitMessages(ctx, msg); commitErr != nil {
				c.logger.Error("failed to commit poisoned message", "error", commitErr)
			}
			continue
		}

		if err := c.handler(ctx, event); err != nil {
			c.logger.Error("article check handler failed", "error", err, "article_id", event.ArticleID, "request_id", event.RequestID)
			if commitErr := c.reader.CommitMessages(ctx, msg); commitErr != nil {
				c.logger.Error("failed to commit message after handler error", "error", commitErr)
			}
			continue
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			c.logger.Error("failed to commit article check message", "error", err)
			continue
		}
	}
}

func (c *KafkaArticleCheckConsumer) Stop(ctx context.Context) error {
	c.logger.Info("stopping article check consumer")
	return c.reader.Close()
}
