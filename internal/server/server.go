package server

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/Fancu1/phoenix-rss/internal/config"
	"github.com/Fancu1/phoenix-rss/internal/core"
	"github.com/Fancu1/phoenix-rss/internal/handler"
	"github.com/Fancu1/phoenix-rss/internal/repository"
)

type Server struct {
	config      *config.Config
	engine      *gin.Engine
	feedHandler *handler.FeedHandler
}

func New(cfg *config.Config) *Server {
	feedRepo := repository.NewFeedRepository() // db options
	feedService := core.NewFeedService(feedRepo)

	s := &Server{
		config:      cfg,
		engine:      gin.Default(),
		feedHandler: handler.NewFeedHandler(feedService),
	}

	s.setupRoutes()

	return s
}

func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.config.Server.Port)
	fmt.Printf("Server listening on %s\n", addr)
	return s.engine.Run(addr)
}
