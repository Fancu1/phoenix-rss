package core

import (
	"context"
	"fmt"
	"time"

	"github.com/mmcdole/gofeed"

	"github.com/Fancu1/phoenix-rss/internal/models"
	"github.com/Fancu1/phoenix-rss/internal/repository"
)

type FeedService struct {
	parser *gofeed.Parser
	repo   *repository.FeedRepository
}

func NewFeedService(repo *repository.FeedRepository) *FeedService {
	return &FeedService{
		parser: gofeed.NewParser(),
		repo:   repo,
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
