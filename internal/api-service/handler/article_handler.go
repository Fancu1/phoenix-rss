package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Fancu1/phoenix-rss/internal/api-service/core"
	"github.com/Fancu1/phoenix-rss/internal/api-service/repository"
	"github.com/Fancu1/phoenix-rss/internal/feed-service/models"
	"github.com/Fancu1/phoenix-rss/pkg/ierr"
	"github.com/Fancu1/phoenix-rss/pkg/logger"
)

// PaginationMeta contains pagination metadata for list responses
type PaginationMeta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// ArticleListResponse is the paginated response for article listings
type ArticleListResponse struct {
	Items      []*models.Article `json:"items"`
	Pagination PaginationMeta    `json:"pagination"`
}

type ArticleHandler struct {
	service          core.ArticleServiceInterface
	subscriptionRepo *repository.SubscriptionRepository
	articleRepo      *repository.ArticleRepository
}

func NewArticleHandler(service core.ArticleServiceInterface, subscriptionRepo *repository.SubscriptionRepository, articleRepo *repository.ArticleRepository) *ArticleHandler {
	return &ArticleHandler{
		service:          service,
		subscriptionRepo: subscriptionRepo,
		articleRepo:      articleRepo,
	}
}

func (h *ArticleHandler) TriggerFetch(c *gin.Context) {
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

	if err := h.service.TriggerFetch(ctx, userID, uint(feedID)); err != nil {
		log.Error("failed to trigger feed fetch", "user_id", userID, "feed_id", feedID, "error", err.Error())
		c.Error(err)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"message": "Feed fetch job accepted"})
}

func (h *ArticleHandler) ListArticles(c *gin.Context) {
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

	// Parse pagination parameters from query string
	page := parseIntQueryParam(c, "page", 1)
	pageSize := parseIntQueryParam(c, "page_size", repository.DefaultPageSize)

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

	articles, total, err := h.articleRepo.ListByFeedIDPaginated(ctx, uint(feedID), page, pageSize)
	if err != nil {
		log.Error("failed to list articles", "feed_id", feedID, "page", page, "error", err.Error())
		c.Error(ierr.NewDatabaseError(err))
		return
	}

	// Normalize page/pageSize in response (repo may have adjusted invalid values)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > repository.MaxPageSize {
		pageSize = repository.DefaultPageSize
	}

	c.JSON(http.StatusOK, ArticleListResponse{
		Items: articles,
		Pagination: PaginationMeta{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: calculateTotalPages(total, pageSize),
		},
	})
}

// parseIntQueryParam extracts an integer query parameter with a fallback default
func parseIntQueryParam(c *gin.Context, key string, defaultVal int) int {
	valStr := c.Query(key)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}

// calculateTotalPages computes the number of pages needed for a given total and page size
func calculateTotalPages(total int64, pageSize int) int {
	if pageSize <= 0 {
		return 0
	}
	pages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		pages++
	}
	return pages
}

func (h *ArticleHandler) GetArticle(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.FromContext(ctx)

	userID, exists := GetUserIDFromContext(c)
	if !exists {
		c.Error(ierr.ErrUnauthorized)
		return
	}

	articleID, err := strconv.ParseUint(c.Param("article_id"), 10, 32)
	if err != nil {
		c.Error(ierr.NewValidationError("invalid article ID"))
		return
	}

	feedID, err := h.articleRepo.GetFeedID(ctx, uint(articleID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.Error(ierr.ErrArticleNotFound)
			return
		}
		log.Error("failed to get article feed_id", "article_id", articleID, "error", err.Error())
		c.Error(ierr.NewDatabaseError(err))
		return
	}

	subscribed, err := h.subscriptionRepo.IsUserSubscribed(ctx, userID, feedID)
	if err != nil {
		log.Error("failed to check subscription", "user_id", userID, "feed_id", feedID, "error", err.Error())
		c.Error(ierr.NewDatabaseError(err))
		return
	}
	if !subscribed {
		c.Error(ierr.ErrNotSubscribed)
		return
	}

	article, err := h.articleRepo.GetByID(ctx, uint(articleID))
	if err != nil {
		log.Error("failed to get article", "article_id", articleID, "error", err.Error())
		c.Error(ierr.NewDatabaseError(err))
		return
	}

	c.JSON(http.StatusOK, article)
}
