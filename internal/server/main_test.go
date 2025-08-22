package server

import (
	"log"
	"log/slog"
	"net"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"google.golang.org/grpc"
	"gorm.io/gorm"

	"github.com/Fancu1/phoenix-rss/internal/config"
	"github.com/Fancu1/phoenix-rss/internal/core"
	"github.com/Fancu1/phoenix-rss/internal/events"
	feedCore "github.com/Fancu1/phoenix-rss/internal/feed-service/core"
	feedHandler "github.com/Fancu1/phoenix-rss/internal/feed-service/handler"
	feedModels "github.com/Fancu1/phoenix-rss/internal/feed-service/models"
	feedRepo "github.com/Fancu1/phoenix-rss/internal/feed-service/repository"
	feedWorker "github.com/Fancu1/phoenix-rss/internal/feed-service/worker"
	"github.com/Fancu1/phoenix-rss/internal/repository"
	userCore "github.com/Fancu1/phoenix-rss/internal/user-service/core"
	"github.com/Fancu1/phoenix-rss/internal/user-service/handler"
	userModels "github.com/Fancu1/phoenix-rss/internal/user-service/models"
	userRepo "github.com/Fancu1/phoenix-rss/internal/user-service/repository"
	"github.com/Fancu1/phoenix-rss/pkg/logger"
	feedpb "github.com/Fancu1/phoenix-rss/protos/gen/go/feed"
	userpb "github.com/Fancu1/phoenix-rss/protos/gen/go/user"
)

var app *TestApp

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
	db := repository.InitDB(&cfg.Database)

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
	s := New(cfg, logger.New(slog.LevelDebug), feedService, articleService, userService)

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

	// Initialize services
	feedService := feedCore.NewFeedService(feedRepository, logger.New(slog.LevelDebug))
	articleService := feedCore.NewArticleService(feedRepository, articleRepository, logger.New(slog.LevelDebug))

	// Create event handler for processing
	eventHandler := feedWorker.NewEventHandler(logger.New(slog.LevelDebug), articleService)

	// In tests, use in-memory bus to avoid Kafka dependency
	memBus := events.NewMemoryBus(logger.New(slog.LevelDebug), eventHandler.HandleFeedFetch)

	// Create gRPC handler with memory bus as producer
	grpcHandler := feedHandler.NewFeedServiceHandler(
		logger.New(slog.LevelDebug),
		feedService,
		articleService,
		memBus,
	)

	// Create worker and register the memory bus as consumer
	appWorker := feedWorker.NewWorker(logger.New(slog.LevelDebug))
	appWorker.RegisterConsumer(memBus)

	// Start worker in background
	go func() {
		if err := appWorker.Start(); err != nil {
			log.Printf("Feed service worker error: %v", err)
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
		appWorker.Stop()
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
