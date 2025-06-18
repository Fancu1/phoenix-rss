package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Fancu1/phoenix-rss/internal/core"
)

type FeedHandler struct {
	feedService *core.FeedService
}

func NewFeedHandler(feedService *core.FeedService) *FeedHandler {
	return &FeedHandler{feedService: feedService}
}

type AddFeedRequest struct {
	URL string `json:"url" binding:"required,url"`
}

func (h *FeedHandler) AddFeed(c *gin.Context) {
	var req AddFeedRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	feed, err := h.feedService.AddFeedByURL(c.Request.Context(), req.URL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}

	c.JSON(http.StatusCreated, feed)
}

func (h *FeedHandler) ListAllFeeds(c *gin.Context) {
	feeds, err := h.feedService.ListAllFeeds()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}

	c.JSON(http.StatusOK, feeds)
}
