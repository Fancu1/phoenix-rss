package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"

	"github.com/Fancu1/phoenix-rss/internal/config"
	"github.com/Fancu1/phoenix-rss/internal/events"
	"github.com/Fancu1/phoenix-rss/internal/feed-service/core"
	"github.com/Fancu1/phoenix-rss/internal/feed-service/handler"
	"github.com/Fancu1/phoenix-rss/internal/feed-service/repository"
	"github.com/Fancu1/phoenix-rss/internal/feed-service/worker"
	"github.com/Fancu1/phoenix-rss/internal/logger"
	feedpb "github.com/Fancu1/phoenix-rss/protos/gen/go/feed"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	log := logger.New(slog.LevelDebug)

	db := repository.InitDB(&cfg.Database)

	feedRepo := repository.NewFeedRepository(db)
	articleRepo := repository.NewArticleRepository(db)

	feedService := core.NewFeedService(feedRepo, log)
	articleService := core.NewArticleService(feedRepo, articleRepo, log)

	producer := events.NewKafkaProducer(log, events.KafkaConfig{
		Brokers: cfg.Kafka.Brokers,
		Topic:   cfg.Kafka.Topic,
		GroupID: cfg.Kafka.GroupID,
	})
	defer producer.Close()

	eventHandler := worker.NewEventHandler(log, articleService)

	consumer := events.NewKafkaConsumer(log, events.KafkaConfig{
		Brokers: cfg.Kafka.Brokers,
		Topic:   cfg.Kafka.Topic,
		GroupID: "feed-service-" + cfg.Kafka.GroupID, // different group ID for feed service
	}, eventHandler.HandleFeedFetch)

	grpcHandler := handler.NewFeedServiceHandler(log, feedService, articleService, producer)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return startGRPCServer(ctx, grpcHandler, cfg.FeedService.Port, log)
	})

	g.Go(func() error {
		log.Info("starting Kafka consumer")
		return consumer.Start(ctx)
	})

	g.Go(func() error {
		select {
		case sig := <-signalChan:
			log.Info("received shutdown signal", "signal", sig)
			cancel()
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	if err := g.Wait(); err != nil && err != context.Canceled {
		log.Error("Feed Service error", "error", err)
		os.Exit(1)
	}

	log.Info("Feed Service shutdown completed")
}

func startGRPCServer(ctx context.Context, handler *handler.FeedServiceHandler, port int, log *slog.Logger) error {
	address := fmt.Sprintf(":%d", port)
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", address, err)
	}

	grpcServer := grpc.NewServer()
	feedpb.RegisterFeedServiceServer(grpcServer, handler)

	log.Info("starting gRPC server", "address", address)

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- grpcServer.Serve(lis)
	}()

	select {
	case err := <-serverErr:
		return fmt.Errorf("gRPC server error: %w", err)
	case <-ctx.Done():
		log.Info("gracefully stopping gRPC server")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		shutdownComplete := make(chan struct{})
		go func() {
			grpcServer.GracefulStop()
			close(shutdownComplete)
		}()

		select {
		case <-shutdownComplete:
			log.Info("gRPC server stopped gracefully")
		case <-shutdownCtx.Done():
			log.Warn("gRPC server shutdown timeout, forcing stop")
			grpcServer.Stop()
		}

		return nil
	}
}
