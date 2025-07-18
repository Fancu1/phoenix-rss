package server

import (
	"github.com/Fancu1/phoenix-rss/internal/handler"
)

func (s *Server) setupRoutes() {
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
			protected.DELETE("/feeds/:feed_id", s.feedHandler.UnsubscribeFeed)
			protected.POST("/feeds/:feed_id/fetch", s.articleHandler.TriggerFetch)

			// Article access (user-specific)
			protected.GET("/feeds/:feed_id/articles", s.articleHandler.ListArticles)
		}
	}
}
