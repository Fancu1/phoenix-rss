package core

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/Fancu1/phoenix-rss/internal/events"
	"github.com/Fancu1/phoenix-rss/internal/feed-service/models"
	"github.com/Fancu1/phoenix-rss/internal/feed-service/repository"
)

func setupCheckerRepo(t *testing.T) (*repository.ArticleRepository, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Article{}))
	return repository.NewArticleRepository(db), db
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestArticleUpdateChecker_UpdatesArticleOnChange(t *testing.T) {
	repo, _ := setupCheckerRepo(t)
	logger := newTestLogger()
	now := time.Now().UTC()

	article := &models.Article{
		FeedID:      1,
		Title:       "Test",
		URL:         "",
		PublishedAt: now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_, err := repo.Create(context.Background(), article)
	require.NoError(t, err)

	headHits := 0
	getHits := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/robots.txt":
			w.WriteHeader(http.StatusNotFound)
		case "/article":
			if r.Method == http.MethodHead {
				headHits++
				w.Header().Set("ETag", "\"v1\"")
				w.Header().Set("Last-Modified", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Format(http.TimeFormat))
				w.WriteHeader(http.StatusOK)
				return
			}
			if r.Method == http.MethodGet {
				getHits++
				w.Header().Set("Content-Type", "text/html")
				_, _ = w.Write([]byte("<p>updated</p>"))
				return
			}
			w.WriteHeader(http.StatusMethodNotAllowed)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	article.URL = srv.URL + "/article"
	_, err = repo.Update(context.Background(), article)
	require.NoError(t, err)

	httpClient := srv.Client()
	httpClient.Timeout = time.Second

	robots := NewRobotsClient(httpClient, time.Hour, logger)
	checker := NewArticleUpdateChecker(repo, logger, httpClient, robots, ArticleUpdateConfig{
		UserAgent:       "testrunner",
		MaxAttempts:     1,
		BackoffInitial:  10 * time.Millisecond,
		BackoffMax:      10 * time.Millisecond,
		Jitter:          false,
		MaxContentBytes: 1024,
		RespectRobots:   false,
	})

	evt := events.ArticleCheckEvent{
		ArticleID:   article.ID,
		FeedID:      article.FeedID,
		URL:         article.URL,
		RequestID:   "test",
		Attempt:     1,
		ScheduledAt: time.Now().UTC(),
		Reason:      "scheduled",
	}

	err = checker.HandleEvent(context.Background(), evt)
	require.NoError(t, err)
	assert.Equal(t, 1, headHits)
	assert.Equal(t, 1, getHits)

	stored, err := repo.GetByID(context.Background(), article.ID)
	require.NoError(t, err)
	assert.Contains(t, stored.Content, "updated")
	require.NotNil(t, stored.HTTPETag)
	assert.Equal(t, "\"v1\"", *stored.HTTPETag)
	require.NotNil(t, stored.LastCheckedAt)
}

func TestArticleUpdateChecker_RespectsRobots(t *testing.T) {
	repo, _ := setupCheckerRepo(t)
	logger := newTestLogger()
	now := time.Now().UTC()

	article := &models.Article{
		FeedID:      1,
		Title:       "Test",
		URL:         "",
		PublishedAt: now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_, err := repo.Create(context.Background(), article)
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/robots.txt":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("User-agent: *\nDisallow: /"))
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	article.URL = srv.URL + "/article"
	_, err = repo.Update(context.Background(), article)
	require.NoError(t, err)

	httpClient := srv.Client()
	httpClient.Timeout = time.Second

	robots := NewRobotsClient(httpClient, time.Hour, logger)
	checker := NewArticleUpdateChecker(repo, logger, httpClient, robots, ArticleUpdateConfig{
		UserAgent:       "testrunner",
		MaxAttempts:     1,
		BackoffInitial:  10 * time.Millisecond,
		BackoffMax:      10 * time.Millisecond,
		Jitter:          false,
		MaxContentBytes: 1024,
		RespectRobots:   true,
	})

	evt := events.ArticleCheckEvent{
		ArticleID:   article.ID,
		FeedID:      article.FeedID,
		URL:         article.URL,
		RequestID:   "test",
		Attempt:     1,
		ScheduledAt: time.Now().UTC(),
		Reason:      "scheduled",
	}

	err = checker.HandleEvent(context.Background(), evt)
	require.NoError(t, err)

	stored, err := repo.GetByID(context.Background(), article.ID)
	require.NoError(t, err)
	require.NotNil(t, stored.LastCheckedAt)
	assert.Equal(t, "", stored.Content)
	assert.Nil(t, stored.HTTPETag)
}

func TestArticleUpdateChecker_FallbackOnHeadNotAllowed(t *testing.T) {
	repo, _ := setupCheckerRepo(t)
	logger := newTestLogger()
	now := time.Now().UTC()

	article := &models.Article{FeedID: 1, Title: "Test", URL: "", PublishedAt: now, CreatedAt: now, UpdatedAt: now}
	_, err := repo.Create(context.Background(), article)
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/robots.txt":
			w.WriteHeader(http.StatusNotFound)
		case "/article":
			if r.Method == http.MethodHead {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			if r.Method == http.MethodGet {
				w.Header().Set("ETag", "new")
				_, _ = w.Write([]byte("<div>body</div>"))
				return
			}
			w.WriteHeader(http.StatusMethodNotAllowed)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	article.URL = srv.URL + "/article"
	_, err = repo.Update(context.Background(), article)
	require.NoError(t, err)

	httpClient := srv.Client()
	httpClient.Timeout = time.Second

	robots := NewRobotsClient(httpClient, time.Hour, logger)
	checker := NewArticleUpdateChecker(repo, logger, httpClient, robots, ArticleUpdateConfig{
		UserAgent:       "testrunner",
		MaxAttempts:     1,
		BackoffInitial:  10 * time.Millisecond,
		BackoffMax:      10 * time.Millisecond,
		Jitter:          false,
		MaxContentBytes: 1024,
		RespectRobots:   false,
	})

	evt := events.ArticleCheckEvent{
		ArticleID:   article.ID,
		FeedID:      article.FeedID,
		URL:         article.URL,
		RequestID:   "test",
		Attempt:     1,
		ScheduledAt: time.Now().UTC(),
		Reason:      "scheduled",
	}

	err = checker.HandleEvent(context.Background(), evt)
	require.NoError(t, err)

	stored, err := repo.GetByID(context.Background(), article.ID)
	require.NoError(t, err)
	require.NotNil(t, stored.HTTPETag)
	assert.Equal(t, "new", *stored.HTTPETag)
}
