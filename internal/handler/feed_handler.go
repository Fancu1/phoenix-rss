package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Fancu1/phoenix-rss/internal/core"
	"github.com/Fancu1/phoenix-rss/pkg/ierr"
	"github.com/Fancu1/phoenix-rss/pkg/logger"
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
	log := logger.FromContext(c.Request.Context())

	var req AddFeedRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("invalid request payload", "error", err.Error())
		c.Error(ierr.NewValidationError(err.Error()))
		return
	}

	userID, exists := GetUserIDFromContext(c)
	if !exists {
		log.Error("user not authenticated in protected route")
		c.Error(ierr.ErrUnauthorized)
		return
	}

	log.Info("user attempting to subscribe to feed", "user_id", userID, "feed_url", req.URL)

	feed, err := h.feedService.SubscribeToFeed(c.Request.Context(), userID, req.URL)
	if err != nil {
		log.Error("failed to subscribe to feed", "user_id", userID, "feed_url", req.URL, "error", err.Error())
		c.Error(err)
		return
	}

	log.Info("user successfully subscribed to feed", "user_id", userID, "feed_id", feed.ID, "feed_url", req.URL)
	c.JSON(http.StatusCreated, feed)
}

// ListFeeds now returns only feeds subscribed by the authenticated user
func (h *FeedHandler) ListFeeds(c *gin.Context) {
	// Get contextual logger for this request
	log := logger.FromContext(c.Request.Context())

	// Get user ID from context
	userID, exists := GetUserIDFromContext(c)
	if !exists {
		log.Error("user not authenticated in protected route")
		c.Error(ierr.ErrUnauthorized)
		return
	}

	log.Info("user requesting feed list", "user_id", userID)

	feeds, err := h.feedService.ListUserFeeds(c.Request.Context(), userID)
	if err != nil {
		log.Error("failed to list user feeds", "user_id", userID, "error", err.Error())
		c.Error(err)
		return
	}

	log.Info("successfully retrieved user feeds", "user_id", userID, "feed_count", len(feeds))
	c.JSON(http.StatusOK, feeds)
}

// UnsubscribeFeed removes a subscription between user and feed
func (h *FeedHandler) UnsubscribeFeed(c *gin.Context) {
	// Get contextual logger for this request
	log := logger.FromContext(c.Request.Context())

	// Get user ID from context
	userID, exists := GetUserIDFromContext(c)
	if !exists {
		log.Error("user not authenticated in protected route")
		c.Error(ierr.ErrUnauthorized)
		return
	}

	// Get feed ID from URL parameter
	feedIDStr := c.Param("feed_id")
	feedID, err := strconv.ParseUint(feedIDStr, 10, 32)
	if err != nil {
		log.Warn("invalid feed ID parameter", "feed_id_str", feedIDStr, "error", err.Error())
		c.Error(ierr.ErrInvalidFeedID)
		return
	}

	log.Info("user attempting to unsubscribe from feed", "user_id", userID, "feed_id", feedID)

	err = h.feedService.UnsubscribeFromFeed(c.Request.Context(), userID, uint(feedID))
	if err != nil {
		log.Error("failed to unsubscribe from feed", "user_id", userID, "feed_id", feedID, "error", err.Error())
		c.Error(err)
		return
	}

	log.Info("user successfully unsubscribed from feed", "user_id", userID, "feed_id", feedID)
	c.JSON(http.StatusOK, gin.H{"message": "successfully unsubscribed from feed"})
}

// Keep the old method for backward compatibility (will be deprecated)
func (h *FeedHandler) ListAllFeeds(c *gin.Context) {
	// Get contextual logger for this request
	log := logger.FromContext(c.Request.Context())

	log.Info("listing all feeds (deprecated endpoint)")

	feeds, err := h.feedService.ListAllFeeds(c.Request.Context())
	if err != nil {
		log.Error("failed to list all feeds", "error", err.Error())
		c.Error(err)
		return
	}

	log.Info("successfully retrieved all feeds", "feed_count", len(feeds))
	c.JSON(http.StatusOK, feeds)
}
