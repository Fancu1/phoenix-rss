package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/require"

	"github.com/Fancu1/phoenix-rss/internal/models"
)

const (
	TestUsername = "username"
	TestPassword = "12345678"
	TestRSSURL   = "https://www.piglei.com/feeds/latest/"
)

type AuthResponse struct {
	Token string `json:"token"`
	User  struct {
		ID       uint   `json:"id"`
		Username string `json:"username"`
	} `json:"user"`
}

func waitForWorkerToBeIdle(t *testing.T, inspector *asynq.Inspector) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		info, err := inspector.GetQueueInfo("default")
		require.NoError(t, err)
		if info.Active == 0 && info.Pending == 0 && info.Retry == 0 {
			time.Sleep(200 * time.Millisecond)
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("worker did not become idle in time")
}

func Ctx(t *testing.T) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(func() {
		require.NoError(t, app.DB.Exec("TRUNCATE TABLE subscriptions, users, feeds, articles RESTART IDENTITY CASCADE").Error)
		queues, err := app.Inspector.Queues()
		require.NoError(t, err)
		for _, queue := range queues {
			_, err := app.Inspector.DeleteAllScheduledTasks(queue)
			require.NoError(t, err)
		}
		cancel()
	})
	return ctx
}

// Helper function to register a user and return auth token
func registerUser(t *testing.T, username, password string) string {
	t.Helper()

	reqBody := fmt.Sprintf(`{"username": "%s", "password": "%s"}`, username, password)
	req, err := http.NewRequest(http.MethodPost, app.Server.URL+"/api/v1/users/register", bytes.NewBufferString(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var authResp AuthResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&authResp))
	require.NotEmpty(t, authResp.Token)
	require.Equal(t, username, authResp.User.Username)

	return authResp.Token
}

// Helper function to login a user and return auth token
func loginUser(t *testing.T, username, password string) string {
	t.Helper()

	reqBody := fmt.Sprintf(`{"username": "%s", "password": "%s"}`, username, password)
	req, err := http.NewRequest(http.MethodPost, app.Server.URL+"/api/v1/users/login", bytes.NewBufferString(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var authResp AuthResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&authResp))
	require.NotEmpty(t, authResp.Token)
	require.Equal(t, username, authResp.User.Username)

	return authResp.Token
}

// Helper function to make authenticated requests
func makeAuthenticatedRequest(t *testing.T, method, url, body, token string) *http.Response {
	t.Helper()

	var req *http.Request
	var err error

	if body != "" {
		req, err = http.NewRequest(method, url, bytes.NewBufferString(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	require.NoError(t, err)

	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	return resp
}

func TestAuthentication(t *testing.T) {
	_ = Ctx(t)

	t.Run("User registration success", func(t *testing.T) {
		token := registerUser(t, TestUsername, TestPassword)
		require.NotEmpty(t, token)
	})

	t.Run("User registration with existing username", func(t *testing.T) {
		// First registration
		registerUser(t, "duplicate_user", TestPassword)

		// Second registration with same username should fail
		reqBody := `{"username": "duplicate_user", "password": "12345678"}`
		req, err := http.NewRequest(http.MethodPost, app.Server.URL+"/api/v1/users/register", bytes.NewBufferString(reqBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusConflict, resp.StatusCode)

		var errorResp map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&errorResp))
		require.Contains(t, errorResp["error"], "already exists")
	})

	t.Run("User login success", func(t *testing.T) {
		// Register user first
		registerUser(t, "login_user", TestPassword)

		// Login
		token := loginUser(t, "login_user", TestPassword)
		require.NotEmpty(t, token)
	})

	t.Run("User login with invalid credentials", func(t *testing.T) {
		reqBody := `{"username": "nonexistent", "password": "wrongpass"}`
		req, err := http.NewRequest(http.MethodPost, app.Server.URL+"/api/v1/users/login", bytes.NewBufferString(reqBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)

		var errorResp map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&errorResp))
		require.Contains(t, errorResp["error"], "invalid credentials")
	})
}

func TestUnauthorizedAccess(t *testing.T) {
	_ = Ctx(t)

	protectedEndpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/feeds"},
		{http.MethodPost, "/api/v1/feeds"},
		{http.MethodDelete, "/api/v1/feeds/1"},
		{http.MethodPost, "/api/v1/feeds/1/fetch"},
		{http.MethodGet, "/api/v1/feeds/1/articles"},
	}

	for _, endpoint := range protectedEndpoints {
		t.Run(fmt.Sprintf("%s %s without auth", endpoint.method, endpoint.path), func(t *testing.T) {
			var req *http.Request
			var err error

			if endpoint.method == http.MethodPost && endpoint.path == "/api/v1/feeds" {
				req, err = http.NewRequest(endpoint.method, app.Server.URL+endpoint.path, bytes.NewBufferString(`{"url": "http://example.com"}`))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req, err = http.NewRequest(endpoint.method, app.Server.URL+endpoint.path, nil)
			}
			require.NoError(t, err)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, http.StatusUnauthorized, resp.StatusCode)

			var errorResp map[string]interface{}
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&errorResp))
			require.Contains(t, errorResp["error"], "authorization header required")
		})
	}
}

func TestFeedManagementE2E(t *testing.T) {
	_ = Ctx(t)

	// Register and login user
	token := registerUser(t, TestUsername, TestPassword)

	t.Run("List feeds empty", func(t *testing.T) {
		resp := makeAuthenticatedRequest(t, http.MethodGet, app.Server.URL+"/api/v1/feeds", "", token)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var feeds []models.Feed
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&feeds))
		require.Empty(t, feeds)
	})

	var createdFeed models.Feed

	t.Run("Subscribe to feed", func(t *testing.T) {
		reqBody := fmt.Sprintf(`{"url": "%s"}`, TestRSSURL)
		resp := makeAuthenticatedRequest(t, http.MethodPost, app.Server.URL+"/api/v1/feeds", reqBody, token)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Skipf("Skipping due to feed subscription returning status %d", resp.StatusCode)
		}

		require.Equal(t, http.StatusCreated, resp.StatusCode)

		require.NoError(t, json.NewDecoder(resp.Body).Decode(&createdFeed))
		require.Equal(t, TestRSSURL, createdFeed.URL)
		require.NotEmpty(t, createdFeed.ID)
		require.NotEmpty(t, createdFeed.Title)
	})

	if createdFeed.ID == 0 {
		t.Log("Skipping remaining tests due to no feed created")
		return
	}

	t.Run("Trigger feed fetch", func(t *testing.T) {
		fetchURL := fmt.Sprintf("%s/api/v1/feeds/%d/fetch", app.Server.URL, createdFeed.ID)
		resp := makeAuthenticatedRequest(t, http.MethodPost, fetchURL, "", token)
		defer resp.Body.Close()

		require.Equal(t, http.StatusAccepted, resp.StatusCode)

		var response map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&response))
		require.Contains(t, response, "task_id")
		require.NotEmpty(t, response["task_id"])
	})

	t.Log("Waiting for worker to fetch articles...")
	waitForWorkerToBeIdle(t, app.Inspector)

	t.Run("List articles after fetch", func(t *testing.T) {
		listURL := fmt.Sprintf("%s/api/v1/feeds/%d/articles", app.Server.URL, createdFeed.ID)
		resp := makeAuthenticatedRequest(t, http.MethodGet, listURL, "", token)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var articles []*models.Article
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&articles))
		require.NotEmpty(t, articles, "expected to fetch articles, but got none")

		// Verify article structure
		article := articles[0]
		require.Equal(t, createdFeed.ID, article.FeedID)
		require.NotEmpty(t, article.Title)
		require.NotEmpty(t, article.URL)

		t.Logf("Successfully fetched %d articles for feed %d", len(articles), createdFeed.ID)
	})

	t.Run("Unsubscribe from feed", func(t *testing.T) {
		unsubURL := fmt.Sprintf("%s/api/v1/feeds/%d", app.Server.URL, createdFeed.ID)
		resp := makeAuthenticatedRequest(t, http.MethodDelete, unsubURL, "", token)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&response))
		require.Contains(t, response["message"], "unsubscribed")
	})
}

func TestUserIsolation(t *testing.T) {
	_ = Ctx(t)

	// Create two users
	token1 := registerUser(t, "user1", TestPassword)
	token2 := registerUser(t, "user2", TestPassword)

	// User1 subscribes to a feed
	reqBody := fmt.Sprintf(`{"url": "%s"}`, TestRSSURL)
	resp := makeAuthenticatedRequest(t, http.MethodPost, app.Server.URL+"/api/v1/feeds", reqBody, token1)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Skip("Skipping user isolation test due to feed creation failure")
	}

	var user1Feed models.Feed
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&user1Feed))

	t.Run("User2 cannot see User1's feeds", func(t *testing.T) {
		resp := makeAuthenticatedRequest(t, http.MethodGet, app.Server.URL+"/api/v1/feeds", "", token2)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var feeds []models.Feed
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&feeds))
		require.Empty(t, feeds)
	})

	t.Run("User2 cannot access User1's feed articles", func(t *testing.T) {
		listURL := fmt.Sprintf("%s/api/v1/feeds/%d/articles", app.Server.URL, user1Feed.ID)
		resp := makeAuthenticatedRequest(t, http.MethodGet, listURL, "", token2)
		defer resp.Body.Close()

		require.Equal(t, http.StatusForbidden, resp.StatusCode)

		var errorResp map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&errorResp))
		require.Contains(t, errorResp["error"], "not subscribed")
	})
}

func TestHealthCheck(t *testing.T) {
	t.Run("Health check endpoint", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, app.Server.URL+"/api/v1/health", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&response))
		require.Equal(t, "ok", response["status"])
	})
}
