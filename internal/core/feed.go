package core

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mmcdole/gofeed"

	"github.com/Fancu1/phoenix-rss/internal/models"
	"github.com/Fancu1/phoenix-rss/internal/repository"
)

type FeedService struct {
	parser *gofeed.Parser
	repo   *repository.FeedRepository
	logger *slog.Logger
}

func NewFeedService(repo *repository.FeedRepository, logger *slog.Logger) *FeedService {
	return &FeedService{
		parser: gofeed.NewParser(),
		repo:   repo,
		logger: logger,
	}
}

func (s *FeedService) AddFeedByURL(ctx context.Context, url string) (*models.Feed, error) {
	feed, err := s.parser.ParseURLWithContext(url, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse feed: %w", err)
	}

	newFeed := &models.Feed{
		Title:       feed.Title,
		URL:         url,
		Description: feed.Description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	createdFeed, err := s.repo.Create(newFeed)
	if err != nil {
		return nil, fmt.Errorf("failed to create feed: %w", err)
	}

	return createdFeed, nil
}

func (s *FeedService) ListAllFeeds() ([]*models.Feed, error) {
	feeds, err := s.repo.ListAll()
	if err != nil {
		return nil, fmt.Errorf("failed to list all feeds: %w", err)
	}
	return feeds, nil
}
