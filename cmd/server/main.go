package main

import (
	"fmt"
	"os"

	"github.com/Fancu1/phoenix-rss/internal/config"
	"github.com/Fancu1/phoenix-rss/internal/server"
)

func main() {
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	srv := server.New(cfg)
	if err := srv.Start(); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		os.Exit(1)
	}
}
