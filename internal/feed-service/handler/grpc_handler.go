package handler

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Fancu1/phoenix-rss/internal/events"
	"github.com/Fancu1/phoenix-rss/internal/feed-service/core"
	"github.com/Fancu1/phoenix-rss/internal/feed-service/models"
	"github.com/Fancu1/phoenix-rss/pkg/ierr"
	"github.com/Fancu1/phoenix-rss/pkg/logger"
	feedpb "github.com/Fancu1/phoenix-rss/protos/gen/go/feed"
)

type FeedServiceHandler struct {
	feedpb.UnimplementedFeedServiceServer
	logger         *slog.Logger
	feedService    core.FeedServiceInterface
	articleService core.ArticleServiceInterface
	producer       events.Producer
}

func NewFeedServiceHandler(
	logger *slog.Logger,
	feedService core.FeedServiceInterface,
	articleService core.ArticleServiceInterface,
	producer events.Producer,
) *FeedServiceHandler {
	return &FeedServiceHandler{
		logger:         logger,
		feedService:    feedService,
		articleService: articleService,
		producer:       producer,
	}
}

func (h *FeedServiceHandler) SubscribeToFeed(ctx context.Context, req *feedpb.SubscribeToFeedRequest) (*feedpb.SubscribeToFeedResponse, error) {
	log := logger.FromContext(ctx)
	log.Info("gRPC: SubscribeToFeed", "user_id", req.UserId, "feed_url", req.FeedUrl)

	if req.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.FeedUrl == "" {
		return nil, status.Error(codes.InvalidArgument, "feed_url is required")
	}

	feed, err := h.feedService.SubscribeToFeed(ctx, uint(req.UserId), req.FeedUrl)
	if err != nil {
		log.Error("failed to subscribe to feed", "user_id", req.UserId, "feed_url", req.FeedUrl, "error", err.Error())
		return nil, h.mapErrorToGRPC(err)
	}

	pbFeed := &feedpb.Feed{
		Id:          uint64(feed.ID),
		Title:       feed.Title,
		Url:         feed.URL,
		Description: feed.Description,
		Status:      string(feed.Status),
		CreatedAt:   feed.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   feed.UpdatedAt.Format(time.RFC3339),
	}

	log.Info("successfully subscribed user to feed", "user_id", req.UserId, "feed_id", feed.ID)
	return &feedpb.SubscribeToFeedResponse{Feed: pbFeed}, nil
}

func (h *FeedServiceHandler) BatchSubscribeToFeeds(ctx context.Context, req *feedpb.BatchSubscribeToFeedsRequest) (*feedpb.BatchSubscribeToFeedsResponse, error) {
	log := logger.FromContext(ctx)
	log.Info("gRPC: BatchSubscribeToFeeds", "user_id", req.UserId, "url_count", len(req.FeedUrls))

	if req.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if len(req.FeedUrls) == 0 {
		return nil, status.Error(codes.InvalidArgument, "feed_urls is required")
	}

	results, err := h.feedService.BatchSubscribeToFeeds(ctx, uint(req.UserId), req.FeedUrls)
	if err != nil {
		log.Error("failed to batch subscribe to feeds", "user_id", req.UserId, "error", err.Error())
		return nil, h.mapErrorToGRPC(err)
	}

	pbResults := make([]*feedpb.BatchSubscribeResult, len(results))
	var imported, failed int32

	for i, r := range results {
		pbResult := &feedpb.BatchSubscribeResult{
			Url:     r.URL,
			Success: r.Success,
			Error:   r.Error,
		}
		if r.Feed != nil {
			pbResult.Feed = &feedpb.Feed{
				Id:          uint64(r.Feed.ID),
				Title:       r.Feed.Title,
				Url:         r.Feed.URL,
				Description: r.Feed.Description,
				Status:      string(r.Feed.Status),
				CreatedAt:   r.Feed.CreatedAt.Format(time.RFC3339),
				UpdatedAt:   r.Feed.UpdatedAt.Format(time.RFC3339),
			}
		}
		pbResults[i] = pbResult

		if r.Success {
			imported++
		} else if r.Error != "" {
			failed++
		}
	}

	log.Info("batch subscribe completed", "user_id", req.UserId, "imported", imported, "failed", failed)
	return &feedpb.BatchSubscribeToFeedsResponse{
		Results:  pbResults,
		Imported: imported,
		Failed:   failed,
	}, nil
}

// ListUserFeeds return active feeds subscribed by a specific user (pending feeds are hidden)
func (h *FeedServiceHandler) ListUserFeeds(ctx context.Context, req *feedpb.ListUserFeedsRequest) (*feedpb.ListUserFeedsResponse, error) {
	log := logger.FromContext(ctx)
	log.Info("gRPC: ListUserFeeds", "user_id", req.UserId)

	if req.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	// FeedService.ListUserFeeds now returns UserFeed with custom_title
	feeds, err := h.feedService.ListUserFeeds(ctx, uint(req.UserId))
	if err != nil {
		log.Error("failed to list user feeds", "user_id", req.UserId, "error", err.Error())
		return nil, h.mapErrorToGRPC(err)
	}

	// Convert to protobuf
	pbFeeds := make([]*feedpb.Feed, len(feeds))
	for i, feed := range feeds {
		pbFeeds[i] = &feedpb.Feed{
			Id:          uint64(feed.ID),
			Title:       feed.Title,
			Url:         feed.URL,
			Description: feed.Description,
			Status:      string(feed.Status),
			CreatedAt:   feed.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   feed.UpdatedAt.Format(time.RFC3339),
		}
		if feed.CustomTitle != nil {
			pbFeeds[i].CustomTitle = feed.CustomTitle
		}
	}

	log.Info("successfully listed user feeds", "user_id", req.UserId, "count", len(feeds))
	return &feedpb.ListUserFeedsResponse{Feeds: pbFeeds}, nil
}

// UnsubscribeFromFeed remove a subscription between user and feed
func (h *FeedServiceHandler) UnsubscribeFromFeed(ctx context.Context, req *feedpb.UnsubscribeFromFeedRequest) (*feedpb.UnsubscribeFromFeedResponse, error) {
	log := logger.FromContext(ctx)
	log.Info("gRPC: UnsubscribeFromFeed", "user_id", req.UserId, "feed_id", req.FeedId)

	if req.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.FeedId == 0 {
		return nil, status.Error(codes.InvalidArgument, "feed_id is required")
	}

	err := h.feedService.UnsubscribeFromFeed(ctx, uint(req.UserId), uint(req.FeedId))
	if err != nil {
		log.Error("failed to unsubscribe from feed", "user_id", req.UserId, "feed_id", req.FeedId, "error", err.Error())
		return nil, h.mapErrorToGRPC(err)
	}

	log.Info("successfully unsubscribed user from feed", "user_id", req.UserId, "feed_id", req.FeedId)
	return &feedpb.UnsubscribeFromFeedResponse{
		Success: true,
		Message: "Successfully unsubscribed from feed",
	}, nil
}

// ListArticles return articles for a specific feed (user must be subscribed)
func (h *FeedServiceHandler) ListArticles(ctx context.Context, req *feedpb.ListArticlesRequest) (*feedpb.ListArticlesResponse, error) {
	log := logger.FromContext(ctx)
	log.Info("gRPC: ListArticles", "user_id", req.UserId, "feed_id", req.FeedId)

	if req.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.FeedId == 0 {
		return nil, status.Error(codes.InvalidArgument, "feed_id is required")
	}

	articles, err := h.articleService.ListArticlesByFeedID(ctx, uint(req.UserId), uint(req.FeedId))
	if err != nil {
		log.Error("failed to list articles", "user_id", req.UserId, "feed_id", req.FeedId, "error", err.Error())
		return nil, h.mapErrorToGRPC(err)
	}

	pbArticles := make([]*feedpb.Article, len(articles))
	for i, article := range articles {
		pbArticles[i] = toProtoArticle(article)
	}

	log.Info("successfully listed articles", "user_id", req.UserId, "feed_id", req.FeedId, "count", len(articles))
	return &feedpb.ListArticlesResponse{Articles: pbArticles}, nil
}

func (h *FeedServiceHandler) GetArticle(ctx context.Context, req *feedpb.GetArticleRequest) (*feedpb.GetArticleResponse, error) {
	log := logger.FromContext(ctx)
	log.Info("gRPC: GetArticle", "user_id", req.UserId, "article_id", req.ArticleId)

	if req.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.ArticleId == 0 {
		return nil, status.Error(codes.InvalidArgument, "article_id is required")
	}

	article, err := h.articleService.GetArticleByID(ctx, uint(req.UserId), uint(req.ArticleId))
	if err != nil {
		log.Error("failed to get article", "user_id", req.UserId, "article_id", req.ArticleId, "error", err.Error())
		return nil, h.mapErrorToGRPC(err)
	}

	log.Info("successfully retrieved article", "user_id", req.UserId, "article_id", req.ArticleId)
	return &feedpb.GetArticleResponse{Article: toProtoArticle(article)}, nil
}

// TriggerFetch publishe a Kafka event for manual feed fetch
func (h *FeedServiceHandler) TriggerFetch(ctx context.Context, req *feedpb.TriggerFetchRequest) (*feedpb.TriggerFetchResponse, error) {
	log := logger.FromContext(ctx)
	log.Info("gRPC: TriggerFetch", "user_id", req.UserId, "feed_id", req.FeedId)

	if req.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.FeedId == 0 {
		return nil, status.Error(codes.InvalidArgument, "feed_id is required")
	}

	// Verify user is subscribed to this feed before allowing fetch
	isSubscribed, err := h.feedService.IsUserSubscribed(ctx, uint(req.UserId), uint(req.FeedId))
	if err != nil {
		log.Error("failed to check subscription", "user_id", req.UserId, "feed_id", req.FeedId, "error", err.Error())
		return nil, h.mapErrorToGRPC(err)
	}

	if !isSubscribed {
		log.Warn("user not subscribed to feed", "user_id", req.UserId, "feed_id", req.FeedId)
		return nil, status.Error(codes.PermissionDenied, "Not subscribed to this feed")
	}

	if err := h.producer.PublishFeedFetch(ctx, uint(req.FeedId)); err != nil {
		log.Error("failed to publish feed fetch event", "feed_id", req.FeedId, "error", err.Error())
		return nil, status.Error(codes.Internal, "Failed to trigger feed fetch")
	}

	log.Info("successfully triggered feed fetch", "user_id", req.UserId, "feed_id", req.FeedId)
	return &feedpb.TriggerFetchResponse{
		Success: true,
		Message: "Feed fetch job accepted",
	}, nil
}

// ListAllFeeds return all feeds in the system
func (h *FeedServiceHandler) ListAllFeeds(ctx context.Context, req *feedpb.ListAllFeedsRequest) (*feedpb.ListAllFeedsResponse, error) {
	log := logger.FromContext(ctx)
	log.Info("gRPC: ListAllFeeds")

	feeds, err := h.feedService.ListAllFeeds(ctx)
	if err != nil {
		log.Error("failed to list all feeds", "error", err.Error())
		return nil, h.mapErrorToGRPC(err)
	}

	pbFeeds := make([]*feedpb.Feed, len(feeds))
	for i, feed := range feeds {
		pbFeeds[i] = &feedpb.Feed{
			Id:          uint64(feed.ID),
			Title:       feed.Title,
			Url:         feed.URL,
			Description: feed.Description,
			Status:      string(feed.Status),
			CreatedAt:   feed.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   feed.UpdatedAt.Format(time.RFC3339),
		}
	}

	log.Info("successfully listed all feeds", "count", len(feeds))
	return &feedpb.ListAllFeedsResponse{Feeds: pbFeeds}, nil
}

// CheckSubscription check if user is subscribed to a feed
func (h *FeedServiceHandler) CheckSubscription(ctx context.Context, req *feedpb.CheckSubscriptionRequest) (*feedpb.CheckSubscriptionResponse, error) {
	log := logger.FromContext(ctx)
	log.Info("gRPC: CheckSubscription", "user_id", req.UserId, "feed_id", req.FeedId)

	if req.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.FeedId == 0 {
		return nil, status.Error(codes.InvalidArgument, "feed_id is required")
	}

	// Check subscription status via feed service
	isSubscribed, err := h.feedService.IsUserSubscribed(ctx, uint(req.UserId), uint(req.FeedId))
	if err != nil {
		log.Error("failed to check subscription", "user_id", req.UserId, "feed_id", req.FeedId, "error", err.Error())
		return nil, h.mapErrorToGRPC(err)
	}

	log.Info("subscription check completed", "user_id", req.UserId, "feed_id", req.FeedId, "is_subscribed", isSubscribed)
	return &feedpb.CheckSubscriptionResponse{
		IsSubscribed: isSubscribed,
	}, nil
}

// UpdateSubscription updates subscription settings (e.g., custom title)
func (h *FeedServiceHandler) UpdateSubscription(ctx context.Context, req *feedpb.UpdateSubscriptionRequest) (*feedpb.UpdateSubscriptionResponse, error) {
	log := logger.FromContext(ctx)
	log.Info("gRPC: UpdateSubscription", "user_id", req.UserId, "feed_id", req.FeedId)

	if req.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.FeedId == 0 {
		return nil, status.Error(codes.InvalidArgument, "feed_id is required")
	}

	userFeed, err := h.feedService.UpdateFeedCustomTitle(ctx, uint(req.UserId), uint(req.FeedId), req.CustomTitle)
	if err != nil {
		log.Error("failed to update subscription", "user_id", req.UserId, "feed_id", req.FeedId, "error", err.Error())
		return nil, h.mapErrorToGRPC(err)
	}

	pbFeed := &feedpb.Feed{
		Id:          uint64(userFeed.ID),
		Title:       userFeed.Title,
		Url:         userFeed.URL,
		Description: userFeed.Description,
		Status:      string(userFeed.Status),
		CreatedAt:   userFeed.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   userFeed.UpdatedAt.Format(time.RFC3339),
	}
	if userFeed.CustomTitle != nil {
		pbFeed.CustomTitle = userFeed.CustomTitle
	}

	log.Info("successfully updated subscription", "user_id", req.UserId, "feed_id", req.FeedId)
	return &feedpb.UpdateSubscriptionResponse{Feed: pbFeed}, nil
}

func (h *FeedServiceHandler) ListArticlesToCheck(ctx context.Context, req *feedpb.ListArticlesToCheckRequest) (*feedpb.ListArticlesToCheckResponse, error) {
	log := logger.FromContext(ctx)
	log.Info("gRPC: ListArticlesToCheck",
		"published_since", req.PublishedSince,
		"last_checked_before", req.LastCheckedBefore,
		"page_size", req.PageSize,
	)

	if req.PublishedSince == "" {
		return nil, status.Error(codes.InvalidArgument, "published_since is required")
	}
	if req.LastCheckedBefore == "" {
		return nil, status.Error(codes.InvalidArgument, "last_checked_before is required")
	}

	publishedSince, err := time.Parse(time.RFC3339, req.PublishedSince)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid published_since timestamp")
	}
	lastCheckedBefore, err := time.Parse(time.RFC3339, req.LastCheckedBefore)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid last_checked_before timestamp")
	}

	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 500
	} else if pageSize > 2000 {
		pageSize = 2000
	}

	items, nextToken, svcErr := h.articleService.ListArticlesToCheck(ctx, publishedSince, lastCheckedBefore, pageSize, req.PageToken)
	if svcErr != nil {
		log.Error("failed to list articles to check", "error", svcErr)
		return nil, h.mapErrorToGRPC(svcErr)
	}

	pbItems := make([]*feedpb.ArticleToCheck, len(items))
	for i, item := range items {
		pbItems[i] = &feedpb.ArticleToCheck{
			ArticleId: uint64(item.ID),
			FeedId:    uint64(item.FeedID),
			Url:       item.URL,
		}
		if item.HTTPETag != nil {
			pbItems[i].PrevEtag = *item.HTTPETag
		}
		if item.HTTPLastModified != nil {
			pbItems[i].PrevLastModified = *item.HTTPLastModified
		}
	}

	log.Info("successfully listed articles to check",
		"count", len(pbItems),
		"has_next", nextToken != "",
	)

	return &feedpb.ListArticlesToCheckResponse{
		Items:         pbItems,
		NextPageToken: nextToken,
	}, nil
}

// mapErrorToGRPC map internal errors to appropriate gRPC status codes
func (h *FeedServiceHandler) mapErrorToGRPC(err error) error {
	if err == ierr.ErrNotSubscribed {
		return status.Error(codes.PermissionDenied, "Not subscribed to this feed")
	}

	switch {
	case ierr.IsValidationError(err):
		return status.Error(codes.InvalidArgument, err.Error())
	case ierr.IsDatabaseError(err):
		return status.Error(codes.Internal, "Database error")
	case ierr.IsNotFound(err):
		return status.Error(codes.NotFound, err.Error())
	case ierr.IsUnauthorized(err):
		return status.Error(codes.Unauthenticated, err.Error())
	case ierr.IsAlreadyExists(err):
		return status.Error(codes.AlreadyExists, err.Error())
	default:
		h.logger.Error("unmapped error", "error", err.Error())
		return status.Error(codes.Internal, "Internal server error")
	}
}

func toProtoArticle(article *models.Article) *feedpb.Article {
	pb := &feedpb.Article{
		Id:          uint64(article.ID),
		FeedId:      uint64(article.FeedID),
		Title:       article.Title,
		Url:         article.URL,
		Description: article.Description,
		Content:     article.Content,
		CreatedAt:   article.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   article.UpdatedAt.Format(time.RFC3339),
		Read:        article.Read,
		Starred:     article.Starred,
		PublishedAt: article.PublishedAt.Format(time.RFC3339),
	}

	if article.Summary != nil {
		pb.Summary = *article.Summary
	}
	if article.ProcessingModel != nil {
		pb.ProcessingModel = *article.ProcessingModel
	}
	if article.ProcessedAt != nil {
		pb.ProcessedAt = article.ProcessedAt.Format(time.RFC3339)
	}
	if article.LastCheckedAt != nil {
		pb.LastCheckedAt = article.LastCheckedAt.Format(time.RFC3339)
	}
	if article.HTTPETag != nil {
		pb.HttpEtag = *article.HTTPETag
	}
	if article.HTTPLastModified != nil {
		pb.HttpLastModified = *article.HTTPLastModified
	}

	return pb
}
