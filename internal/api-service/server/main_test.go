package server

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"log/slog"
	"net"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"gorm.io/gorm"

	"github.com/Fancu1/phoenix-rss/internal/api-service/core"
	"github.com/Fancu1/phoenix-rss/internal/config"
	"github.com/Fancu1/phoenix-rss/internal/events"
	feedCore "github.com/Fancu1/phoenix-rss/internal/feed-service/core"
	feedHandler "github.com/Fancu1/phoenix-rss/internal/feed-service/handler"
	feedModels "github.com/Fancu1/phoenix-rss/internal/feed-service/models"
	feedRepo "github.com/Fancu1/phoenix-rss/internal/feed-service/repository"
	feedWorker "github.com/Fancu1/phoenix-rss/internal/feed-service/worker"
	userCore "github.com/Fancu1/phoenix-rss/internal/user-service/core"
	"github.com/Fancu1/phoenix-rss/internal/user-service/handler"
	userModels "github.com/Fancu1/phoenix-rss/internal/user-service/models"
	userRepo "github.com/Fancu1/phoenix-rss/internal/user-service/repository"
	"github.com/Fancu1/phoenix-rss/pkg/logger"
	article_eventspb "github.com/Fancu1/phoenix-rss/proto/gen/article_events"
	feedpb "github.com/Fancu1/phoenix-rss/protos/gen/go/feed"
	userpb "github.com/Fancu1/phoenix-rss/protos/gen/go/user"
)

// MockArticleEventProducer is a simple mock implementation for testing
type MockArticleEventProducer struct{}

func (m *MockArticleEventProducer) PublishArticlePersisted(ctx context.Context, event *article_eventspb.ArticlePersistedEvent) error {
	// In tests, we just ignore the events
	return nil
}

func (m *MockArticleEventProducer) Close() error {
	return nil
}

var app *TestApp

//go:embed testdata/dist/**
var testStaticFS embed.FS

type TestApp struct {
	Server       *httptest.Server
	DB           *gorm.DB
	StopFunc     func()
	UserGRPCStop func()
	FeedGRPCStop func()
}

func TestMain(m *testing.M) {
	// Load config from the same file as the application
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config for testing: %v", err)
	}

	// Connect to the database started by db-setup.sh
	db := feedRepo.InitDB(&cfg.Database)

	// Run database migrations (ensure tables exist)
	runMigrations(db)

	// Start a test gRPC User Service
	userGRPCAddr := "127.0.0.1:50052" // Use different port for testing
	userGRPCStop := startTestUserService(db, cfg.Auth.JWTSecret, userGRPCAddr)

	// Start a test gRPC Feed Service
	feedGRPCAddr := "127.0.0.1:50054" // Use different port for testing
	feedGRPCStop := startTestFeedService(db, feedGRPCAddr)

	// Wait a moment for the gRPC servers to start
	time.Sleep(200 * time.Millisecond)

	// Create gRPC clients
	userService, err := core.NewUserServiceClient(userGRPCAddr)
	if err != nil {
		log.Fatalf("Failed to create user service client: %v", err)
	}

	feedService, err := core.NewFeedServiceClient(feedGRPCAddr)
	if err != nil {
		log.Fatalf("Failed to create feed service client: %v", err)
	}

	articleService, err := core.NewArticleServiceClient(feedGRPCAddr)
	if err != nil {
		log.Fatalf("Failed to create article service client: %v", err)
	}

	// Create server with gRPC clients
	staticFS, err := fs.Sub(testStaticFS, "testdata")
	if err != nil {
		log.Fatalf("Failed to load test static assets: %v", err)
	}

	redisClient := redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})
	defer redisClient.Close()

	s, err := New(cfg, db, feedService, articleService, userService, redisClient, staticFS)
	if err != nil {
		log.Fatalf("Failed to create test server: %v", err)
	}

	app = &TestApp{
		Server:       httptest.NewServer(s.engine),
		DB:           db,
		StopFunc:     func() {}, // No worker in new architecture
		UserGRPCStop: userGRPCStop,
		FeedGRPCStop: feedGRPCStop,
	}
	defer app.Server.Close()
	defer userGRPCStop()
	defer feedGRPCStop()
	defer userService.Close()
	defer feedService.Close()

	// Run tests
	code := m.Run()

	os.Exit(code)
}

// startTestUserService starts a gRPC user service for testing
func startTestUserService(db *gorm.DB, jwtSecret, address string) func() {
	// Initialize user repository and service for the gRPC service
	userRepository := userRepo.NewUserRepository(db)
	userSvc := userCore.NewUserService(userRepository, jwtSecret)

	// Create gRPC handler
	grpcHandler := handler.NewUserServiceHandler(userSvc)

	// Create gRPC server
	grpcServer := grpc.NewServer()
	userpb.RegisterUserServiceServer(grpcServer, grpcHandler)

	// Start listening
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", address, err)
	}

	// Start server in background
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	// Return stop function
	return func() {
		grpcServer.GracefulStop()
	}
}

// startTestFeedService starts a gRPC feed service for testing with in-memory Kafka
func startTestFeedService(db *gorm.DB, address string) func() {
	// Use the same database instance for consistency in tests
	feedDB := db

	// Initialize repositories
	feedRepository := feedRepo.NewFeedRepository(feedDB)
	articleRepository := feedRepo.NewArticleRepository(feedDB)

	// Create a mock article event producer for testing
	mockEventProducer := &MockArticleEventProducer{}

	// Initialize services (pass nil for producer in tests - will use memBus later)
	feedService := feedCore.NewFeedService(feedRepository, logger.New(slog.LevelDebug), nil)
	articleService := feedCore.NewArticleService(feedRepository, articleRepository, mockEventProducer, logger.New(slog.LevelDebug))

	// Create event handler for processing
	feedFetcher := feedWorker.NewFeedFetcher(logger.New(slog.LevelDebug), articleService, feedRepository)

	// In tests, use in-memory bus to avoid Kafka dependency
	memBus := events.NewMemoryBus(logger.New(slog.LevelDebug), feedFetcher.HandleFeedFetch)

	// Create gRPC handler with memory bus as producer
	grpcHandler := feedHandler.NewFeedServiceHandler(
		logger.New(slog.LevelDebug),
		feedService,
		articleService,
		memBus,
	)

	// Start memory bus in background for test events
	go func() {
		ctx := context.Background()
		if err := memBus.Start(ctx); err != nil {
			log.Printf("Memory bus error: %v", err)
		}
	}()

	// Create gRPC server
	grpcServer := grpc.NewServer()
	feedpb.RegisterFeedServiceServer(grpcServer, grpcHandler)

	// Start listening
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", address, err)
	}

	// Start server in background
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("Feed gRPC server error: %v", err)
		}
	}()

	// Return stop function
	return func() {
		grpcServer.GracefulStop()
	}
}

// runMigrations perform GORM AutoMigrate for all models to ensure database schema is ready
func runMigrations(db *gorm.DB) {
	err := db.AutoMigrate(
		&userModels.User{},
		&feedModels.Feed{},
		&feedModels.Article{},
		&feedModels.Subscription{},
	)
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
}
