package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Fancu1/phoenix-rss/internal/core"
	"github.com/Fancu1/phoenix-rss/internal/worker"
)

type ArticleHandler struct {
	service    *core.ArticleService
	dispatcher *worker.Dispatcher
}

func NewArticleHandler(service *core.ArticleService, dispatcher *worker.Dispatcher) *ArticleHandler {
	return &ArticleHandler{
		service:    service,
		dispatcher: dispatcher,
	}
}

func (h *ArticleHandler) TriggerFetch(c *gin.Context) {
	feedID := c.Param("feed_id")
	id, err := strconv.ParseUint(feedID, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feed ID"})
		return
	}

	job := worker.Job{FeedID: uint(id)}
	h.dispatcher.AddJob(job)

	c.JSON(http.StatusAccepted, gin.H{"message": "Feed fetch job accepted"})
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
