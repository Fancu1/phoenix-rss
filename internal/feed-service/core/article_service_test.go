package core

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/Fancu1/phoenix-rss/internal/feed-service/models"
	"github.com/Fancu1/phoenix-rss/internal/feed-service/repository"
	"github.com/Fancu1/phoenix-rss/pkg/ierr"
	"github.com/Fancu1/phoenix-rss/pkg/logger"
)

func setupArticleService(t *testing.T) (*ArticleService, *repository.FeedRepository, *repository.ArticleRepository, *gorm.DB) {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(&models.Feed{}, &models.Article{}, &models.Subscription{}))

	feedRepo := repository.NewFeedRepository(db)
	articleRepo := repository.NewArticleRepository(db)

	service := NewArticleService(feedRepo, articleRepo, nil, logger.New(0))
	return service, feedRepo, articleRepo, db
}

func TestGetArticleByID_Success(t *testing.T) {
	service, _, articleRepo, db := setupArticleService(t)

	feed := &models.Feed{Title: "Feed", URL: "https://example.com", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, db.Create(feed).Error)

	subscription := &models.Subscription{UserID: 1, FeedID: feed.ID}
	require.NoError(t, db.Create(subscription).Error)

	article := &models.Article{
		FeedID:      feed.ID,
		Title:       "Article",
		URL:         "https://example.com/article",
		Description: "desc",
		Content:     "<p>content</p>",
		PublishedAt: time.Now(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	_, err := articleRepo.Create(context.Background(), article)
	require.NoError(t, err)

	got, err := service.GetArticleByID(context.Background(), 1, article.ID)
	require.NoError(t, err)
	require.Equal(t, article.ID, got.ID)
}

func TestGetArticleByID_NotSubscribed(t *testing.T) {
	service, _, articleRepo, db := setupArticleService(t)

	feed := &models.Feed{Title: "Feed", URL: "https://example.com", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, db.Create(feed).Error)

	article := &models.Article{
		FeedID:      feed.ID,
		Title:       "Article",
		URL:         "https://example.com/article",
		Description: "desc",
		Content:     "<p>content</p>",
		PublishedAt: time.Now(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	_, err := articleRepo.Create(context.Background(), article)
	require.NoError(t, err)

	_, err = service.GetArticleByID(context.Background(), 99, article.ID)
	require.ErrorIs(t, err, ierr.ErrNotSubscribed)
}

func TestGetArticleByID_NotFound(t *testing.T) {
	service, _, _, _ := setupArticleService(t)

	_, err := service.GetArticleByID(context.Background(), 1, 123)
	require.ErrorIs(t, err, ierr.ErrArticleNotFound)
}
