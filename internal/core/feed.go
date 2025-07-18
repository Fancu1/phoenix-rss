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

type FeedServiceInterface interface {
	AddFeedByURL(ctx context.Context, url string) (*models.Feed, error)
	ListAllFeeds() ([]*models.Feed, error)
	SubscribeToFeed(ctx context.Context, userID uint, url string) (*models.Feed, error)
	ListUserFeeds(userID uint) ([]*models.Feed, error)
	UnsubscribeFromFeed(userID, feedID uint) error
}

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

// SubscribeToFeed creates a subscription between user and feed
func (s *FeedService) SubscribeToFeed(ctx context.Context, userID uint, url string) (*models.Feed, error) {
	// First, try to find existing feed
	feed, err := s.AddFeedByURL(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to add feed: %w", err)
	}

	// Check if user is already subscribed
	isSubscribed, err := s.repo.IsUserSubscribed(userID, feed.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to check subscription: %w", err)
	}

	if isSubscribed {
		return feed, nil // Already subscribed, return existing feed
	}

	// Create subscription
	subscription := &models.Subscription{
		UserID:    userID,
		FeedID:    feed.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = s.repo.CreateSubscription(subscription)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	s.logger.Info("user subscribed to feed", "user_id", userID, "feed_id", feed.ID)
	return feed, nil
}

// ListUserFeeds returns feeds subscribed by a specific user
func (s *FeedService) ListUserFeeds(userID uint) ([]*models.Feed, error) {
	feeds, err := s.repo.ListByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list user feeds: %w", err)
	}
	return feeds, nil
}

// UnsubscribeFromFeed removes a subscription between user and feed
func (s *FeedService) UnsubscribeFromFeed(userID, feedID uint) error {
	err := s.repo.DeleteSubscription(userID, feedID)
	if err != nil {
		return fmt.Errorf("failed to unsubscribe: %w", err)
	}

	s.logger.Info("user unsubscribed from feed", "user_id", userID, "feed_id", feedID)
	return nil
}
