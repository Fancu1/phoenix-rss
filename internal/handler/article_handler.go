package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"

	"github.com/Fancu1/phoenix-rss/internal/core"
	"github.com/Fancu1/phoenix-rss/internal/tasks"
)

type ArticleHandler struct {
	logger     *slog.Logger
	taskClient *asynq.Client
	service    *core.ArticleService
}

func NewArticleHandler(logger *slog.Logger, taskClient *asynq.Client, articleService *core.ArticleService) *ArticleHandler {
	return &ArticleHandler{
		logger:     logger,
		taskClient: taskClient,
		service:    articleService,
	}
}

func (h *ArticleHandler) TriggerFetch(c *gin.Context) {
	feedID := c.Param("feed_id")
	id, err := strconv.ParseUint(feedID, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feed ID"})
		return
	}

	task, err := tasks.NewFeedFetchTask(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	info, err := h.taskClient.EnqueueContext(c.Request.Context(), task)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("feed fetch job enqueued", "feed_id", id, "task_id", info.ID)

	c.JSON(http.StatusAccepted, gin.H{"message": "Feed fetch job accepted", "task_id": info.ID})
}

func (h *ArticleHandler) ListArticles(c *gin.Context) {
	feedID := c.Param("feed_id")
	id, err := strconv.ParseUint(feedID, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feed ID"})
		return
	}

	articles, err := h.service.ListArticlesByFeedID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}

	c.JSON(http.StatusOK, articles)
}
