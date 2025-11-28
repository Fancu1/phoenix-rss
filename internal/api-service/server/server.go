package server

import (
	"fmt"
	"io/fs"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/Fancu1/phoenix-rss/internal/api-service/core"
	"github.com/Fancu1/phoenix-rss/internal/api-service/handler"
	"github.com/Fancu1/phoenix-rss/internal/api-service/repository"
	"github.com/Fancu1/phoenix-rss/internal/config"
)

type Server struct {
	config          *config.Config
	engine          *gin.Engine
	feedHandler     *handler.FeedHandler
	articleHandler  *handler.ArticleHandler
	userHandler     *handler.UserHandler
	opmlHandler     *handler.OPMLHandler
	authMiddleware  *handler.AuthMiddleware
	frontendHandler *handler.StaticFrontendHandler
}

func New(cfg *config.Config, db *gorm.DB, feedService core.FeedServiceInterface, articleService core.ArticleServiceInterface, userService core.UserServiceInterface, redisClient *redis.Client, staticFS fs.FS) (*Server, error) {
	subscriptionRepo := repository.NewSubscriptionRepository(db)
	articleRepo := repository.NewArticleRepository(db)

	feedHandler := handler.NewFeedHandler(feedService, subscriptionRepo, redisClient)
	articleHandler := handler.NewArticleHandler(articleService, subscriptionRepo, articleRepo)
	userHandler := handler.NewUserHandler(userService)
	opmlHandler := handler.NewOPMLHandler(feedService, subscriptionRepo, redisClient)
	authMiddleware := handler.NewAuthMiddleware(cfg.Auth.JWTSecret)
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
		opmlHandler:     opmlHandler,
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
