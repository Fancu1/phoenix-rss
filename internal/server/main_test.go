package server

import (
	"log"
	"log/slog"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/hibiken/asynq"
	"gorm.io/gorm"

	"github.com/Fancu1/phoenix-rss/internal/config"
	"github.com/Fancu1/phoenix-rss/internal/core"
	"github.com/Fancu1/phoenix-rss/internal/logger"
	"github.com/Fancu1/phoenix-rss/internal/repository"
	"github.com/Fancu1/phoenix-rss/internal/worker"
)

var app *TestApp

type TestApp struct {
	Server    *httptest.Server
	DB        *gorm.DB
	Inspector *asynq.Inspector
	Worker    *asynq.Server
}

func TestMain(m *testing.M) {
	// Load config from the same file as the application
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config for testing: %v", err)
	}

	// Connect to the database started by db-setup.sh
	db := repository.InitDB(&cfg.Database)
	if err := repository.InitSchema(db); err != nil {
		log.Fatalf("Failed to init schema: %v", err)
	}

	// Connect to the redis instance started by redis-setup.sh
	redisOpt := asynq.RedisClientOpt{Addr: cfg.Redis.Address}
	taskClient := asynq.NewClient(redisOpt)
	inspector := asynq.NewInspector(redisOpt)
	defer taskClient.Close()
	defer inspector.Close()

	// Setup services and handlers
	feedRepo := repository.NewFeedRepository(db)
	articleRepo := repository.NewArticleRepository(db)
	feedService := core.NewFeedService(feedRepo, logger.New(slog.LevelDebug))
	articleService := core.NewArticleService(feedRepo, articleRepo, logger.New(slog.LevelDebug))
	userRepo := repository.NewUserRepository(db)
	userService := core.NewUserService(userRepo, cfg.Auth.JWTSecret)
	s := New(cfg, logger.New(slog.LevelDebug), taskClient, feedService, articleService, userService, feedRepo)

	// Start worker for processing tasks during tests
	processor := worker.NewTaskProcesser(logger.New(slog.LevelDebug), redisOpt, articleService)

	// Start worker in background
	go func() {
		if err := processor.Start(); err != nil {
			log.Printf("Worker error: %v", err)
		}
	}()

	app = &TestApp{
		Server:    httptest.NewServer(s.engine),
		DB:        db,
		Inspector: inspector,
		Worker:    nil, // We don't need to expose the worker server directly
	}
	defer app.Server.Close()
	defer processor.Stop()

	// Run tests
	code := m.Run()

	os.Exit(code)
}
