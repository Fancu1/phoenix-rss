package server

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/Fancu1/phoenix-rss/internal/config"
	"github.com/Fancu1/phoenix-rss/internal/core"
	"github.com/Fancu1/phoenix-rss/internal/handler"
	"github.com/Fancu1/phoenix-rss/internal/repository"
	"github.com/Fancu1/phoenix-rss/internal/worker"
)

type Server struct {
	config         *config.Config
	engine         *gin.Engine
	feedHandler    *handler.FeedHandler
	articleHandler *handler.ArticleHandler
}

func New(cfg *config.Config, dispatcher *worker.Dispatcher) *Server {
	feedRepo := repository.NewFeedRepository() // db options
	articleRepo := repository.NewArticleRepository()

	feedService := core.NewFeedService(feedRepo)
	articleService := core.NewArticleService(feedRepo, articleRepo)

	feedHandler := handler.NewFeedHandler(feedService)
	articleHandler := handler.NewArticleHandler(articleService, dispatcher)

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
