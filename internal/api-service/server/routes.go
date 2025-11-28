package server

import (
	"github.com/gin-contrib/gzip"

	"github.com/Fancu1/phoenix-rss/internal/api-service/handler"
	"github.com/Fancu1/phoenix-rss/pkg/ierr"
	"github.com/Fancu1/phoenix-rss/pkg/logger"
)

func (s *Server) setupRoutes() {
	// Apply global middleware
	s.engine.Use(handler.RequestIDMiddleware())
	s.engine.Use(logger.GinLoggingMiddleware())
	s.engine.Use(gzip.Gzip(gzip.DefaultCompression))
	s.engine.Use(ierr.ErrorHandlerMiddleware())

	// Register frontend routes
	s.frontendHandler.RegisterRoutes(s.engine)

	// Register API v1 routes
	apiV1 := s.engine.Group("/api/v1")
	{
		// Public routes (no authentication required)
		apiV1.GET("/health", handler.HealthCheck)

		// Authentication routes
		apiV1.POST("/users/register", s.userHandler.Register)
		apiV1.POST("/users/login", s.userHandler.Login)

		// Protected routes (authentication required)
		protected := apiV1.Group("")
		protected.Use(s.authMiddleware.RequireAuth())
		{
			// Feed management (user-specific)
			protected.GET("/feeds", s.feedHandler.ListFeeds)
			protected.POST("/feeds", s.feedHandler.AddFeed)

			// OPML import/export (must be before :feed_id routes)
			protected.GET("/feeds/export", s.opmlHandler.ExportOPML)
			protected.POST("/feeds/import/preview", s.opmlHandler.PreviewOPML)
			protected.POST("/feeds/import", s.opmlHandler.ImportOPML)

			// Feed-specific routes (with :feed_id parameter)
			protected.DELETE("/feeds/:feed_id", s.feedHandler.UnsubscribeFeed)
			protected.PATCH("/feeds/:feed_id", s.feedHandler.UpdateFeed)
			protected.POST("/feeds/:feed_id/fetch", s.articleHandler.TriggerFetch)
			protected.GET("/feeds/:feed_id/articles", s.articleHandler.ListArticles)

			// Article access (user-specific)
			protected.GET("/articles/:article_id", s.articleHandler.GetArticle)
		}
	}
}
