package worker

import (
	"context"
	"fmt"

	"github.com/Fancu1/phoenix-rss/internal/core"
)

type Job struct {
	FeedID uint
}

type Dispatcher struct {
	jobQueue    chan Job
	workerCount int
	articleSvc  *core.ArticleService
}

func NewDispatcher(queueSize int, workerCount int, articleSvc *core.ArticleService) *Dispatcher {
	jobQueue := make(chan Job, queueSize)

	return &Dispatcher{
		jobQueue:    jobQueue,
		workerCount: workerCount,
		articleSvc:  articleSvc,
	}
}

func (d *Dispatcher) Start() {
	for i := 0; i < d.workerCount; i++ {
		go d.worker(i)
	}
}

func (d *Dispatcher) worker(id int) {
	for job := range d.jobQueue {
		_, err := d.articleSvc.FetchAndSaveArticles(context.Background(), job.FeedID)
		if err != nil {
			fmt.Printf("worker %d: error fetching and saving articles: %v\n", id, err)
		}
		fmt.Printf("worker %d: fetched and saved articles for feed %d\n", id, job.FeedID)
	}
}

func (d *Dispatcher) AddJob(job Job) {
	d.jobQueue <- job
}
