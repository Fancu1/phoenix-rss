package server

import (
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"

	"github.com/Fancu1/phoenix-rss/internal/config"
	"github.com/Fancu1/phoenix-rss/internal/core"
	"github.com/Fancu1/phoenix-rss/internal/handler"
)

type Server struct {
	config         *config.Config
	engine         *gin.Engine
	logger         *slog.Logger
	feedHandler    *handler.FeedHandler
	articleHandler *handler.ArticleHandler
}

func New(cfg *config.Config, logger *slog.Logger, taskClient *asynq.Client, feedService *core.FeedService, articleService *core.ArticleService) *Server {
	feedHandler := handler.NewFeedHandler(feedService)
	articleHandler := handler.NewArticleHandler(logger, taskClient, articleService)

	s := &Server{
		config:         cfg,
		engine:         gin.Default(),
		feedHandler:    feedHandler,
		articleHandler: articleHandler,
	}

	s.setupRoutes()

	return s
}

func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.config.Server.Port)
	fmt.Printf("Server listening on %s\n", addr)
	return s.engine.Run(addr)
}
