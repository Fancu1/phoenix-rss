package main

import (
	"fmt"
	"os"

	"github.com/Fancu1/phoenix-rss/internal/config"
	"github.com/Fancu1/phoenix-rss/internal/core"
	"github.com/Fancu1/phoenix-rss/internal/repository"
	"github.com/Fancu1/phoenix-rss/internal/server"
	"github.com/Fancu1/phoenix-rss/internal/worker"
)

func main() {
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	feedRepo := repository.NewFeedRepository()
	articleRepo := repository.NewArticleRepository()
	articleSvc := core.NewArticleService(feedRepo, articleRepo)

	dispatcher := worker.NewDispatcher(100, 5, articleSvc)
	dispatcher.Start()

	srv := server.New(cfg, dispatcher)
	if err := srv.Start(); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		os.Exit(1)
	}
}
