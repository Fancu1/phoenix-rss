package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/Fancu1/phoenix-rss/internal/events"
	"github.com/Fancu1/phoenix-rss/internal/feed-service/repository"
	"github.com/Fancu1/phoenix-rss/pkg/logger"
)

type ArticleUpdateConfig struct {
	UserAgent       string
	MaxAttempts     int
	BackoffInitial  time.Duration
	BackoffMax      time.Duration
	Jitter          bool
	MaxContentBytes int64
	RespectRobots   bool
}

type ArticleUpdateChecker struct {
	repo       *repository.ArticleRepository
	logger     *slog.Logger
	httpClient *http.Client
	robots     *RobotsClient
	cfg        ArticleUpdateConfig
	randSource *rand.Rand
}

func NewArticleUpdateChecker(repo *repository.ArticleRepository, logger *slog.Logger, httpClient *http.Client, robots *RobotsClient, cfg ArticleUpdateConfig) *ArticleUpdateChecker {
	if cfg.UserAgent == "" {
		cfg.UserAgent = "PhoenixRSS/1.0 (+https://github.com/Fancu1/phoenix-rss)"
	}
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 3
	}
	if cfg.BackoffInitial <= 0 {
		cfg.BackoffInitial = 500 * time.Millisecond
	}
	if cfg.BackoffMax < cfg.BackoffInitial {
		cfg.BackoffMax = 10 * time.Second
	}
	if cfg.MaxContentBytes <= 0 {
		cfg.MaxContentBytes = 2 << 20 // 2 MiB
	}

	return &ArticleUpdateChecker{
		repo:       repo,
		logger:     logger,
		httpClient: httpClient,
		robots:     robots,
		cfg:        cfg,
		randSource: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (c *ArticleUpdateChecker) HandleEvent(ctx context.Context, event events.ArticleCheckEvent) error {
	taskCtx := logger.WithValue(ctx, "article_id", event.ArticleID)
	taskCtx = logger.WithValue(taskCtx, "request_id", event.RequestID)
	log := logger.FromContext(taskCtx)

	if strings.TrimSpace(event.URL) == "" {
		return fmt.Errorf("event url cannot be empty")
	}

	if c.cfg.RespectRobots && c.robots != nil {
		allowed, err := c.robots.IsAllowed(taskCtx, event.URL, c.cfg.UserAgent)
		if err != nil {
			log.Warn("robots check failed, proceeding", "error", err)
		} else if !allowed {
			log.Info("robots disallow article fetch", "url", event.URL)
			return c.repo.MarkLastChecked(taskCtx, event.ArticleID, time.Now().UTC())
		}
	}

	headResp, err := c.performRequest(taskCtx, http.MethodHead, event.URL, event)
	if err != nil {
		log.Error("head request failed", "error", err)
		return err
	}
	defer headResp.Body.Close()

	switch headResp.StatusCode {
	case http.StatusNotModified:
		log.Info("article not modified", "status", headResp.StatusCode)
		return c.repo.MarkLastChecked(taskCtx, event.ArticleID, time.Now().UTC())
	case http.StatusOK:
		// continue to GET
	case http.StatusMethodNotAllowed, http.StatusNotImplemented:
		log.Info("head not supported, falling back to GET", "status", headResp.StatusCode)
	default:
		if isRetryableStatus(headResp.StatusCode) {
			return fmt.Errorf("head request returned retryable status %d", headResp.StatusCode)
		}
		log.Warn("head request returned non-retryable status", "status", headResp.StatusCode)
		return c.repo.MarkLastChecked(taskCtx, event.ArticleID, time.Now().UTC())
	}

	getResp, err := c.performRequest(taskCtx, http.MethodGet, event.URL, event)
	if err != nil {
		log.Error("get request failed", "error", err)
		return err
	}
	defer getResp.Body.Close()

	switch getResp.StatusCode {
	case http.StatusOK:
		// proceed
	case http.StatusNotModified:
		log.Info("article unchanged on GET", "status", getResp.StatusCode)
		return c.repo.MarkLastChecked(taskCtx, event.ArticleID, time.Now().UTC())
	default:
		if isRetryableStatus(getResp.StatusCode) {
			return fmt.Errorf("get request returned retryable status %d", getResp.StatusCode)
		}
		log.Warn("get request returned non-retryable status", "status", getResp.StatusCode)
		return c.repo.MarkLastChecked(taskCtx, event.ArticleID, time.Now().UTC())
	}

	body, err := readLimited(getResp.Body, c.cfg.MaxContentBytes)
	if err != nil {
		return fmt.Errorf("failed to read article body: %w", err)
	}

	content, description := c.sanitizeContent(taskCtx, string(body), event.URL)

	newEtag := preferHeader(getResp.Header.Get("ETag"), headResp.Header.Get("ETag"))
	newLastModified := normalizeHTTPDate(preferHeader(getResp.Header.Get("Last-Modified"), headResp.Header.Get("Last-Modified")))

	now := time.Now().UTC()
	updated, updateErr := c.repo.UpdateArticleOnChange(
		taskCtx,
		event.ArticleID,
		content,
		description,
		optionalString(newEtag),
		optionalString(newLastModified),
		now,
		optionalString(trim(event.PrevETag)),
		optionalString(trim(event.PrevLastModified)),
	)
	if updateErr != nil {
		return fmt.Errorf("failed to update article: %w", updateErr)
	}

	if !updated {
		log.Info("article update skipped due to concurrent changes")
		return c.repo.MarkLastChecked(taskCtx, event.ArticleID, now)
	}

	log.Info("article updated", "etag", newEtag, "last_modified", newLastModified)
	return nil
}

func (c *ArticleUpdateChecker) performRequest(ctx context.Context, method, rawURL string, event events.ArticleCheckEvent) (*http.Response, error) {
	headers := make(http.Header)
	headers.Set("User-Agent", c.cfg.UserAgent)
	headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	headers.Set("Accept-Language", "en-US,en;q=0.9")

	if method == http.MethodHead || method == http.MethodGet {
		if etag := trim(event.PrevETag); etag != "" {
			headers.Set("If-None-Match", etag)
		}
		if lm := trim(event.PrevLastModified); lm != "" {
			if httpDate := toHTTPDate(lm); httpDate != "" {
				headers.Set("If-Modified-Since", httpDate)
			}
		}
	}

	backoff := c.cfg.BackoffInitial
	for attempt := 1; attempt <= c.cfg.MaxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, rawURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header = headers.Clone()

		resp, err := c.httpClient.Do(req)
		if err == nil && !isRetryableStatus(resp.StatusCode) {
			return resp, nil
		}
		if err == nil && attempt == c.cfg.MaxAttempts {
			return resp, nil
		}
		if err != nil && attempt == c.cfg.MaxAttempts {
			return nil, err
		}
		if err == nil {
			drain(resp.Body)
			resp.Body.Close()
		}

		wait := backoff
		if c.cfg.Jitter {
			wait = time.Duration(c.randSource.Int63n(int64(backoff))) + backoff/2
		}

		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		backoff = backoff * 2
		if backoff > c.cfg.BackoffMax {
			backoff = c.cfg.BackoffMax
		}
	}

	return nil, errors.New("request attempts exhausted")
}

func (c *ArticleUpdateChecker) sanitizeContent(ctx context.Context, raw, base string) (string, string) {
	log := logger.FromContext(ctx)

	sanitized, err := sanitizeHTML(raw, base)
	if err != nil {
		log.Warn("failed to sanitize html", "error", err)
		sanitized = ensureHTML(raw)
	}

	description := sanitizePlainText(sanitized)
	if description == "" {
		description = sanitizePlainText(raw)
	}

	return sanitized, description
}

func isRetryableStatus(code int) bool {
	if code == http.StatusTooManyRequests || code == http.StatusRequestTimeout {
		return true
	}
	return code >= 500
}

func drain(body io.Reader) {
	_, _ = io.Copy(io.Discard, io.LimitReader(body, 512))
}

func readLimited(r io.Reader, limit int64) (string, error) {
	reader := io.LimitReader(r, limit+1)
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	if int64(len(data)) > limit {
		return "", fmt.Errorf("response exceeds limit of %d bytes", limit)
	}
	return string(data), nil
}

func optionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	v := value
	return &v
}

func trim(value string) string {
	return strings.TrimSpace(value)
}

func preferHeader(primary, fallback string) string {
	if v := trim(primary); v != "" {
		return v
	}
	return trim(fallback)
}

func normalizeHTTPDate(value string) string {
	if value == "" {
		return ""
	}
	if t, err := http.ParseTime(value); err == nil {
		return t.UTC().Format(time.RFC3339)
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t.UTC().Format(time.RFC3339)
	}
	return value
}

func toHTTPDate(value string) string {
	if value == "" {
		return ""
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t.UTC().Format(http.TimeFormat)
	}
	if t, err := time.Parse(http.TimeFormat, value); err == nil {
		return t.UTC().Format(http.TimeFormat)
	}
	return value
}
