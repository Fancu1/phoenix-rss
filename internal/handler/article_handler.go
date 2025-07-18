package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"

	"github.com/Fancu1/phoenix-rss/internal/core"
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
	// Get user ID from context
	userID, exists := GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	feedIDStr := c.Param("feed_id")
	feedID, err := strconv.ParseUint(feedIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feed ID"})
		return
	}

	// Check if user is subscribed to this feed
	isSubscribed, err := h.feedRepo.IsUserSubscribed(userID, uint(feedID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check subscription"})
		return
	}
	if !isSubscribed {
		c.JSON(http.StatusForbidden, gin.H{"error": "not subscribed to this feed"})
		return
	}

	task, err := tasks.NewFeedFetchTask(uint(feedID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	info, err := h.taskClient.EnqueueContext(c.Request.Context(), task, asynq.MaxRetry(3))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("feed fetch job enqueued", "feed_id", feedID, "user_id", userID, "task_id", info.ID)

	c.JSON(http.StatusAccepted, gin.H{"message": "Feed fetch job accepted", "task_id": info.ID})
}

func (h *ArticleHandler) ListArticles(c *gin.Context) {
	// Get user ID from context
	userID, exists := GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	feedIDStr := c.Param("feed_id")
	feedID, err := strconv.ParseUint(feedIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feed ID"})
		return
	}

	// Check if user is subscribed to this feed
	isSubscribed, err := h.feedRepo.IsUserSubscribed(userID, uint(feedID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check subscription"})
		return
	}
	if !isSubscribed {
		c.JSON(http.StatusForbidden, gin.H{"error": "not subscribed to this feed"})
		return
	}

	articles, err := h.service.ListArticlesByFeedID(uint(feedID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, articles)
}
