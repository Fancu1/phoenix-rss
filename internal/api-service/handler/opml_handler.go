package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/Fancu1/phoenix-rss/internal/api-service/core"
	"github.com/Fancu1/phoenix-rss/pkg/ierr"
	"github.com/Fancu1/phoenix-rss/pkg/logger"
)

const (
	maxOPMLFileSize = 10 << 20 // 10 MB max file size
)

type OPMLHandler struct {
	feedService core.FeedServiceInterface
	opmlService *core.OPMLService
	cache       redis.Cmdable
}

func NewOPMLHandler(feedService core.FeedServiceInterface, cache redis.Cmdable) *OPMLHandler {
	return &OPMLHandler{
		feedService: feedService,
		opmlService: core.NewOPMLService(),
		cache:       cache,
	}
}

func (h *OPMLHandler) ExportOPML(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.FromContext(ctx)

	userID, exists := GetUserIDFromContext(c)
	if !exists {
		log.Error("user not authenticated in protected route")
		c.Error(ierr.ErrUnauthorized)
		return
	}

	log.Info("user requesting OPML export", "user_id", userID)

	feeds, err := h.feedService.ListUserFeeds(ctx, userID)
	if err != nil {
		log.Error("failed to list user feeds for export", "user_id", userID, "error", err.Error())
		c.Error(err)
		return
	}

	username := fmt.Sprintf("user_%d", userID) // TODO: get actual username
	opmlData, err := h.opmlService.GenerateOPML(feeds, username)
	if err != nil {
		log.Error("failed to generate OPML", "user_id", userID, "error", err.Error())
		c.Error(ierr.NewInternalError(errors.New("failed to generate OPML export")))
		return
	}

	filename := fmt.Sprintf("phoenix-rss-subscriptions-%s.opml", time.Now().Format("2006-01-02"))

	log.Info("successfully exported OPML", "user_id", userID, "feed_count", len(feeds))

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "application/xml")
	c.Data(http.StatusOK, "application/xml", opmlData)
}

type PreviewImportRequest struct {
	ToImport   []core.OPMLFeedItem `json:"to_import"`
	Duplicates []core.OPMLFeedItem `json:"duplicates"`
	Total      int                 `json:"total"`
}

func (h *OPMLHandler) PreviewOPML(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.FromContext(ctx)

	userID, exists := GetUserIDFromContext(c)
	if !exists {
		log.Error("user not authenticated in protected route")
		c.Error(ierr.ErrUnauthorized)
		return
	}

	log.Info("user requesting OPML import preview", "user_id", userID)

	file, err := c.FormFile("file")
	if err != nil {
		log.Warn("no file provided for OPML import", "error", err.Error())
		c.Error(ierr.NewValidationError("no file provided"))
		return
	}

	if file.Size > maxOPMLFileSize {
		log.Warn("OPML file too large", "size", file.Size)
		c.Error(ierr.NewValidationError("file size exceeds maximum allowed (10MB)"))
		return
	}

	f, err := file.Open()
	if err != nil {
		log.Error("failed to open uploaded file", "error", err.Error())
		c.Error(ierr.NewInternalError(err))
		return
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		log.Error("failed to read uploaded file", "error", err.Error())
		c.Error(ierr.NewInternalError(err))
		return
	}

	parseResult, err := h.opmlService.ParseOPML(data)
	if err != nil {
		log.Warn("failed to parse OPML file", "error", err.Error())
		c.Error(ierr.NewValidationError("invalid OPML file format"))
		return
	}

	existingFeeds, err := h.feedService.ListUserFeeds(ctx, userID)
	if err != nil {
		log.Error("failed to list existing feeds", "user_id", userID, "error", err.Error())
		c.Error(err)
		return
	}

	toImport, duplicates := h.opmlService.FilterDuplicates(parseResult.Feeds, existingFeeds)

	log.Info("OPML preview generated",
		"user_id", userID,
		"total_parsed", parseResult.Total,
		"to_import", len(toImport),
		"duplicates", len(duplicates))

	c.JSON(http.StatusOK, PreviewImportRequest{
		ToImport:   toImport,
		Duplicates: duplicates,
		Total:      parseResult.Total,
	})
}

type ImportOPMLRequest struct {
	Feeds []core.OPMLFeedItem `json:"feeds" binding:"required"`
}

func (h *OPMLHandler) ImportOPML(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.FromContext(ctx)

	userID, exists := GetUserIDFromContext(c)
	if !exists {
		log.Error("user not authenticated in protected route")
		c.Error(ierr.ErrUnauthorized)
		return
	}

	var req ImportOPMLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("invalid import request", "error", err.Error())
		c.Error(ierr.NewValidationError("invalid request body"))
		return
	}

	if len(req.Feeds) == 0 {
		log.Warn("no feeds to import")
		c.Error(ierr.NewValidationError("no feeds to import"))
		return
	}

	log.Info("user importing feeds from OPML", "user_id", userID, "feed_count", len(req.Feeds))

	urls := make([]string, len(req.Feeds))
	for i, feedItem := range req.Feeds {
		urls[i] = feedItem.URL
	}

	results, imported, failed, err := h.feedService.BatchSubscribeToFeeds(ctx, userID, urls)
	if err != nil {
		log.Error("batch subscribe failed", "user_id", userID, "error", err.Error())
		c.Error(err)
		return
	}

	result := core.OPMLImportResult{
		Imported:   imported,
		Failed:     failed,
		SkippedIDs: make([]string, 0),
		FailedIDs:  make([]string, 0),
	}

	for _, r := range results {
		if !r.Success && r.Error != "" {
			if r.Error == "already subscribed" {
				result.Skipped++
				result.SkippedIDs = append(result.SkippedIDs, r.URL)
			} else {
				result.FailedIDs = append(result.FailedIDs, r.URL)
			}
		}
	}

	if imported > 0 {
		h.invalidateUserFeedsCache(ctx, userID)
	}

	log.Info("OPML import completed",
		"user_id", userID,
		"imported", result.Imported,
		"skipped", result.Skipped,
		"failed", result.Failed)

	c.JSON(http.StatusOK, result)
}

func (h *OPMLHandler) invalidateUserFeedsCache(ctx context.Context, userID uint) {
	if h.cache == nil {
		return
	}

	cacheKey := fmt.Sprintf(userFeedsCacheKeyPattern, userID)
	if err := h.cache.Del(ctx, cacheKey).Err(); err != nil && err != redis.Nil {
		logger.FromContext(ctx).Warn("failed to invalidate user feeds cache", "user_id", userID, "error", err.Error())
	}
}

