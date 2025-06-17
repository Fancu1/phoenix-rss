package server

import (
	"github.com/Fancu1/phoenix-rss/internal/handler"
)

func (s *Server) setupRoutes() {
	apiV1 := s.engine.Group("/api/v1")
	{
		apiV1.GET("/health", handler.HealthCheck)

		// feeds
		apiV1.POST("/feeds", s.feedHandler.AddFeed)
	}
}
