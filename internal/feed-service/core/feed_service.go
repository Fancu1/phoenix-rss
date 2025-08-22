package core

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mmcdole/gofeed"

	"github.com/Fancu1/phoenix-rss/internal/feed-service/models"
	"github.com/Fancu1/phoenix-rss/internal/feed-service/repository"
	"github.com/Fancu1/phoenix-rss/pkg/ierr"
	"github.com/Fancu1/phoenix-rss/pkg/logger"
)

type FeedServiceInterface interface {
	AddFeedByURL(ctx context.Context, url string) (*models.Feed, error)
	ListAllFeeds(ctx context.Context) ([]*models.Feed, error)
	SubscribeToFeed(ctx context.Context, userID uint, url string) (*models.Feed, error)
	ListUserFeeds(ctx context.Context, userID uint) ([]*models.Feed, error)
	UnsubscribeFromFeed(ctx context.Context, userID, feedID uint) error
	IsUserSubscribed(ctx context.Context, userID, feedID uint) (bool, error)
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
		return nil, fmt.Errorf("failed to parse feed from URL '%s': %w", url, ierr.ErrFeedFetchFailed.WithCause(err))
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
		return nil, ierr.NewDatabaseError(fmt.Errorf("failed to create feed '%s' (%s): %w", feed.Title, url, err))
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
		return nil, ierr.NewDatabaseError(fmt.Errorf("failed to list all feeds: %w", err))
	}

	log.Info("successfully listed all feeds", "count", len(feeds))
	return feeds, nil
}

func (s *FeedService) SubscribeToFeed(ctx context.Context, userID uint, url string) (*models.Feed, error) {
	log := logger.FromContext(ctx)

	log.Info("attempting to subscribe user to feed", "user_id", userID, "url", url)

	existingFeed, err := s.repo.GetByURL(ctx, url)
	if err != nil && err.Error() != "record not found" {
		log.Error("failed to check for existing feed", "url", url, "error", err.Error())
		return nil, ierr.NewDatabaseError(fmt.Errorf("failed to check existing feed for URL '%s': %w", url, err))
	}

	var feed *models.Feed
	if existingFeed != nil {
		log.Info("found existing feed", "feed_id", existingFeed.ID, "url", url)
		feed = existingFeed
	} else {
		log.Info("feed does not exist, creating new feed", "url", url)
		feed, err = s.AddFeedByURL(ctx, url)
		if err != nil {
			log.Error("failed to create new feed for subscription", "url", url, "error", err.Error())
			return nil, err
		}
	}

	isSubscribed, err := s.repo.IsUserSubscribed(ctx, userID, feed.ID)
	if err != nil {
		log.Error("failed to check subscription status", "user_id", userID, "feed_id", feed.ID, "error", err.Error())
		return nil, ierr.NewDatabaseError(fmt.Errorf("failed to check subscription status for user %d and feed %d: %w", userID, feed.ID, err))
	}

	if isSubscribed {
		log.Info("user already subscribed to feed", "user_id", userID, "feed_id", feed.ID)
		return nil, fmt.Errorf("user %d already subscribed to feed %d (%s): %w", userID, feed.ID, feed.Title, ierr.ErrAlreadySubscribed)
	}

	subscription := &models.Subscription{
		UserID: userID,
		FeedID: feed.ID,
	}

	log.Info("creating subscription", "user_id", userID, "feed_id", feed.ID)

	err = s.repo.CreateSubscription(ctx, subscription)
	if err != nil {
		log.Error("failed to create subscription", "user_id", userID, "feed_id", feed.ID, "error", err.Error())
		return nil, ierr.NewDatabaseError(fmt.Errorf("failed to create subscription for user %d to feed %d (%s): %w", userID, feed.ID, feed.Title, err))
	}

	log.Info("successfully created subscription", "user_id", userID, "feed_id", feed.ID)
	return feed, nil
}

func (s *FeedService) ListUserFeeds(ctx context.Context, userID uint) ([]*models.Feed, error) {
	log := logger.FromContext(ctx)

	log.Info("listing feeds for user", "user_id", userID)

	feeds, err := s.repo.ListByUserID(ctx, userID)
	if err != nil {
		log.Error("failed to list user feeds", "user_id", userID, "error", err.Error())
		return nil, ierr.NewDatabaseError(fmt.Errorf("failed to list feeds for user %d: %w", userID, err))
	}

	log.Info("successfully listed user feeds", "user_id", userID, "count", len(feeds))
	return feeds, nil
}

func (s *FeedService) UnsubscribeFromFeed(ctx context.Context, userID, feedID uint) error {
	log := logger.FromContext(ctx)

	log.Info("unsubscribing user from feed", "user_id", userID, "feed_id", feedID)

	isSubscribed, err := s.repo.IsUserSubscribed(ctx, userID, feedID)
	if err != nil {
		log.Error("failed to check subscription status", "user_id", userID, "feed_id", feedID, "error", err.Error())
		return ierr.NewDatabaseError(fmt.Errorf("failed to check subscription status for user %d and feed %d: %w", userID, feedID, err))
	}

	if !isSubscribed {
		log.Warn("user not subscribed to feed", "user_id", userID, "feed_id", feedID)
		return fmt.Errorf("user %d not subscribed to feed %d: %w", userID, feedID, ierr.ErrNotSubscribed)
	}

	err = s.repo.DeleteSubscription(ctx, userID, feedID)
	if err != nil {
		log.Error("failed to delete subscription", "user_id", userID, "feed_id", feedID, "error", err.Error())
		return ierr.NewDatabaseError(fmt.Errorf("failed to delete subscription for user %d from feed %d: %w", userID, feedID, err))
	}

	log.Info("successfully unsubscribed user from feed", "user_id", userID, "feed_id", feedID)
	return nil
}

// IsUserSubscribed check if a user is subscribed to a feed
func (s *FeedService) IsUserSubscribed(ctx context.Context, userID, feedID uint) (bool, error) {
	log := logger.FromContext(ctx)

	log.Debug("checking user subscription", "user_id", userID, "feed_id", feedID)

	isSubscribed, err := s.repo.IsUserSubscribed(ctx, userID, feedID)
	if err != nil {
		log.Error("failed to check subscription status", "user_id", userID, "feed_id", feedID, "error", err.Error())
		return false, ierr.NewDatabaseError(fmt.Errorf("failed to check subscription status for user %d and feed %d: %w", userID, feedID, err))
	}

	log.Debug("subscription check completed", "user_id", userID, "feed_id", feedID, "is_subscribed", isSubscribed)
	return isSubscribed, nil
}
