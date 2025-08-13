package server

import (
	"log"
	"log/slog"
	"net/http/httptest"
	"os"
	"testing"

	"gorm.io/gorm"

	"github.com/Fancu1/phoenix-rss/internal/config"
	"github.com/Fancu1/phoenix-rss/internal/core"
	"github.com/Fancu1/phoenix-rss/internal/events"
	"github.com/Fancu1/phoenix-rss/internal/logger"
	"github.com/Fancu1/phoenix-rss/internal/repository"
	"github.com/Fancu1/phoenix-rss/internal/worker"
)

var app *TestApp

type TestApp struct {
	Server   *httptest.Server
	DB       *gorm.DB
	StopFunc func()
}

func TestMain(m *testing.M) {
	// Load config from the same file as the application
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config for testing: %v", err)
	}

	// Connect to the database started by db-setup.sh
	db := repository.InitDB(&cfg.Database)

	// Setup services and handlers
	feedRepo := repository.NewFeedRepository(db)
	articleRepo := repository.NewArticleRepository(db)
	feedService := core.NewFeedService(feedRepo, logger.New(slog.LevelDebug))
	articleService := core.NewArticleService(feedRepo, articleRepo, logger.New(slog.LevelDebug))
	userRepo := repository.NewUserRepository(db)
	userService := core.NewUserService(userRepo, cfg.Auth.JWTSecret)

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
		Server:   httptest.NewServer(s.engine),
		DB:       db,
		StopFunc: appWorker.Stop,
	}
	defer app.Server.Close()
	defer appWorker.Stop()

	// Run tests
	code := m.Run()

	os.Exit(code)
}
