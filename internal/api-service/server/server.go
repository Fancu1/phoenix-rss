package server

import (
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/Fancu1/phoenix-rss/internal/api-service/core"
	"github.com/Fancu1/phoenix-rss/internal/api-service/handler"
	"github.com/Fancu1/phoenix-rss/internal/config"
)

type Server struct {
	config          *config.Config
	engine          *gin.Engine
	logger          *slog.Logger
	feedHandler     *handler.FeedHandler
	articleHandler  *handler.ArticleHandler
	userHandler     *handler.UserHandler
	authMiddleware  *handler.AuthMiddleware
	frontendHandler *handler.StaticFrontendHandler
}

func New(cfg *config.Config, logger *slog.Logger, feedService core.FeedServiceInterface, articleService core.ArticleServiceInterface, userService core.UserServiceInterface, redisClient *redis.Client, staticFS fs.FS) (*Server, error) {
	feedHandler := handler.NewFeedHandler(feedService)
	articleHandler := handler.NewArticleHandler(logger, articleService)
	userHandler := handler.NewUserHandler(userService)
	authMiddleware := handler.NewAuthMiddleware(userService, redisClient)
	frontendHandler, err := handler.NewStaticFrontendHandler(staticFS)
	if err != nil {
		return nil, fmt.Errorf("failed to create frontend handler: %w", err)
	}

	s := &Server{
		config:          cfg,
		engine:          gin.Default(),
		feedHandler:     feedHandler,
		articleHandler:  articleHandler,
		userHandler:     userHandler,
		authMiddleware:  authMiddleware,
		frontendHandler: frontendHandler,
	}

	s.setupRoutes()

	return s, nil
}

func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.config.Server.Port)
	fmt.Printf("Server listening on %s\n", addr)
	return s.engine.Run(addr)
}
