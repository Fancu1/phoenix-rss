package core

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type RobotsClient struct {
	httpClient *http.Client
	logger     *slog.Logger
	ttl        time.Duration
	cacheMu    sync.RWMutex
	cache      map[string]robotsCacheEntry
}

type robotsCacheEntry struct {
	fetchedAt time.Time
	rules     robotsRules
}

type robotsRules struct {
	allows    []string
	disallows []string
}

func NewRobotsClient(httpClient *http.Client, ttl time.Duration, logger *slog.Logger) *RobotsClient {
	if ttl <= 0 {
		ttl = 12 * time.Hour
	}
	return &RobotsClient{
		httpClient: httpClient,
		logger:     logger,
		ttl:        ttl,
		cache:      make(map[string]robotsCacheEntry),
	}
}

func (c *RobotsClient) IsAllowed(ctx context.Context, rawURL, userAgent string) (bool, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false, fmt.Errorf("invalid url for robots check: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return true, nil
	}

	hostKey := parsed.Scheme + "://" + parsed.Host

	c.cacheMu.RLock()
	entry, ok := c.cache[hostKey]
	if ok && time.Since(entry.fetchedAt) < c.ttl {
		c.cacheMu.RUnlock()
		return entry.rules.allowsPath(parsed.EscapedPath()), nil
	}
	c.cacheMu.RUnlock()

	rules, err := c.fetchRules(ctx, hostKey, userAgent)
	if err != nil {
		return true, err
	}

	c.cacheMu.Lock()
	c.cache[hostKey] = robotsCacheEntry{fetchedAt: time.Now(), rules: rules}
	c.cacheMu.Unlock()

	return rules.allowsPath(parsed.EscapedPath()), nil
}

func (c *RobotsClient) fetchRules(ctx context.Context, base, userAgent string) (robotsRules, error) {
	robotsURL := base + "/robots.txt"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, robotsURL, nil)
	if err != nil {
		return robotsRules{}, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Debug("robots fetch failed", "url", robotsURL, "error", err)
		return robotsRules{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return robotsRules{}, nil
	}
	if resp.StatusCode != http.StatusOK {
		c.logger.Debug("robots fetch unexpected status", "url", robotsURL, "status", resp.StatusCode)
		return robotsRules{}, fmt.Errorf("robots fetch status %d", resp.StatusCode)
	}

	body, err := readWithLimit(resp.Body, 1<<20)
	if err != nil {
		return robotsRules{}, err
	}

	rules := parseRobots(string(body), strings.ToLower(userAgent))
	return rules, nil
}

func readWithLimit(r io.Reader, limit int64) ([]byte, error) {
	reader := io.LimitReader(r, limit+1)
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("robots.txt exceeds limit of %d bytes", limit)
	}
	return data, nil
}

func parseRobots(content, userAgent string) robotsRules {
	groups := make(map[string]*robotsRules)
	groups["*"] = &robotsRules{}

	scanner := bufio.NewScanner(strings.NewReader(content))
	currentAgents := []string{"*"}
	lastWasUserAgent := false

	for scanner.Scan() {
		line := scanner.Text()
		if hash := strings.Index(line, "#"); hash >= 0 {
			line = line[:hash]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			lastWasUserAgent = false
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			lastWasUserAgent = false
			continue
		}

		field := strings.ToLower(strings.TrimSpace(parts[0]))
		value := strings.TrimSpace(parts[1])

		switch field {
		case "user-agent":
			ua := strings.ToLower(value)
			if !lastWasUserAgent {
				currentAgents = nil
			}
			currentAgents = append(currentAgents, ua)
			if _, ok := groups[ua]; !ok {
				groups[ua] = &robotsRules{}
			}
			lastWasUserAgent = true
		case "disallow":
			if value == "" {
				lastWasUserAgent = false
				continue
			}
			for _, agent := range currentAgents {
				grp := groups[agent]
				if grp == nil {
					grp = &robotsRules{}
					groups[agent] = grp
				}
				grp.disallows = append(grp.disallows, value)
			}
			lastWasUserAgent = false
		case "allow":
			if value == "" {
				lastWasUserAgent = false
				continue
			}
			for _, agent := range currentAgents {
				grp := groups[agent]
				if grp == nil {
					grp = &robotsRules{}
					groups[agent] = grp
				}
				grp.allows = append(grp.allows, value)
			}
			lastWasUserAgent = false
		default:
			lastWasUserAgent = false
		}
	}

	if rules, ok := groups[userAgent]; ok && (len(rules.allows) > 0 || len(rules.disallows) > 0) {
		return *rules
	}
	if rules, ok := groups["*"]; ok {
		return *rules
	}
	return robotsRules{}
}

func (r robotsRules) allowsPath(path string) bool {
	if path == "" {
		path = "/"
	}
	bestAllow := -1
	bestDisallow := -1

	for _, pattern := range r.allows {
		if length := matchRobots(path, pattern); length > bestAllow {
			bestAllow = length
		}
	}
	for _, pattern := range r.disallows {
		if length := matchRobots(path, pattern); length > bestDisallow {
			bestDisallow = length
		}
	}

	if bestDisallow == -1 {
		return true
	}
	if bestAllow == -1 {
		return false
	}
	return bestAllow >= bestDisallow
}

func matchRobots(path, pattern string) int {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return -1
	}
	pattern = strings.TrimSuffix(pattern, "$")

	if !strings.Contains(pattern, "*") {
		if strings.HasPrefix(path, pattern) {
			return len(pattern)
		}
		return -1
	}

	parts := strings.Split(pattern, "*")
	idx := 0
	for _, part := range parts {
		if part == "" {
			continue
		}
		pos := strings.Index(path[idx:], part)
		if pos == -1 {
			return -1
		}
		idx += pos + len(part)
	}
	return len(strings.ReplaceAll(pattern, "*", ""))
}
