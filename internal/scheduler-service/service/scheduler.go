package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"golang.org/x/sync/semaphore"

	events "github.com/Fancu1/phoenix-rss/internal/events"
	"github.com/Fancu1/phoenix-rss/internal/scheduler-service/interfaces"
	"github.com/Fancu1/phoenix-rss/internal/scheduler-service/models"
	"github.com/Fancu1/phoenix-rss/pkg/logger"
)

type Scheduler struct {
	logger        *slog.Logger
	feedClient    interfaces.FeedServiceClientInterface
	producer      interfaces.ProducerInterface
	articleChecks interfaces.ArticleCheckProducerInterface
	schedule      string
	batchSize     int
	batchDelay    time.Duration
	maxConcurrent int64
	articleCron   string
	articleWindow time.Duration
	articleMinGap time.Duration
	articlePage   int
	cron          *cron.Cron
	running       bool
	mu            sync.RWMutex
}

func NewScheduler(
	logger *slog.Logger,
	feedClient interfaces.FeedServiceClientInterface,
	producer interfaces.ProducerInterface,
	articleProducer interfaces.ArticleCheckProducerInterface,
	schedule string,
	batchSize int,
	batchDelay time.Duration,
	maxConcurrent int,
	articleCron string,
	articleWindow time.Duration,
	articleMinGap time.Duration,
	articlePage int,
) *Scheduler {
	return &Scheduler{
		logger:        logger,
		feedClient:    feedClient,
		producer:      producer,
		articleChecks: articleProducer,
		schedule:      schedule,
		batchSize:     batchSize,
		batchDelay:    batchDelay,
		maxConcurrent: int64(maxConcurrent),
		articleCron:   articleCron,
		articleWindow: articleWindow,
		articleMinGap: articleMinGap,
		articlePage:   articlePage,
		cron:          cron.New(cron.WithSeconds()),
	}
}

// Start the scheduler
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("scheduler already running")
	}

	s.logger.Info("adding cron job", "schedule", s.schedule)

	_, err := s.cron.AddFunc(s.schedule, func() {
		s.triggerFeedFetches(ctx)
	})
	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	if s.articleChecks != nil && s.articleCron != "" {
		s.logger.Info("adding article check cron job", "schedule", s.articleCron)
		if _, err := s.cron.AddFunc(s.articleCron, func() {
			s.triggerArticleChecks(ctx)
		}); err != nil {
			return fmt.Errorf("failed to add article check cron job: %w", err)
		}
	}

	// Start the cron scheduler
	s.cron.Start()
	s.running = true

	s.logger.Info("scheduler started successfully")
	return nil
}

// Stop the scheduler gracefully
func (s *Scheduler) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.logger.Info("stopping scheduler")

	cronCtx := s.cron.Stop()

	// Wait for running jobs to complete or context timeout
	select {
	case <-cronCtx.Done():
		s.logger.Info("all cron jobs completed")
	case <-ctx.Done():
		s.logger.Warn("scheduler stop timeout, some jobs may still be running")
	}

	s.running = false
	s.logger.Info("scheduler stopped")
	return nil
}

// triggerFeedFetches fetch all feeds and publish fetch events with batch processing
func (s *Scheduler) triggerFeedFetches(ctx context.Context) {
	taskCtx := logger.WithValue(ctx, "task", "feed_fetch_scheduler")
	log := logger.FromContext(taskCtx)

	log.Info("starting scheduled feed fetch task with batch processing",
		"batch_size", s.batchSize,
		"batch_delay", s.batchDelay,
		"max_concurrent", s.maxConcurrent,
	)

	// Get all feeds from feed service
	feeds, err := s.feedClient.GetAllFeeds(taskCtx)
	if err != nil {
		log.Error("failed to get feeds from feed service", "error", err.Error())
		return
	}

	if len(feeds) == 0 {
		log.Info("no feeds found to schedule")
		return
	}

	log.Info("processing feeds in batches", "total_feeds", len(feeds))

	// Create batches
	batches := s.createBatches(feeds)
	log.Info("created batches", "batch_count", len(batches), "total_feeds", len(feeds))

	// Process batches with concurrency control and rate limiting
	s.processBatchesConcurrently(taskCtx, batches)

	log.Info("completed scheduled feed fetch task", "total_feeds", len(feeds))
}

func (s *Scheduler) triggerArticleChecks(ctx context.Context) {
	if s.articleChecks == nil {
		return
	}

	taskCtx := logger.WithValue(ctx, "task", "article_check_scheduler")
	log := logger.FromContext(taskCtx)

	now := time.Now().UTC()
	window := models.ArticleCheckWindow{
		PublishedSince:    now.Add(-s.articleWindow),
		LastCheckedBefore: now.Add(-s.articleMinGap),
	}

	pageSize := s.articlePage
	if pageSize <= 0 {
		pageSize = 500
	}

	var (
		pageToken         string
		totalCandidates   int
		successfulPublish int
		failedPublish     int
		pageNumber        int
	)

	log.Info("starting scheduled article update check",
		"window_start", window.PublishedSince,
		"last_checked_before", window.LastCheckedBefore,
		"page_size", pageSize,
	)

	for {
		select {
		case <-ctx.Done():
			log.Info("article check scheduler cancelled")
			return
		default:
		}

		pageNumber++
		pageCtx := logger.WithValue(taskCtx, "page", pageNumber)
		pageLog := logger.FromContext(pageCtx)

		page, err := s.feedClient.ListArticlesToCheck(pageCtx, window, pageSize, pageToken)
		if err != nil {
			pageLog.Error("failed to fetch articles to check", "error", err)
			break
		}

		if len(page.Items) == 0 {
			pageLog.Info("no articles to check in page")
			break
		}

		totalCandidates += len(page.Items)

		for _, item := range page.Items {
			articleCtx := logger.WithValue(pageCtx, "article_id", item.ArticleID)
			evt := events.ArticleCheckEvent{
				ArticleID:        item.ArticleID,
				FeedID:           item.FeedID,
				URL:              item.URL,
				PrevETag:         item.PrevETag,
				PrevLastModified: item.PrevLastModified,
				RequestID:        uuid.NewString(),
				Attempt:          1,
				ScheduledAt:      now,
				Reason:           "scheduled",
			}

			if err := s.articleChecks.PublishArticleCheck(articleCtx, evt); err != nil {
				failedPublish++
				logger.FromContext(articleCtx).Error("failed to publish article check event", "error", err)
				continue
			}

			successfulPublish++
		}

		if page.NextPageToken == "" {
			break
		}
		pageToken = page.NextPageToken
	}

	log.Info("completed scheduled article update check",
		"candidates", totalCandidates,
		"published", successfulPublish,
		"failed", failedPublish,
	)
}

// createBatches split feeds into smaller batches
func (s *Scheduler) createBatches(feeds []*models.Feed) [][]*models.Feed {
	var batches [][]*models.Feed

	for i := 0; i < len(feeds); i += s.batchSize {
		end := i + s.batchSize
		if end > len(feeds) {
			end = len(feeds)
		}
		batches = append(batches, feeds[i:end])
	}

	return batches
}

// processBatchesConcurrently process batches with concurrency control and rate limiting
func (s *Scheduler) processBatchesConcurrently(ctx context.Context, batches [][]*models.Feed) {
	log := logger.FromContext(ctx)

	// Create semaphore for concurrency control
	sem := semaphore.NewWeighted(s.maxConcurrent)

	var wg sync.WaitGroup
	totalSuccessCount := 0
	totalFailedCount := 0
	var countMu sync.Mutex

	for batchIndex, batch := range batches {
		// Acquire semaphore before processing batch
		if err := sem.Acquire(ctx, 1); err != nil {
			log.Error("failed to acquire semaphore", "error", err.Error())
			continue
		}

		wg.Add(1)
		go func(idx int, feedBatch []*models.Feed) {
			defer wg.Done()
			defer sem.Release(1)

			batchCtx := logger.WithValue(ctx, "batch_index", idx)
			batchLog := logger.FromContext(batchCtx)

			batchLog.Info("processing batch",
				"batch_index", idx,
				"batch_size", len(feedBatch),
			)

			successCount, failedCount := s.processBatch(batchCtx, feedBatch)

			// Update global counters
			countMu.Lock()
			totalSuccessCount += successCount
			totalFailedCount += failedCount
			countMu.Unlock()

			batchLog.Info("completed batch",
				"batch_index", idx,
				"successful_dispatches", successCount,
				"failed_dispatches", failedCount,
			)
		}(batchIndex, batch)

		// Add delay between batch starts (except for the last batch)
		if batchIndex < len(batches)-1 {
			select {
			case <-time.After(s.batchDelay):
				// Continue to next batch
			case <-ctx.Done():
				log.Info("context cancelled, stopping batch processing")
				break
			}
		}
	}

	// Wait for all batches to complete
	wg.Wait()

	log.Info("all batches completed",
		"total_successful_dispatches", totalSuccessCount,
		"total_failed_dispatches", totalFailedCount,
	)
}

// processBatch process a single batch of feeds
func (s *Scheduler) processBatch(ctx context.Context, feeds []*models.Feed) (successCount, failedCount int) {
	log := logger.FromContext(ctx)

	for _, feed := range feeds {
		feedCtx := logger.WithValue(ctx, "feed_id", feed.ID)
		feedLog := logger.FromContext(feedCtx)

		err := s.producer.PublishFeedFetch(feedCtx, feed.ID)
		if err != nil {
			feedLog.Error("failed to publish feed fetch event",
				"feed_title", feed.Title,
				"feed_url", feed.URL,
				"error", err.Error(),
			)
			failedCount++
			continue
		}

		feedLog.Debug("published feed fetch event",
			"feed_title", feed.Title,
			"feed_url", feed.URL,
		)
		successCount++
	}

	log.Debug("batch processing completed",
		"successful_dispatches", successCount,
		"failed_dispatches", failedCount,
	)

	return successCount, failedCount
}

// IsRunning check if the scheduler is currently running
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}
