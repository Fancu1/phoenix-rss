package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"

	"github.com/Fancu1/phoenix-rss/internal/core"
	"github.com/Fancu1/phoenix-rss/internal/logger"
	"github.com/Fancu1/phoenix-rss/internal/repository"
	"github.com/Fancu1/phoenix-rss/internal/tasks"
)

type ArticleHandler struct {
	logger     *slog.Logger
	taskClient *asynq.Client
	service    core.ArticleServiceInterface
	feedRepo   *repository.FeedRepository
}

func NewArticleHandler(logger *slog.Logger, taskClient *asynq.Client, articleService core.ArticleServiceInterface, feedRepo *repository.FeedRepository) *ArticleHandler {
	return &ArticleHandler{
		logger:     logger,
		taskClient: taskClient,
		service:    articleService,
		feedRepo:   feedRepo,
	}
}

func (h *ArticleHandler) TriggerFetch(c *gin.Context) {
	// Get contextual logger for this request
	log := logger.FromContext(c.Request.Context())

	// Get user ID from context
	userID, exists := GetUserIDFromContext(c)
	if !exists {
		log.Error("user not authenticated in protected route")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	feedIDStr := c.Param("feed_id")
	feedID, err := strconv.ParseUint(feedIDStr, 10, 32)
	if err != nil {
		log.Warn("invalid feed ID parameter", "feed_id_str", feedIDStr, "error", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feed ID"})
		return
	}

	log.Info("user triggering feed fetch", "user_id", userID, "feed_id", feedID)

	// Check if user is subscribed to this feed
	isSubscribed, err := h.feedRepo.IsUserSubscribed(c.Request.Context(), userID, uint(feedID))
	if err != nil {
		log.Error("failed to check subscription", "user_id", userID, "feed_id", feedID, "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check subscription"})
		return
	}

	if !isSubscribed {
		log.Warn("user not subscribed to feed", "user_id", userID, "feed_id", feedID)
		c.JSON(http.StatusForbidden, gin.H{"error": "not subscribed to this feed"})
		return
	}

	task, err := tasks.NewFeedFetchTask(uint(feedID))
	if err != nil {
		log.Error("failed to create fetch task", "feed_id", feedID, "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	info, err := h.taskClient.EnqueueContext(c.Request.Context(), task, asynq.MaxRetry(3))
	if err != nil {
		log.Error("failed to enqueue fetch task", "feed_id", feedID, "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Info("feed fetch job enqueued successfully", "feed_id", feedID, "user_id", userID, "task_id", info.ID)

	c.JSON(http.StatusAccepted, gin.H{"message": "Feed fetch job accepted", "task_id": info.ID})
}

func (h *ArticleHandler) ListArticles(c *gin.Context) {
	// Get contextual logger for this request
	log := logger.FromContext(c.Request.Context())

	// Get user ID from context
	userID, exists := GetUserIDFromContext(c)
	if !exists {
		log.Error("user not authenticated in protected route")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	feedIDStr := c.Param("feed_id")
	feedID, err := strconv.ParseUint(feedIDStr, 10, 32)
	if err != nil {
		log.Warn("invalid feed ID parameter", "feed_id_str", feedIDStr, "error", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feed ID"})
		return
	}

	log.Info("user requesting articles", "user_id", userID, "feed_id", feedID)

	// Check if user is subscribed to this feed
	isSubscribed, err := h.feedRepo.IsUserSubscribed(c.Request.Context(), userID, uint(feedID))
	if err != nil {
		log.Error("failed to check subscription", "user_id", userID, "feed_id", feedID, "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check subscription"})
		return
	}

	if !isSubscribed {
		log.Warn("user not subscribed to feed", "user_id", userID, "feed_id", feedID)
		c.JSON(http.StatusForbidden, gin.H{"error": "not subscribed to this feed"})
		return
	}

	articles, err := h.service.ListArticlesByFeedID(c.Request.Context(), uint(feedID))
	if err != nil {
		log.Error("failed to list articles", "user_id", userID, "feed_id", feedID, "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Info("successfully retrieved articles", "user_id", userID, "feed_id", feedID, "article_count", len(articles))
	c.JSON(http.StatusOK, articles)
}
