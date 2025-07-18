package server

import (
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"

	"github.com/Fancu1/phoenix-rss/internal/config"
	"github.com/Fancu1/phoenix-rss/internal/core"
	"github.com/Fancu1/phoenix-rss/internal/handler"
	"github.com/Fancu1/phoenix-rss/internal/repository"
)

type Server struct {
	config         *config.Config
	engine         *gin.Engine
	logger         *slog.Logger
	feedHandler    *handler.FeedHandler
	articleHandler *handler.ArticleHandler
	userHandler    *handler.UserHandler
	authMiddleware *handler.AuthMiddleware
}

func New(cfg *config.Config, logger *slog.Logger, taskClient *asynq.Client, feedService core.FeedServiceInterface, articleService *core.ArticleService, userService core.UserServiceInterface, feedRepo *repository.FeedRepository) *Server {
	feedHandler := handler.NewFeedHandler(feedService)
	articleHandler := handler.NewArticleHandler(logger, taskClient, articleService, feedRepo)
	userHandler := handler.NewUserHandler(userService)
	authMiddleware := handler.NewAuthMiddleware(userService)

	s := &Server{
		config:         cfg,
		engine:         gin.Default(),
		feedHandler:    feedHandler,
		articleHandler: articleHandler,
		userHandler:    userHandler,
		authMiddleware: authMiddleware,
	}

	s.setupRoutes()

	return s
}

func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.config.Server.Port)
	fmt.Printf("Server listening on %s\n", addr)
	return s.engine.Run(addr)
}
