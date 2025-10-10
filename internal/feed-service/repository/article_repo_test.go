package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/Fancu1/phoenix-rss/internal/feed-service/models"
)

func setupArticleRepo(t *testing.T) *ArticleRepository {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Article{}))
	return NewArticleRepository(db)
}

func TestArticleRepository_ListArticlesToCheck(t *testing.T) {
	repo := setupArticleRepo(t)
	ctx := context.Background()

	now := time.Now().UTC()

	articles := []*models.Article{
		{FeedID: 1, Title: "A1", URL: "https://example.com/1", PublishedAt: now.Add(-1 * time.Hour), CreatedAt: now, UpdatedAt: now},
		{FeedID: 1, Title: "A2", URL: "https://example.com/2", PublishedAt: now.Add(-2 * time.Hour), CreatedAt: now, UpdatedAt: now, LastCheckedAt: ptrTime(now.Add(-6 * time.Hour))},
		{FeedID: 2, Title: "A3", URL: "https://example.com/3", PublishedAt: now.Add(-3 * time.Hour), CreatedAt: now, UpdatedAt: now, LastCheckedAt: ptrTime(now.Add(-30 * time.Minute))},
	}

	require.NoError(t, repo.CreateBatch(ctx, articles))

	publishedSince := now.Add(-4 * time.Hour)
	lastCheckedBefore := now.Add(-2 * time.Hour)

	records, cursor, err := repo.ListArticlesToCheck(ctx, publishedSince, lastCheckedBefore, 2, nil)
	require.NoError(t, err)
	require.Len(t, records, 2)
	require.NotNil(t, cursor)

	assert.Equal(t, uint(1), records[0].FeedID)
	assert.Equal(t, uint(1), records[0].ID)
	assert.Equal(t, uint(1), records[1].FeedID)

	more, nextCursor, err := repo.ListArticlesToCheck(ctx, publishedSince, lastCheckedBefore, 2, cursor)
	require.NoError(t, err)
	assert.Len(t, more, 0)
	assert.Nil(t, nextCursor)
}

func TestArticleRepository_MarkLastChecked(t *testing.T) {
	repo := setupArticleRepo(t)
	ctx := context.Background()

	article := &models.Article{FeedID: 1, Title: "A1", URL: "https://example.com/1", PublishedAt: time.Now(), CreatedAt: time.Now(), UpdatedAt: time.Now()}
	_, err := repo.Create(ctx, article)
	require.NoError(t, err)

	checkedAt := time.Now().UTC()
	require.NoError(t, repo.MarkLastChecked(ctx, article.ID, checkedAt))

	stored, err := repo.GetByID(ctx, article.ID)
	require.NoError(t, err)
	require.NotNil(t, stored.LastCheckedAt)
	assert.WithinDuration(t, checkedAt, *stored.LastCheckedAt, time.Second)
}

func TestArticleRepository_UpdateArticleOnChange(t *testing.T) {
	repo := setupArticleRepo(t)
	ctx := context.Background()

	now := time.Now().UTC()
	article := &models.Article{FeedID: 1, Title: "A1", URL: "https://example.com/1", PublishedAt: now, CreatedAt: now, UpdatedAt: now}
	_, err := repo.Create(ctx, article)
	require.NoError(t, err)

	checkedAt := now.Add(time.Minute)
	updated, err := repo.UpdateArticleOnChange(ctx, article.ID, "content", "desc", optional("etag"), optional("2024-01-01T00:00:00Z"), checkedAt, nil, nil)
	require.NoError(t, err)
	assert.True(t, updated)

	stored, err := repo.GetByID(ctx, article.ID)
	require.NoError(t, err)
	assert.Equal(t, "content", stored.Content)
	assert.Equal(t, "desc", stored.Description)
	require.NotNil(t, stored.HTTPETag)
	assert.Equal(t, "etag", *stored.HTTPETag)

	updated, err = repo.UpdateArticleOnChange(ctx, article.ID, "new", "desc", optional("etag2"), nil, checkedAt, optional("missing"), nil)
	require.NoError(t, err)
	assert.False(t, updated)
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

func optional(s string) *string {
	return &s
}
