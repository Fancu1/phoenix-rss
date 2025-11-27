package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/segmentio/kafka-go"
)

// KafkaConfig contains producer/consumer configuration
type KafkaConfig struct {
	Brokers []string
	Topic   string
	GroupID string
}

// KafkaProducer implements Producer using kafka-go
type KafkaProducer struct {
	logger *slog.Logger
	writer *kafka.Writer
}

func NewKafkaProducer(logger *slog.Logger, cfg KafkaConfig) *KafkaProducer {
	w := kafka.NewWriter(kafka.WriterConfig{
		Brokers: cfg.Brokers,
		Topic:   cfg.Topic,
	})
	return &KafkaProducer{logger: logger, writer: w}
}

func (p *KafkaProducer) PublishFeedFetch(ctx context.Context, feedID uint) error {
	payload := FeedFetchEvent{FeedID: feedID}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal feed fetch event: %w", err)
	}
	msg := kafka.Message{Key: []byte("feed_id"), Value: data}
	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("failed to write kafka message: %w", err)
	}
	p.logger.Info("published feed fetch event", "topic", p.writer.Topic, "feed_id", feedID)
	return nil
}

// Close the producer
func (p *KafkaProducer) Close() error {
	p.logger.Info("closing kafka producer")
	return p.writer.Close()
}

// KafkaConsumer implements Consumer using kafka-go
type KafkaConsumer struct {
	logger  *slog.Logger
	cfg     KafkaConfig
	handler func(ctx context.Context, evt FeedFetchEvent) error
	reader  *kafka.Reader
}

func NewKafkaConsumer(logger *slog.Logger, cfg KafkaConfig, handler func(ctx context.Context, evt FeedFetchEvent) error) *KafkaConsumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Brokers,
		GroupID:        cfg.GroupID,
		Topic:          cfg.Topic,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: 0,
	})
	return &KafkaConsumer{logger: logger, cfg: cfg, handler: handler, reader: r}
}

func (c *KafkaConsumer) Start(ctx context.Context) error {
	c.logger.Info("starting kafka consumer", "group", c.cfg.GroupID, "topic", c.cfg.Topic)
	for {
		m, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			c.logger.Error("failed to fetch message", "error", err, "topic", c.cfg.Topic)
			continue
		}
		var evt FeedFetchEvent
		if err := json.Unmarshal(m.Value, &evt); err != nil {
			c.logger.Error("failed to unmarshal event", "error", err)
			continue
		}
		if err := c.handler(ctx, evt); err != nil {
			c.logger.Error("handler failed", "error", err, "feed_id", evt.FeedID)
			continue
		}
		if err := c.reader.CommitMessages(ctx, m); err != nil {
			c.logger.Error("failed to commit message", "error", err)
			continue
		}
	}
}

func (c *KafkaConsumer) Stop(ctx context.Context) error {
	c.logger.Info("stopping kafka consumer")
	return c.reader.Close()
}
