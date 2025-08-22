package handler

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Fancu1/phoenix-rss/internal/events"
	"github.com/Fancu1/phoenix-rss/internal/feed-service/core"
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

// SubscribeToFeed create a subscription
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

	if err := h.producer.PublishFeedFetch(ctx, feed.ID); err != nil {
		log.Warn("failed to publish feed fetch event, but subscription created", "feed_id", feed.ID, "error", err.Error())
	}

	pbFeed := &feedpb.Feed{
		Id:          uint64(feed.ID),
		Title:       feed.Title,
		Url:         feed.URL,
		Description: feed.Description,
		CreatedAt:   feed.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   feed.UpdatedAt.Format(time.RFC3339),
	}

	log.Info("successfully subscribed user to feed", "user_id", req.UserId, "feed_id", feed.ID)
	return &feedpb.SubscribeToFeedResponse{Feed: pbFeed}, nil
}

// ListUserFeeds return all feeds subscribed by a specific user
func (h *FeedServiceHandler) ListUserFeeds(ctx context.Context, req *feedpb.ListUserFeedsRequest) (*feedpb.ListUserFeedsResponse, error) {
	log := logger.FromContext(ctx)
	log.Info("gRPC: ListUserFeeds", "user_id", req.UserId)

	if req.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

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
			CreatedAt:   feed.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   feed.UpdatedAt.Format(time.RFC3339),
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
		pbArticles[i] = &feedpb.Article{
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
	}

	log.Info("successfully listed articles", "user_id", req.UserId, "feed_id", req.FeedId, "count", len(articles))
	return &feedpb.ListArticlesResponse{Articles: pbArticles}, nil
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
