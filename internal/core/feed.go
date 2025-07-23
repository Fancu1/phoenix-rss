package core

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mmcdole/gofeed"

	"github.com/Fancu1/phoenix-rss/internal/logger"
	"github.com/Fancu1/phoenix-rss/internal/models"
	"github.com/Fancu1/phoenix-rss/internal/repository"
)

type FeedServiceInterface interface {
	AddFeedByURL(ctx context.Context, url string) (*models.Feed, error)
	ListAllFeeds(ctx context.Context) ([]*models.Feed, error)
	SubscribeToFeed(ctx context.Context, userID uint, url string) (*models.Feed, error)
	ListUserFeeds(ctx context.Context, userID uint) ([]*models.Feed, error)
	UnsubscribeFromFeed(ctx context.Context, userID, feedID uint) error
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
	log := logger.FromContext(ctx)

	log.Info("parsing feed from URL", "url", url)

	feed, err := s.parser.ParseURLWithContext(url, ctx)
	if err != nil {
		log.Error("failed to parse feed", "url", url, "error", err.Error())
		return nil, fmt.Errorf("failed to parse feed: %w", err)
	}

	newFeed := &models.Feed{
		Title:       feed.Title,
		URL:         url,
		Description: feed.Description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	log.Info("creating new feed record", "title", feed.Title, "url", url)

	createdFeed, err := s.repo.Create(ctx, newFeed)
	if err != nil {
		log.Error("failed to create feed in database", "url", url, "error", err.Error())
		return nil, fmt.Errorf("failed to create feed: %w", err)
	}

	log.Info("successfully created feed", "feed_id", createdFeed.ID, "title", createdFeed.Title)
	return createdFeed, nil
}

func (s *FeedService) ListAllFeeds(ctx context.Context) ([]*models.Feed, error) {
	log := logger.FromContext(ctx)

	log.Info("listing all feeds")

	feeds, err := s.repo.ListAll(ctx)
	if err != nil {
		log.Error("failed to list all feeds", "error", err.Error())
		return nil, fmt.Errorf("failed to list all feeds: %w", err)
	}

	log.Info("successfully listed all feeds", "count", len(feeds))
	return feeds, nil
}

// SubscribeToFeed creates a subscription between user and feed
// If the feed doesn't exist, it will be created first
func (s *FeedService) SubscribeToFeed(ctx context.Context, userID uint, url string) (*models.Feed, error) {
	log := logger.FromContext(ctx)

	log.Info("attempting to subscribe user to feed", "user_id", userID, "url", url)

	// First, try to find existing feed by URL
	existingFeed, err := s.repo.GetByURL(ctx, url)
	if err != nil && err.Error() != "record not found" {
		log.Error("failed to check for existing feed", "url", url, "error", err.Error())
		return nil, fmt.Errorf("failed to check existing feed: %w", err)
	}

	var feed *models.Feed
	if existingFeed != nil {
		log.Info("found existing feed", "feed_id", existingFeed.ID, "url", url)
		feed = existingFeed
	} else {
		log.Info("feed does not exist, creating new feed", "url", url)
		// Feed doesn't exist, create it
		feed, err = s.AddFeedByURL(ctx, url)
		if err != nil {
			log.Error("failed to create new feed for subscription", "url", url, "error", err.Error())
			return nil, err
		}
	}

	// Check if user is already subscribed
	isSubscribed, err := s.repo.IsUserSubscribed(ctx, userID, feed.ID)
	if err != nil {
		log.Error("failed to check subscription status", "user_id", userID, "feed_id", feed.ID, "error", err.Error())
		return nil, fmt.Errorf("failed to check subscription: %w", err)
	}

	if isSubscribed {
		log.Info("user already subscribed to feed", "user_id", userID, "feed_id", feed.ID)
		return feed, nil
	}

	// Create subscription
	subscription := &models.Subscription{
		UserID: userID,
		FeedID: feed.ID,
	}

	log.Info("creating subscription", "user_id", userID, "feed_id", feed.ID)

	err = s.repo.CreateSubscription(ctx, subscription)
	if err != nil {
		log.Error("failed to create subscription", "user_id", userID, "feed_id", feed.ID, "error", err.Error())
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	log.Info("successfully created subscription", "user_id", userID, "feed_id", feed.ID)
	return feed, nil
}

// ListUserFeeds returns all feeds subscribed by a specific user
func (s *FeedService) ListUserFeeds(ctx context.Context, userID uint) ([]*models.Feed, error) {
	log := logger.FromContext(ctx)

	log.Info("listing feeds for user", "user_id", userID)

	feeds, err := s.repo.ListByUserID(ctx, userID)
	if err != nil {
		log.Error("failed to list user feeds", "user_id", userID, "error", err.Error())
		return nil, fmt.Errorf("failed to list user feeds: %w", err)
	}

	log.Info("successfully listed user feeds", "user_id", userID, "count", len(feeds))
	return feeds, nil
}

// UnsubscribeFromFeed removes a subscription between user and feed
func (s *FeedService) UnsubscribeFromFeed(ctx context.Context, userID, feedID uint) error {
	log := logger.FromContext(ctx)

	log.Info("unsubscribing user from feed", "user_id", userID, "feed_id", feedID)

	// Check if user is subscribed
	isSubscribed, err := s.repo.IsUserSubscribed(ctx, userID, feedID)
	if err != nil {
		log.Error("failed to check subscription status", "user_id", userID, "feed_id", feedID, "error", err.Error())
		return fmt.Errorf("failed to check subscription: %w", err)
	}

	if !isSubscribed {
		log.Warn("user not subscribed to feed", "user_id", userID, "feed_id", feedID)
		return fmt.Errorf("user not subscribed to this feed")
	}

	err = s.repo.DeleteSubscription(ctx, userID, feedID)
	if err != nil {
		log.Error("failed to delete subscription", "user_id", userID, "feed_id", feedID, "error", err.Error())
		return fmt.Errorf("failed to delete subscription: %w", err)
	}

	log.Info("successfully unsubscribed user from feed", "user_id", userID, "feed_id", feedID)
	return nil
}
