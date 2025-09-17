package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Fancu1/phoenix-rss/internal/api-service/core"
	"github.com/Fancu1/phoenix-rss/pkg/ierr"
	"github.com/Fancu1/phoenix-rss/pkg/logger"
)

type ArticleHandler struct {
	logger  *slog.Logger
	service core.ArticleServiceInterface
}

func NewArticleHandler(logger *slog.Logger, articleService core.ArticleServiceInterface) *ArticleHandler {
	return &ArticleHandler{
		logger:  logger,
		service: articleService,
	}
}

func (h *ArticleHandler) TriggerFetch(c *gin.Context) {
	log := logger.FromContext(c.Request.Context())

	userID, exists := GetUserIDFromContext(c)
	if !exists {
		log.Error("user not authenticated in protected route")
		c.Error(ierr.ErrUnauthorized)
		return
	}

	feedIDStr := c.Param("feed_id")
	feedID, err := strconv.ParseUint(feedIDStr, 10, 32)
	if err != nil {
		log.Warn("invalid feed ID parameter", "feed_id_str", feedIDStr, "error", err.Error())
		c.Error(ierr.ErrInvalidFeedID)
		return
	}

	log.Info("user triggering feed fetch", "user_id", userID, "feed_id", feedID)

	if err := h.service.TriggerFetch(c.Request.Context(), userID, uint(feedID)); err != nil {
		log.Error("failed to trigger feed fetch", "user_id", userID, "feed_id", feedID, "error", err.Error())
		c.Error(err)
		return
	}

	log.Info("feed fetch triggered successfully", "feed_id", feedID, "user_id", userID)

	c.JSON(http.StatusAccepted, gin.H{"message": "Feed fetch job accepted"})
}

func (h *ArticleHandler) ListArticles(c *gin.Context) {
	// Get contextual logger for this request
	log := logger.FromContext(c.Request.Context())

	// Get user ID from context
	userID, exists := GetUserIDFromContext(c)
	if !exists {
		log.Error("user not authenticated in protected route")
		c.Error(ierr.ErrUnauthorized)
		return
	}

	feedIDStr := c.Param("feed_id")
	feedID, err := strconv.ParseUint(feedIDStr, 10, 32)
	if err != nil {
		log.Warn("invalid feed ID parameter", "feed_id_str", feedIDStr, "error", err.Error())
		c.Error(ierr.ErrInvalidFeedID)
		return
	}

	log.Info("user requesting articles", "user_id", userID, "feed_id", feedID)

	articles, err := h.service.ListArticlesByFeedID(c.Request.Context(), userID, uint(feedID))
	if err != nil {
		log.Error("failed to list articles", "user_id", userID, "feed_id", feedID, "error", err.Error())
		c.Error(err)
		return
	}

	log.Info("successfully retrieved articles", "user_id", userID, "feed_id", feedID, "article_count", len(articles))
	c.JSON(http.StatusOK, articles)
}

func (h *ArticleHandler) GetArticle(c *gin.Context) {
	log := logger.FromContext(c.Request.Context())

	userID, exists := GetUserIDFromContext(c)
	if !exists {
		log.Error("user not authenticated in protected route")
		c.Error(ierr.ErrUnauthorized)
		return
	}

	articleIDStr := c.Param("article_id")
	articleID, err := strconv.ParseUint(articleIDStr, 10, 32)
	if err != nil {
		log.Warn("invalid article ID parameter", "article_id_str", articleIDStr, "error", err.Error())
		c.Error(ierr.NewValidationError("Invalid article ID"))
		return
	}

	log.Info("user requesting article", "user_id", userID, "article_id", articleID)

	article, err := h.service.GetArticleByID(c.Request.Context(), userID, uint(articleID))
	if err != nil {
		log.Error("failed to get article", "user_id", userID, "article_id", articleID, "error", err.Error())
		c.Error(err)
		return
	}

	log.Info("successfully retrieved article", "user_id", userID, "article_id", articleID)
	c.JSON(http.StatusOK, article)
}
