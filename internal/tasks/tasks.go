package tasks

import (
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
)

const (
	TaskFeedFetch = "feed:fetch"
)

type FetchFeedPayload struct {
	FeedID uint
}

func NewFeedFetchTask(feedID uint) (*asynq.Task, error) {
	payload, err := json.Marshal(FetchFeedPayload{
		FeedID: feedID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal feed fetch task: %w", err)
	}

	return asynq.NewTask(TaskFeedFetch, payload), nil
}
