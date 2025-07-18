package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Fancu1/phoenix-rss/internal/core"
)

type FeedHandler struct {
	feedService core.FeedServiceInterface
}

func NewFeedHandler(feedService core.FeedServiceInterface) *FeedHandler {
	return &FeedHandler{feedService: feedService}
}

type AddFeedRequest struct {
	URL string `json:"url" binding:"required,url"`
}

// AddFeed now creates a subscription between the authenticated user and the feed
func (h *FeedHandler) AddFeed(c *gin.Context) {
	var req AddFeedRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from context
	userID, exists := GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	feed, err := h.feedService.SubscribeToFeed(c.Request.Context(), userID, req.URL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, feed)
}

// ListFeeds now returns only feeds subscribed by the authenticated user
func (h *FeedHandler) ListFeeds(c *gin.Context) {
	// Get user ID from context
	userID, exists := GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	feeds, err := h.feedService.ListUserFeeds(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, feeds)
}

// UnsubscribeFeed removes a subscription between user and feed
func (h *FeedHandler) UnsubscribeFeed(c *gin.Context) {
	// Get user ID from context
	userID, exists := GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	// Get feed ID from URL parameter
	feedIDStr := c.Param("feed_id")
	feedID, err := strconv.ParseUint(feedIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feed ID"})
		return
	}

	err = h.feedService.UnsubscribeFromFeed(userID, uint(feedID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "successfully unsubscribed from feed"})
}

// Keep the old method for backward compatibility (will be deprecated)
func (h *FeedHandler) ListAllFeeds(c *gin.Context) {
	feeds, err := h.feedService.ListAllFeeds()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, feeds)
}
