package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"

	"github.com/Fancu1/phoenix-rss/internal/ai-service/core"
	article_eventspb "github.com/Fancu1/phoenix-rss/proto/gen/article_events"
)

// ArticleProcessor handle Kafka events for AI article processing
type ArticleProcessor struct {
	logger            *slog.Logger
	processingService *core.ProcessingService
	consumer          *kafka.Reader
	producer          *kafka.Writer
	brokers           []string
	groupID           string
	inputTopic        string
	outputTopic       string
}

// NewArticleProcessor creates a new article processor instance
func NewArticleProcessor(
	logger *slog.Logger,
	processingService *core.ProcessingService,
	brokers []string,
	groupID string,
	inputTopic string,
	outputTopic string,
) *ArticleProcessor {
	return &ArticleProcessor{
		logger:            logger,
		processingService: processingService,
		brokers:           brokers,
		groupID:           groupID,
		inputTopic:        inputTopic,
		outputTopic:       outputTopic,
	}
}

// Start begins processing article events from Kafka
func (p *ArticleProcessor) Start(ctx context.Context) error {
	p.consumer = kafka.NewReader(kafka.ReaderConfig{
		Brokers:        p.brokers,
		Topic:          p.inputTopic,
		GroupID:        p.groupID,
		MinBytes:       1,
		MaxBytes:       10e6, // 10MB
		CommitInterval: time.Second,
	})

	p.producer = &kafka.Writer{
		Addr:         kafka.TCP(p.brokers...),
		Topic:        p.outputTopic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		Async:        false,
	}

	p.logger.Info("starting AI article processor",
		"input_topic", p.inputTopic,
		"output_topic", p.outputTopic,
		"group_id", p.groupID,
		"brokers", p.brokers,
	)

	defer func() {
		if p.consumer != nil {
			p.consumer.Close()
		}
		if p.producer != nil {
			p.producer.Close()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("stopping article processor due to context cancellation")
			return ctx.Err()
		default:
		}

		message, err := p.consumer.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			p.logger.Error("failed to fetch message", "error", err)
			continue
		}

		if err := p.processMessage(ctx, message); err != nil {
			p.logger.Error("failed to process message",
				"error", err,
				"offset", message.Offset,
				"partition", message.Partition,
			)
		}

		// Commit the message
		if err := p.consumer.CommitMessages(ctx, message); err != nil {
			p.logger.Error("failed to commit message", "error", err)
		}
	}
}

// Stop stops the article processor
func (p *ArticleProcessor) Stop(ctx context.Context) error {
	p.logger.Info("stopping AI article processor")

	if p.consumer != nil {
		if err := p.consumer.Close(); err != nil {
			p.logger.Error("failed to close consumer", "error", err)
		}
	}

	// Close producer
	if p.producer != nil {
		if err := p.producer.Close(); err != nil {
			p.logger.Error("failed to close producer", "error", err)
		}
	}

	return nil
}

// processMessage processes a single Kafka message
func (p *ArticleProcessor) processMessage(ctx context.Context, message kafka.Message) error {
	p.logger.Debug("processing message",
		"offset", message.Offset,
		"partition", message.Partition,
		"key", string(message.Key),
	)

	// Parse the message as ArticlePersistedEvent
	var event article_eventspb.ArticlePersistedEvent
	if err := p.unmarshalEvent(message.Value, &event); err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	p.logger.Info("received article persisted event",
		"article_id", event.ArticleId,
		"feed_id", event.FeedId,
		"title", event.Title,
	)

	// Process the article
	processedEvent, err := p.processingService.ProcessArticle(ctx, &event)
	if err != nil {
		return fmt.Errorf("failed to process article: %w", err)
	}

	// Publish the processed event
	if err := p.publishProcessedEvent(ctx, processedEvent); err != nil {
		return fmt.Errorf("failed to publish processed event: %w", err)
	}

	p.logger.Info("successfully processed and published article",
		"article_id", event.ArticleId,
		"summary_length", len(processedEvent.Summary),
	)

	return nil
}

// unmarshalEvent unmarshals the event based on the message format
func (p *ArticleProcessor) unmarshalEvent(data []byte, event *article_eventspb.ArticlePersistedEvent) error {
	if err := proto.Unmarshal(data, event); err == nil {
		return nil
	}

	if err := json.Unmarshal(data, event); err != nil {
		return fmt.Errorf("failed to unmarshal as both protobuf and JSON: %w", err)
	}

	return nil
}

// publishProcessedEvent publishes the processed event to Kafka
func (p *ArticleProcessor) publishProcessedEvent(ctx context.Context, event *article_eventspb.ArticleProcessedEvent) error {
	data, err := proto.Marshal(event)
	if err != nil {
		data, err = json.Marshal(event)
		if err != nil {
			return fmt.Errorf("failed to marshal processed event: %w", err)
		}
	}

	message := kafka.Message{
		Key:   []byte(fmt.Sprintf("article_%d", event.ArticleId)),
		Value: data,
		Headers: []kafka.Header{
			{
				Key:   "event_type",
				Value: []byte("article_processed"),
			},
			{
				Key:   "source",
				Value: []byte("ai-service"),
			},
		},
		Time: time.Now(),
	}

	if err := p.producer.WriteMessages(ctx, message); err != nil {
		return fmt.Errorf("failed to write message to Kafka: %w", err)
	}

	p.logger.Debug("published processed event",
		"article_id", event.ArticleId,
		"topic", p.outputTopic,
	)

	return nil
}
