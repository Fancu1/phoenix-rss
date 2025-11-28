package core

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mmcdole/gofeed"

	"github.com/Fancu1/phoenix-rss/internal/events"
	"github.com/Fancu1/phoenix-rss/internal/feed-service/models"
	"github.com/Fancu1/phoenix-rss/internal/feed-service/repository"
	"github.com/Fancu1/phoenix-rss/pkg/ierr"
	"github.com/Fancu1/phoenix-rss/pkg/logger"
)

// BatchSubscribeResult represents the result of a single feed subscription attempt
type BatchSubscribeResult struct {
	URL     string
	Success bool
	Error   string
	Feed    *models.Feed
}

type FeedServiceInterface interface {
	AddFeedByURL(ctx context.Context, url string) (*models.Feed, error)
	ListAllFeeds(ctx context.Context) ([]*models.Feed, error)
	SubscribeToFeed(ctx context.Context, userID uint, url string) (*models.Feed, error)
	BatchSubscribeToFeeds(ctx context.Context, userID uint, urls []string) ([]BatchSubscribeResult, error)
	ListUserFeeds(ctx context.Context, userID uint) ([]*models.UserFeed, error)
	UnsubscribeFromFeed(ctx context.Context, userID, feedID uint) error
	IsUserSubscribed(ctx context.Context, userID, feedID uint) (bool, error)
	UpdateFeedCustomTitle(ctx context.Context, userID, feedID uint, customTitle *string) (*models.UserFeed, error)
}

type FeedService struct {
	parser   *gofeed.Parser
	repo     *repository.FeedRepository
	producer events.Producer
	logger   *slog.Logger
}

// NewFeedService creates a FeedService. Producer can be nil (sync mode).
func NewFeedService(repo *repository.FeedRepository, logger *slog.Logger, producer events.Producer) *FeedService {
	return &FeedService{
		parser:   gofeed.NewParser(),
		repo:     repo,
		producer: producer,
		logger:   logger,
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
	var needFetch bool

	if existingFeed != nil {
		log.Info("found existing feed", "feed_id", existingFeed.ID, "url", url)
		feed = existingFeed
	} else {
		log.Info("feed does not exist, creating new feed record", "url", url)
		feed, err = s.createFeed(ctx, url)
		if err != nil {
			log.Error("failed to create feed", "url", url, "error", err.Error())
			return nil, err
		}
		needFetch = true
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

	if needFetch && s.producer != nil {
		if err := s.producer.PublishFeedFetch(ctx, feed.ID); err != nil {
			log.Warn("failed to publish feed fetch event, scheduler will retry", "feed_id", feed.ID, "error", err.Error())
		} else {
			log.Info("published feed fetch event", "feed_id", feed.ID)
		}
	}

	log.Info("successfully created subscription", "user_id", userID, "feed_id", feed.ID, "async", needFetch)
	return feed, nil
}

func (s *FeedService) createFeed(ctx context.Context, url string) (*models.Feed, error) {
	log := logger.FromContext(ctx)

	newFeed := &models.Feed{
		Title:     url, // temporary title until first fetch
		URL:       url,
		Status:    models.FeedStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	log.Info("creating feed record", "url", url)

	createdFeed, err := s.repo.Create(ctx, newFeed)
	if err != nil {
		log.Error("failed to create feed in database", "url", url, "error", err.Error())
		return nil, ierr.NewDatabaseError(fmt.Errorf("failed to create feed for URL '%s': %w", url, err))
	}

	log.Info("successfully created feed", "feed_id", createdFeed.ID, "url", url)
	return createdFeed, nil
}

func (s *FeedService) ListUserFeeds(ctx context.Context, userID uint) ([]*models.UserFeed, error) {
	log := logger.FromContext(ctx)

	log.Info("listing feeds for user", "user_id", userID)

	feeds, err := s.repo.ListUserFeeds(ctx, userID)
	if err != nil {
		log.Error("failed to list user feeds", "user_id", userID, "error", err.Error())
		return nil, ierr.NewDatabaseError(fmt.Errorf("failed to list feeds for user %d: %w", userID, err))
	}

	log.Info("successfully listed user feeds", "user_id", userID, "count", len(feeds))
	return feeds, nil
}

func (s *FeedService) UpdateFeedCustomTitle(ctx context.Context, userID, feedID uint, customTitle *string) (*models.UserFeed, error) {
	log := logger.FromContext(ctx)
	log.Info("updating feed custom title", "user_id", userID, "feed_id", feedID)

	isSubscribed, err := s.repo.IsUserSubscribed(ctx, userID, feedID)
	if err != nil {
		log.Error("failed to check subscription status", "user_id", userID, "feed_id", feedID, "error", err.Error())
		return nil, ierr.NewDatabaseError(fmt.Errorf("failed to check subscription status for user %d and feed %d: %w", userID, feedID, err))
	}

	if !isSubscribed {
		log.Warn("user not subscribed to feed", "user_id", userID, "feed_id", feedID)
		return nil, fmt.Errorf("user %d not subscribed to feed %d: %w", userID, feedID, ierr.ErrNotSubscribed)
	}

	err = s.repo.UpdateSubscriptionCustomTitle(ctx, userID, feedID, customTitle)
	if err != nil {
		log.Error("failed to update custom title", "user_id", userID, "feed_id", feedID, "error", err.Error())
		return nil, ierr.NewDatabaseError(fmt.Errorf("failed to update custom title for user %d and feed %d: %w", userID, feedID, err))
	}

	subscription, err := s.repo.GetSubscription(ctx, userID, feedID)
	if err != nil {
		log.Error("failed to get updated subscription", "user_id", userID, "feed_id", feedID, "error", err.Error())
		return nil, ierr.NewDatabaseError(fmt.Errorf("failed to get subscription for user %d and feed %d: %w", userID, feedID, err))
	}

	log.Info("successfully updated feed custom title", "user_id", userID, "feed_id", feedID)
	return &models.UserFeed{
		Feed:        subscription.Feed,
		CustomTitle: subscription.CustomTitle,
	}, nil
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

// BatchSubscribeToFeeds subscribes a user to multiple feeds in a single batch operation.
func (s *FeedService) BatchSubscribeToFeeds(ctx context.Context, userID uint, urls []string) ([]BatchSubscribeResult, error) {
	log := logger.FromContext(ctx)
	log.Info("batch subscribing user to feeds", "user_id", userID, "url_count", len(urls))

	if len(urls) == 0 {
		return []BatchSubscribeResult{}, nil
	}

	// Deduplicate URLs and build index mapping
	urlSet := make(map[string]bool, len(urls))
	uniqueURLs := make([]string, 0, len(urls))
	results := make([]BatchSubscribeResult, len(urls))
	urlToIndex := make(map[string][]int, len(urls))

	for i, url := range urls {
		if urlSet[url] {
			results[i] = BatchSubscribeResult{URL: url, Success: false, Error: "duplicate URL in import"}
			continue
		}
		urlSet[url] = true
		uniqueURLs = append(uniqueURLs, url)
		urlToIndex[url] = append(urlToIndex[url], i)
	}

	// Query existing feeds
	existingFeeds, err := s.repo.GetByURLs(ctx, uniqueURLs)
	if err != nil {
		log.Error("failed to batch query existing feeds", "error", err.Error())
		return nil, ierr.NewDatabaseError(fmt.Errorf("failed to query existing feeds: %w", err))
	}

	urlToFeed := make(map[string]*models.Feed, len(existingFeeds))
	for _, feed := range existingFeeds {
		urlToFeed[feed.URL] = feed
	}

	// Create new feeds for URLs not in database
	newFeedURLSet := make(map[string]bool)
	newFeedsToCreate := make([]*models.Feed, 0)
	now := time.Now()

	for _, url := range uniqueURLs {
		if _, exists := urlToFeed[url]; !exists {
			newFeedsToCreate = append(newFeedsToCreate, &models.Feed{
				Title:     url,
				URL:       url,
				Status:    models.FeedStatusActive,
				CreatedAt: now,
				UpdatedAt: now,
			})
			newFeedURLSet[url] = true
		}
	}

	if len(newFeedsToCreate) > 0 {
		if err := s.repo.BatchCreateFeeds(ctx, newFeedsToCreate); err != nil {
			log.Error("failed to batch create feeds", "error", err.Error())
			return nil, ierr.NewDatabaseError(fmt.Errorf("failed to create feeds: %w", err))
		}
		for _, feed := range newFeedsToCreate {
			urlToFeed[feed.URL] = feed
		}
	}

	// Check existing subscriptions
	allFeedIDs := make([]uint, 0, len(uniqueURLs))
	for _, url := range uniqueURLs {
		if feed, ok := urlToFeed[url]; ok {
			allFeedIDs = append(allFeedIDs, feed.ID)
		}
	}

	existingSubscriptions, err := s.repo.GetUserSubscriptionsByFeedIDs(ctx, userID, allFeedIDs)
	if err != nil {
		log.Error("failed to query existing subscriptions", "error", err.Error())
		return nil, ierr.NewDatabaseError(fmt.Errorf("failed to query subscriptions: %w", err))
	}

	// Create subscriptions and build results
	newSubscriptions := make([]*models.Subscription, 0)
	feedsNeedingFetch := make([]uint, 0)

	for _, url := range uniqueURLs {
		feed, ok := urlToFeed[url]
		if !ok {
			for _, idx := range urlToIndex[url] {
				results[idx] = BatchSubscribeResult{URL: url, Success: false, Error: "feed not found"}
			}
			continue
		}

		if existingSubscriptions[feed.ID] {
			for _, idx := range urlToIndex[url] {
				results[idx] = BatchSubscribeResult{URL: url, Success: false, Error: "already subscribed", Feed: feed}
			}
			continue
		}

		newSubscriptions = append(newSubscriptions, &models.Subscription{
			UserID: userID,
			FeedID: feed.ID,
		})

		if newFeedURLSet[url] {
			feedsNeedingFetch = append(feedsNeedingFetch, feed.ID)
		}

		for _, idx := range urlToIndex[url] {
			results[idx] = BatchSubscribeResult{URL: url, Success: true, Feed: feed}
		}
	}

	if len(newSubscriptions) > 0 {
		if err := s.repo.BatchCreateSubscriptions(ctx, newSubscriptions); err != nil {
			log.Error("failed to batch create subscriptions", "error", err.Error())
			return nil, ierr.NewDatabaseError(fmt.Errorf("failed to create subscriptions: %w", err))
		}
	}

	// Trigger async feed fetch for new feeds
	if s.producer != nil && len(feedsNeedingFetch) > 0 {
		go func() {
			for _, feedID := range feedsNeedingFetch {
				if err := s.producer.PublishFeedFetch(context.Background(), feedID); err != nil {
					s.logger.Warn("failed to publish feed fetch event", "feed_id", feedID, "error", err.Error())
				}
			}
		}()
	}

	imported, failed := 0, 0
	for _, r := range results {
		if r.Success {
			imported++
		} else if r.Error != "" {
			failed++
		}
	}

	log.Info("batch subscribe completed", "user_id", userID, "imported", imported, "failed", failed)
	return results, nil
}
