package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/Fancu1/phoenix-rss/internal/config"
	"github.com/Fancu1/phoenix-rss/internal/events"
	"github.com/Fancu1/phoenix-rss/internal/feed-service/core"
	"github.com/Fancu1/phoenix-rss/internal/feed-service/handler"
	"github.com/Fancu1/phoenix-rss/internal/feed-service/repository"
	"github.com/Fancu1/phoenix-rss/internal/feed-service/worker"
	"github.com/Fancu1/phoenix-rss/pkg/logger"
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

	aiEventProducer := events.NewKafkaArticleEventProducer(log, cfg.Kafka.Brokers, cfg.Kafka.AIProcessing.ArticlesNewTopic)
	defer aiEventProducer.Close()

	aiEventConsumer := events.NewKafkaArticleEventConsumer(
		log,
		cfg.Kafka.Brokers,
		cfg.Kafka.AIProcessing.FeedServiceAIGroupID,
		cfg.Kafka.AIProcessing.ArticlesProcessedTopic,
	)

	feedService := core.NewFeedService(feedRepo, log)
	articleService := core.NewArticleService(feedRepo, articleRepo, aiEventProducer, log)

	updateTimeout, err := time.ParseDuration(cfg.FeedService.ArticleUpdate.HTTPTimeout)
	if err != nil {
		log.Error("invalid article update http timeout", "value", cfg.FeedService.ArticleUpdate.HTTPTimeout, "error", err)
		os.Exit(1)
	}
	backoffInitial, err := time.ParseDuration(cfg.FeedService.ArticleUpdate.HTTPRetryBackoffInitial)
	if err != nil {
		log.Error("invalid article update backoff initial", "value", cfg.FeedService.ArticleUpdate.HTTPRetryBackoffInitial, "error", err)
		os.Exit(1)
	}
	backoffMax, err := time.ParseDuration(cfg.FeedService.ArticleUpdate.HTTPRetryBackoffMax)
	if err != nil {
		log.Error("invalid article update backoff max", "value", cfg.FeedService.ArticleUpdate.HTTPRetryBackoffMax, "error", err)
		os.Exit(1)
	}
	robotsTTL, err := time.ParseDuration(cfg.FeedService.ArticleUpdate.RobotsCacheTTL)
	if err != nil {
		log.Error("invalid robots cache ttl", "value", cfg.FeedService.ArticleUpdate.RobotsCacheTTL, "error", err)
		os.Exit(1)
	}

	httpClient := &http.Client{Timeout: updateTimeout}
	robotsClient := core.NewRobotsClient(httpClient, robotsTTL, log)
	articleChecker := core.NewArticleUpdateChecker(articleRepo, log, httpClient, robotsClient, core.ArticleUpdateConfig{
		UserAgent:       cfg.FeedService.ArticleUpdate.HTTPUserAgent,
		MaxAttempts:     cfg.FeedService.ArticleUpdate.HTTPRetryMaxAttempts,
		BackoffInitial:  backoffInitial,
		BackoffMax:      backoffMax,
		Jitter:          cfg.FeedService.ArticleUpdate.HTTPRetryJitter,
		MaxContentBytes: cfg.FeedService.ArticleUpdate.MaxContentBytes,
		RespectRobots:   cfg.FeedService.ArticleUpdate.RespectRobots,
	})
	articleUpdateWorker := worker.NewArticleUpdateWorker(log, articleChecker)

	articleCheckConsumer := events.NewKafkaArticleCheckConsumer(log, events.KafkaConfig{
		Brokers: cfg.Kafka.Brokers,
		Topic:   cfg.Kafka.ArticleCheck.Topic,
		GroupID: cfg.Kafka.ArticleCheck.FeedServiceGroupID,
	}, articleUpdateWorker.HandleArticleCheck)
	defer articleCheckConsumer.Stop(context.Background())

	feedFetchProducer := events.NewKafkaProducer(log, events.KafkaConfig{
		Brokers: cfg.Kafka.Brokers,
		Topic:   cfg.Kafka.FeedFetch.Topic,
		GroupID: cfg.Kafka.FeedFetch.FeedServiceGroupID,
	})
	defer feedFetchProducer.Close()

	feedFetcher := worker.NewFeedFetcher(log, articleService)

	feedFetchConsumer := events.NewKafkaConsumer(log, events.KafkaConfig{
		Brokers: cfg.Kafka.Brokers,
		Topic:   cfg.Kafka.FeedFetch.Topic,
		GroupID: cfg.Kafka.FeedFetch.FeedServiceGroupID,
	}, feedFetcher.HandleFeedFetch)

	aiResultHandler := worker.NewAIResultHandler(log, articleService, aiEventConsumer)

	grpcHandler := handler.NewFeedServiceHandler(log, feedService, articleService, feedFetchProducer)

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
		return feedFetchConsumer.Start(ctx)
	})

	g.Go(func() error {
		log.Info("starting AI event handler")
		return aiResultHandler.Start(ctx)
	})

	g.Go(func() error {
		log.Info("starting article check consumer")
		return articleCheckConsumer.Start(ctx)
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

	// register gRPC health check service
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

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
