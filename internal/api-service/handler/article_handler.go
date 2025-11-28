package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Fancu1/phoenix-rss/internal/api-service/core"
	"github.com/Fancu1/phoenix-rss/internal/api-service/repository"
	"github.com/Fancu1/phoenix-rss/pkg/ierr"
	"github.com/Fancu1/phoenix-rss/pkg/logger"
)

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

	articles, err := h.articleRepo.ListByFeedID(ctx, uint(feedID))
	if err != nil {
		log.Error("failed to list articles", "feed_id", feedID, "error", err.Error())
		c.Error(ierr.NewDatabaseError(err))
		return
	}

	c.JSON(http.StatusOK, articles)
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
