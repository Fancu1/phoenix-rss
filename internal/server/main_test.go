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
	"github.com/Fancu1/phoenix-rss/internal/logger"
	"github.com/Fancu1/phoenix-rss/internal/repository"
	userCore "github.com/Fancu1/phoenix-rss/internal/user-service/core"
	"github.com/Fancu1/phoenix-rss/internal/user-service/handler"
	userRepo "github.com/Fancu1/phoenix-rss/internal/user-service/repository"
	"github.com/Fancu1/phoenix-rss/internal/worker"
	userpb "github.com/Fancu1/phoenix-rss/protos/gen/go/user"
)

var app *TestApp

type TestApp struct {
	Server       *httptest.Server
	DB           *gorm.DB
	StopFunc     func()
	UserGRPCStop func()
}

func TestMain(m *testing.M) {
	// Load config from the same file as the application
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config for testing: %v", err)
	}

	// Connect to the database started by db-setup.sh
	db := repository.InitDB(&cfg.Database)

	// Start a test gRPC User Service
	userGRPCAddr := "127.0.0.1:50052" // Use different port for testing
	userGRPCStop := startTestUserService(db, cfg.Auth.JWTSecret, userGRPCAddr)

	// Wait a moment for the gRPC server to start
	time.Sleep(100 * time.Millisecond)

	// Setup services and handlers
	feedRepo := repository.NewFeedRepository(db)
	articleRepo := repository.NewArticleRepository(db)
	feedService := core.NewFeedService(feedRepo, logger.New(slog.LevelDebug))
	articleService := core.NewArticleService(feedRepo, articleRepo, logger.New(slog.LevelDebug))

	// Create gRPC client for user service
	userService, err := core.NewUserServiceClient(userGRPCAddr)
	if err != nil {
		log.Fatalf("Failed to create user service client: %v", err)
	}

	// Create event handler for processing
	eventHandler := worker.NewEventHandler(logger.New(slog.LevelDebug), articleService)

	// In tests, use in-memory bus to avoid Kafka dependency
	memBus := events.NewMemoryBus(logger.New(slog.LevelDebug), eventHandler.HandleFeedFetch)

	// Create server with memory bus as producer
	s := New(cfg, logger.New(slog.LevelDebug), memBus, feedService, articleService, userService, feedRepo)

	// Create worker and register the memory bus as consumer
	appWorker := worker.NewWorker(logger.New(slog.LevelDebug))
	appWorker.RegisterConsumer(memBus)

	// Start worker in background
	go func() {
		if err := appWorker.Start(); err != nil {
			log.Printf("Worker error: %v", err)
		}
	}()

	app = &TestApp{
		Server:       httptest.NewServer(s.engine),
		DB:           db,
		StopFunc:     appWorker.Stop,
		UserGRPCStop: userGRPCStop,
	}
	defer app.Server.Close()
	defer appWorker.Stop()
	defer userGRPCStop()
	defer userService.Close()

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
