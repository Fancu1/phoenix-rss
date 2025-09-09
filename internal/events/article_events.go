package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"

	article_eventspb "github.com/Fancu1/phoenix-rss/proto/gen/article_events"
)

// ArticleEventProducer handle article-related event publishing
type ArticleEventProducer interface {
	PublishArticlePersisted(ctx context.Context, event *article_eventspb.ArticlePersistedEvent) error
	Close() error
}

// ArticleEventConsumer handle article-related event consumption
type ArticleEventConsumer interface {
	StartProcessedEventConsumer(ctx context.Context, handler func(ctx context.Context, event *article_eventspb.ArticleProcessedEvent) error) error
	Stop(ctx context.Context) error
}

// KafkaArticleEventProducer implement ArticleEventProducer using Kafka
type KafkaArticleEventProducer struct {
	logger           *slog.Logger
	articleNewWriter *kafka.Writer
	articleNewTopic  string
}

// NewKafkaArticleEventProducer create a new Kafka-based article event producer
func NewKafkaArticleEventProducer(logger *slog.Logger, brokers []string, articleNewTopic string) *KafkaArticleEventProducer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        articleNewTopic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		Async:        false,
	}

	return &KafkaArticleEventProducer{
		logger:           logger,
		articleNewWriter: writer,
		articleNewTopic:  articleNewTopic,
	}
}

// PublishArticlePersisted publishe an ArticlePersistedEvent to Kafka
func (p *KafkaArticleEventProducer) PublishArticlePersisted(ctx context.Context, event *article_eventspb.ArticlePersistedEvent) error {
	data, err := proto.Marshal(event)
	if err != nil {
		data, err = json.Marshal(event)
		if err != nil {
			return fmt.Errorf("failed to marshal article persisted event: %w", err)
		}
	}

	// Create Kafka message
	message := kafka.Message{
		Key:   []byte(fmt.Sprintf("article_%d", event.ArticleId)),
		Value: data,
		Headers: []kafka.Header{
			{
				Key:   "event_type",
				Value: []byte("article_persisted"),
			},
			{
				Key:   "source",
				Value: []byte("feed-service"),
			},
		},
		Time: time.Now(),
	}

	// Send message
	if err := p.articleNewWriter.WriteMessages(ctx, message); err != nil {
		return fmt.Errorf("failed to write article persisted event to Kafka: %w", err)
	}

	p.logger.Debug("published article persisted event",
		"article_id", event.ArticleId,
		"feed_id", event.FeedId,
		"topic", p.articleNewTopic,
	)

	return nil
}

// Close closes the producer
func (p *KafkaArticleEventProducer) Close() error {
	p.logger.Info("closing kafka article event producer")
	if p.articleNewWriter != nil {
		return p.articleNewWriter.Close()
	}
	return nil
}

// KafkaArticleEventConsumer implement ArticleEventConsumer using Kafka
type KafkaArticleEventConsumer struct {
	logger                *slog.Logger
	brokers               []string
	groupID               string
	articleProcessedTopic string
	processedEventReader  *kafka.Reader
}

// NewKafkaArticleEventConsumer create a new Kafka-based article event consumer
func NewKafkaArticleEventConsumer(logger *slog.Logger, brokers []string, groupID string, articleProcessedTopic string) *KafkaArticleEventConsumer {
	return &KafkaArticleEventConsumer{
		logger:                logger,
		brokers:               brokers,
		groupID:               groupID,
		articleProcessedTopic: articleProcessedTopic,
	}
}

// StartProcessedEventConsumer start consuming ArticleProcessedEvent messages
func (c *KafkaArticleEventConsumer) StartProcessedEventConsumer(ctx context.Context, handler func(ctx context.Context, event *article_eventspb.ArticleProcessedEvent) error) error {
	c.processedEventReader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:        c.brokers,
		Topic:          c.articleProcessedTopic,
		GroupID:        c.groupID,
		MinBytes:       1,
		MaxBytes:       10e6, // 10MB
		CommitInterval: time.Second,
	})

	c.logger.Info("starting article processed event consumer",
		"topic", c.articleProcessedTopic,
		"group_id", c.groupID,
		"brokers", c.brokers,
	)

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("stopping article processed event consumer due to context cancellation")
			return ctx.Err()
		default:
		}

		message, err := c.processedEventReader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			c.logger.Error("failed to fetch processed event message", "error", err)
			continue
		}

		if err := c.processProcessedEventMessage(ctx, message, handler); err != nil {
			c.logger.Error("failed to process processed event message",
				"error", err,
				"offset", message.Offset,
				"partition", message.Partition,
			)
		}

		// Commit the message
		if err := c.processedEventReader.CommitMessages(ctx, message); err != nil {
			c.logger.Error("failed to commit processed event message", "error", err)
		}
	}
}

// processProcessedEventMessage process a single ArticleProcessedEvent message
func (c *KafkaArticleEventConsumer) processProcessedEventMessage(ctx context.Context, message kafka.Message, handler func(ctx context.Context, event *article_eventspb.ArticleProcessedEvent) error) error {
	c.logger.Debug("processing article processed event message",
		"offset", message.Offset,
		"partition", message.Partition,
		"key", string(message.Key),
	)

	var event article_eventspb.ArticleProcessedEvent
	if err := c.unmarshalProcessedEvent(message.Value, &event); err != nil {
		return fmt.Errorf("failed to unmarshal processed event: %w", err)
	}

	c.logger.Info("received article processed event",
		"article_id", event.ArticleId,
		"summary_length", len(event.Summary),
	)

	if err := handler(ctx, &event); err != nil {
		return fmt.Errorf("handler failed for article processed event: %w", err)
	}

	return nil
}

// unmarshalProcessedEvent unmarshal the processed event
func (c *KafkaArticleEventConsumer) unmarshalProcessedEvent(data []byte, event *article_eventspb.ArticleProcessedEvent) error {
	if err := proto.Unmarshal(data, event); err == nil {
		return nil
	}

	// Fallback to JSON
	if err := json.Unmarshal(data, event); err != nil {
		return fmt.Errorf("failed to unmarshal as both protobuf and JSON: %w", err)
	}

	return nil
}

// Stop gracefully stop the consumer
func (c *KafkaArticleEventConsumer) Stop(ctx context.Context) error {
	c.logger.Info("stopping kafka article event consumer")

	if c.processedEventReader != nil {
		if err := c.processedEventReader.Close(); err != nil {
			c.logger.Error("failed to close processed event reader", "error", err)
			return err
		}
	}

	return nil
}
