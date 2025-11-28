package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/Fancu1/phoenix-rss/internal/api-service/core"
	"github.com/Fancu1/phoenix-rss/internal/feed-service/models"
	"github.com/Fancu1/phoenix-rss/pkg/ierr"
	"github.com/Fancu1/phoenix-rss/pkg/logger"
)

type FeedHandler struct {
	feedService core.FeedServiceInterface
	cache       redis.Cmdable
}

func NewFeedHandler(feedService core.FeedServiceInterface, cache redis.Cmdable) *FeedHandler {
	return &FeedHandler{
		feedService: feedService,
		cache:       cache,
	}
}

const (
	userFeedsCacheKeyPattern = "user:%d:feeds"
	userFeedsCacheTTL        = 15 * time.Minute
)

type AddFeedRequest struct {
	URL string `json:"url" binding:"required,url"`
}

// AddFeed now creates a subscription between the authenticated user and the feed
func (h *FeedHandler) AddFeed(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.FromContext(ctx)

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

	feed, err := h.feedService.SubscribeToFeed(ctx, userID, req.URL)
	if err != nil {
		log.Error("failed to subscribe to feed", "user_id", userID, "feed_url", req.URL, "error", err.Error())
		c.Error(err)
		return
	}

	h.invalidateUserFeedsCache(ctx, userID)

	log.Info("user successfully subscribed to feed", "user_id", userID, "feed_id", feed.ID, "feed_url", req.URL)
	c.JSON(http.StatusCreated, feed)
}

// ListFeeds now returns only feeds subscribed by the authenticated user
func (h *FeedHandler) ListFeeds(c *gin.Context) {
	// Get contextual logger for this request
	ctx := c.Request.Context()
	log := logger.FromContext(ctx)

	// Get user ID from context
	userID, exists := GetUserIDFromContext(c)
	if !exists {
		log.Error("user not authenticated in protected route")
		c.Error(ierr.ErrUnauthorized)
		return
	}

	log.Info("user requesting feed list", "user_id", userID)

	if cachedFeeds, ok := h.getCachedUserFeeds(ctx, userID); ok {
		log.Info("user feeds cache hit", "user_id", userID, "feed_count", len(cachedFeeds))
		c.JSON(http.StatusOK, cachedFeeds)
		return
	}

	feeds, err := h.feedService.ListUserFeeds(ctx, userID)
	if err != nil {
		log.Error("failed to list user feeds", "user_id", userID, "error", err.Error())
		c.Error(err)
		return
	}

	h.setCachedUserFeeds(ctx, userID, feeds)

	log.Info("successfully retrieved user feeds", "user_id", userID, "feed_count", len(feeds))
	c.JSON(http.StatusOK, feeds)
}

// UnsubscribeFeed removes a subscription between user and feed
func (h *FeedHandler) UnsubscribeFeed(c *gin.Context) {
	// Get contextual logger for this request
	ctx := c.Request.Context()
	log := logger.FromContext(ctx)

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

	err = h.feedService.UnsubscribeFromFeed(ctx, userID, uint(feedID))
	if err != nil {
		log.Error("failed to unsubscribe from feed", "user_id", userID, "feed_id", feedID, "error", err.Error())
		c.Error(err)
		return
	}

	h.invalidateUserFeedsCache(ctx, userID)

	log.Info("user successfully unsubscribed from feed", "user_id", userID, "feed_id", feedID)
	c.JSON(http.StatusOK, gin.H{"message": "successfully unsubscribed from feed"})
}

type UpdateFeedRequest struct {
	CustomTitle *string `json:"custom_title"`
}

func (h *FeedHandler) UpdateFeed(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.FromContext(ctx)

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

	var req UpdateFeedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("invalid request payload", "error", err.Error())
		c.Error(ierr.NewValidationError(err.Error()))
		return
	}

	log.Info("user updating feed subscription", "user_id", userID, "feed_id", feedID)

	feed, err := h.feedService.UpdateFeedCustomTitle(ctx, userID, uint(feedID), req.CustomTitle)
	if err != nil {
		log.Error("failed to update feed subscription", "user_id", userID, "feed_id", feedID, "error", err.Error())
		c.Error(err)
		return
	}

	h.invalidateUserFeedsCache(ctx, userID)

	log.Info("user successfully updated feed subscription", "user_id", userID, "feed_id", feedID)
	c.JSON(http.StatusOK, feed)
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

func (h *FeedHandler) cacheKeyForUserFeeds(userID uint) string {
	return fmt.Sprintf(userFeedsCacheKeyPattern, userID)
}

func (h *FeedHandler) getCachedUserFeeds(ctx context.Context, userID uint) ([]*models.UserFeed, bool) {
	if h.cache == nil {
		return nil, false
	}

	cacheKey := h.cacheKeyForUserFeeds(userID)
	result, err := h.cache.Get(ctx, cacheKey).Result()
	if err != nil {
		if err != redis.Nil {
			logger.FromContext(ctx).Warn("failed to fetch user feeds cache", "user_id", userID, "error", err.Error())
		}
		return nil, false
	}

	var feeds []*models.UserFeed
	if err := json.Unmarshal([]byte(result), &feeds); err != nil {
		logger.FromContext(ctx).Warn("failed to decode user feeds cache", "user_id", userID, "error", err.Error())
		return nil, false
	}

	return feeds, true
}

func (h *FeedHandler) setCachedUserFeeds(ctx context.Context, userID uint, feeds []*models.UserFeed) {
	if h.cache == nil {
		return
	}

	cacheKey := h.cacheKeyForUserFeeds(userID)
	payload, err := json.Marshal(feeds)
	if err != nil {
		logger.FromContext(ctx).Warn("failed to encode user feeds cache", "user_id", userID, "error", err.Error())
		return
	}

	if err := h.cache.Set(ctx, cacheKey, payload, userFeedsCacheTTL).Err(); err != nil {
		logger.FromContext(ctx).Warn("failed to store user feeds cache", "user_id", userID, "error", err.Error())
	}
}

func (h *FeedHandler) invalidateUserFeedsCache(ctx context.Context, userID uint) {
	if h.cache == nil {
		return
	}

	cacheKey := h.cacheKeyForUserFeeds(userID)
	if err := h.cache.Del(ctx, cacheKey).Err(); err != nil && err != redis.Nil {
		logger.FromContext(ctx).Warn("failed to invalidate user feeds cache", "user_id", userID, "error", err.Error())
	}
}
