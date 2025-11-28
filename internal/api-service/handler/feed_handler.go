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
	"github.com/Fancu1/phoenix-rss/internal/api-service/repository"
	"github.com/Fancu1/phoenix-rss/internal/feed-service/models"
	"github.com/Fancu1/phoenix-rss/pkg/ierr"
	"github.com/Fancu1/phoenix-rss/pkg/logger"
)

type FeedHandler struct {
	feedService      core.FeedServiceInterface
	subscriptionRepo *repository.SubscriptionRepository
	cache            redis.Cmdable
}

func NewFeedHandler(feedService core.FeedServiceInterface, subscriptionRepo *repository.SubscriptionRepository, cache redis.Cmdable) *FeedHandler {
	return &FeedHandler{
		feedService:      feedService,
		subscriptionRepo: subscriptionRepo,
		cache:            cache,
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

func (h *FeedHandler) ListFeeds(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.FromContext(ctx)

	userID, exists := GetUserIDFromContext(c)
	if !exists {
		c.Error(ierr.ErrUnauthorized)
		return
	}

	if cachedFeeds, ok := h.getCachedUserFeeds(ctx, userID); ok {
		c.JSON(http.StatusOK, cachedFeeds)
		return
	}

	feeds, err := h.subscriptionRepo.ListUserFeeds(ctx, userID)
	if err != nil {
		log.Error("failed to list user feeds", "user_id", userID, "error", err.Error())
		c.Error(ierr.NewDatabaseError(err))
		return
	}

	h.setCachedUserFeeds(ctx, userID, feeds)
	c.JSON(http.StatusOK, feeds)
}

func (h *FeedHandler) UnsubscribeFeed(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.FromContext(ctx)

	userID, exists := GetUserIDFromContext(c)
	if !exists {
		c.Error(ierr.ErrUnauthorized)
		return
	}

	feedID, err := strconv.ParseUint(c.Param("feed_id"), 10, 32)
	if err != nil {
		c.Error(ierr.ErrInvalidFeedID)
		return
	}

	subscribed, err := h.subscriptionRepo.IsUserSubscribed(ctx, userID, uint(feedID))
	if err != nil {
		log.Error("failed to check subscription", "user_id", userID, "feed_id", feedID, "error", err.Error())
		c.Error(ierr.NewDatabaseError(err))
		return
	}
	if !subscribed {
		c.Error(ierr.ErrNotSubscribed)
		return
	}

	if err := h.subscriptionRepo.Delete(ctx, userID, uint(feedID)); err != nil {
		log.Error("failed to delete subscription", "user_id", userID, "feed_id", feedID, "error", err.Error())
		c.Error(ierr.NewDatabaseError(err))
		return
	}

	h.invalidateUserFeedsCache(ctx, userID)
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
		c.Error(ierr.ErrUnauthorized)
		return
	}

	feedID, err := strconv.ParseUint(c.Param("feed_id"), 10, 32)
	if err != nil {
		c.Error(ierr.ErrInvalidFeedID)
		return
	}

	var req UpdateFeedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(ierr.NewValidationError(err.Error()))
		return
	}

	subscribed, err := h.subscriptionRepo.IsUserSubscribed(ctx, userID, uint(feedID))
	if err != nil {
		log.Error("failed to check subscription", "user_id", userID, "feed_id", feedID, "error", err.Error())
		c.Error(ierr.NewDatabaseError(err))
		return
	}
	if !subscribed {
		c.Error(ierr.ErrNotSubscribed)
		return
	}

	if err := h.subscriptionRepo.UpdateCustomTitle(ctx, userID, uint(feedID), req.CustomTitle); err != nil {
		log.Error("failed to update custom title", "user_id", userID, "feed_id", feedID, "error", err.Error())
		c.Error(ierr.NewDatabaseError(err))
		return
	}

	sub, err := h.subscriptionRepo.GetWithFeed(ctx, userID, uint(feedID))
	if err != nil {
		log.Error("failed to get subscription", "user_id", userID, "feed_id", feedID, "error", err.Error())
		c.Error(ierr.NewDatabaseError(err))
		return
	}

	h.invalidateUserFeedsCache(ctx, userID)
	c.JSON(http.StatusOK, &models.UserFeed{
		Feed:        sub.Feed,
		CustomTitle: sub.CustomTitle,
	})
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
