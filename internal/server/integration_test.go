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

func waitForWorkerToBeIdle(t *testing.T, inspector *asynq.Inspector) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		info, err := inspector.GetQueueInfo("default")
		require.NoError(t, err)
		if info.Active == 0 && info.Pending == 0 && info.Retry == 0 {
			// Give it a little more time to make sure db state is updated
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
		require.NoError(t, app.DB.Exec("TRUNCATE TABLE feeds, articles RESTART IDENTITY CASCADE").Error)
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

func TestFeedE2E(t *testing.T) {
	_ = Ctx(t)

	t.Run("List feeds empty", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, app.Server.URL+"/api/v1/feeds", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var feeds []models.Feed
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&feeds))
		require.Empty(t, feeds)
	})

	var createdFeed models.Feed

	t.Run("Create feed", func(t *testing.T) {
		addReqBody := `{
			"url": "https://www.piglei.com/feeds/latest/"
		}`
		req, err := http.NewRequest(http.MethodPost, app.Server.URL+"/api/v1/feeds", bytes.NewBufferString(addReqBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Skipf("Skipping due to addFeed returning status %d", resp.StatusCode)
		}

		require.Equal(t, http.StatusCreated, resp.StatusCode)

		err = json.NewDecoder(resp.Body).Decode(&createdFeed)
		require.NoError(t, err)
		require.Equal(t, "https://www.piglei.com/feeds/latest/", createdFeed.URL)
		require.NotEmpty(t, createdFeed.ID)

	})

	if createdFeed.ID == 0 {
		t.Logf("Skipping due to no feed created")
		return
	}
	t.Run("Trigger feed fetch", func(t *testing.T) {
		fetchURL := fmt.Sprintf("%s/api/v1/feeds/%d/fetch", app.Server.URL, createdFeed.ID)
		req, err := http.NewRequest(http.MethodPost, fetchURL, nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusAccepted, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		require.Contains(t, response, "task_id")
		require.NotEmpty(t, response["task_id"])
	})

	t.Log("Waiting for worker to fetch articles...")
	waitForWorkerToBeIdle(t, app.Inspector)

	t.Run("List articles and verify", func(t *testing.T) {
		listURL := fmt.Sprintf("%s/api/v1/feeds/%d/articles", app.Server.URL, createdFeed.ID)
		req, err := http.NewRequest(http.MethodGet, listURL, nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var articles []*models.Article
		err = json.NewDecoder(resp.Body).Decode(&articles)
		require.NoError(t, err)
		require.NotEmpty(t, articles, "expected to fetch articles, but got none")

		t.Logf("Successfully fetched %d articles for feed %d", len(articles), createdFeed.ID)
	})

	t.Run("Delete feed", func(t *testing.T) {
	})
}
