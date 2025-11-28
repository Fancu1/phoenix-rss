package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Fancu1/phoenix-rss/internal/config"
	"github.com/Fancu1/phoenix-rss/internal/events"
	"github.com/Fancu1/phoenix-rss/internal/feed-service/models"
	"github.com/Fancu1/phoenix-rss/pkg/logger"
	article_eventspb "github.com/Fancu1/phoenix-rss/proto/gen/article_events"
)

func newAICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ai",
		Short: "AI processing management",
		Long:  `Manage AI processing for articles.`,
	}

	cmd.AddCommand(newAIProcessCmd())

	return cmd
}

func newAIProcessCmd() *cobra.Command {
	var articleID uint
	var feedID uint

	cmd := &cobra.Command{
		Use:   "process",
		Short: "Trigger AI processing for articles",
		Long:  `Send articles to AI processing queue. Requires either --article-id or --feed-id.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if articleID == 0 && feedID == 0 {
				return fmt.Errorf("either --article-id or --feed-id is required")
			}
			if articleID != 0 && feedID != 0 {
				return fmt.Errorf("cannot specify both --article-id and --feed-id")
			}

			if articleID != 0 {
				return runAIProcessArticle(articleID)
			}
			return runAIProcessFeed(feedID)
		},
	}

	cmd.Flags().UintVarP(&articleID, "article-id", "a", 0, "Process a specific article")
	cmd.Flags().UintVarP(&feedID, "feed-id", "f", 0, "Process all articles in a feed")

	return cmd
}

func runAIProcessArticle(articleID uint) error {
	ctx := context.Background()

	// Get article
	var article models.Article
	if err := db.WithContext(ctx).First(&article, articleID).Error; err != nil {
		return fmt.Errorf("article not found: %w", err)
	}

	// Get feed title
	var feed models.Feed
	db.WithContext(ctx).First(&feed, article.FeedID)

	// Show confirmation
	fmt.Println()
	fmt.Println("=== AI Process Request ===")
	fmt.Println()
	fmt.Printf("Article:      #%d %s\n", article.ID, truncateString(article.Title, 50))
	fmt.Printf("Feed:         %s (ID: %d)\n", feed.Title, feed.ID)
	if article.ProcessedAt != nil {
		fmt.Printf("Status:       Already processed at %s\n", article.ProcessedAt.Format("2006-01-02 15:04:05"))
	} else {
		fmt.Printf("Status:       Pending\n")
	}
	fmt.Println()
	fmt.Print("Type 'yes' to continue: ")

	if !confirmAction() {
		fmt.Println("Cancelled.")
		return nil
	}

	// Send to AI processing queue
	if err := sendToAIQueue(ctx, []models.Article{article}); err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("Done! Article #%d sent to AI processing queue.\n", article.ID)
	return nil
}

func runAIProcessFeed(feedID uint) error {
	ctx := context.Background()

	// Get feed
	var feed models.Feed
	if err := db.WithContext(ctx).First(&feed, feedID).Error; err != nil {
		return fmt.Errorf("feed not found: %w", err)
	}

	// Get all articles for this feed
	var articles []models.Article
	if err := db.WithContext(ctx).Where("feed_id = ?", feedID).Order("published_at DESC").Find(&articles).Error; err != nil {
		return fmt.Errorf("failed to get articles: %w", err)
	}

	if len(articles) == 0 {
		fmt.Println("No articles found for this feed.")
		return nil
	}

	// Count already processed
	var alreadyDone int
	for _, a := range articles {
		if a.ProcessedAt != nil {
			alreadyDone++
		}
	}

	// Show confirmation
	fmt.Println()
	fmt.Println("=== AI Process Request ===")
	fmt.Println()
	fmt.Printf("Feed:         %s (ID: %d)\n", feed.Title, feed.ID)
	fmt.Printf("Total:        %d articles\n", len(articles))
	fmt.Printf("Already Done: %d\n", alreadyDone)
	fmt.Printf("To Process:   %d\n", len(articles))
	fmt.Println()
	fmt.Println("Articles to process:")

	// Show first 10 articles
	displayCount := len(articles)
	if displayCount > 10 {
		displayCount = 10
	}
	for i := 0; i < displayCount; i++ {
		a := articles[i]
		status := ""
		if a.ProcessedAt != nil {
			status = " (already done)"
		}
		fmt.Printf("  #%-4d %s%s\n", a.ID, truncateString(a.Title, 60), status)
	}
	if len(articles) > 10 {
		fmt.Printf("  ... and %d more\n", len(articles)-10)
	}

	fmt.Println()
	fmt.Print("Type 'yes' to continue: ")

	if !confirmAction() {
		fmt.Println("Cancelled.")
		return nil
	}

	// Send to AI processing queue
	if err := sendToAIQueue(ctx, articles); err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("Done! %d articles sent to AI processing queue.\n", len(articles))
	return nil
}

func sendToAIQueue(ctx context.Context, articles []models.Article) error {
	// Load config for Kafka
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create producer
	log := logger.New(0) // quiet logger
	producer := events.NewKafkaArticleEventProducer(log, cfg.Kafka.Brokers, cfg.Kafka.AIProcessing.ArticlesNewTopic)
	defer producer.Close()

	fmt.Println()
	fmt.Printf("Processing %d articles...\n", len(articles))

	for _, article := range articles {
		event := &article_eventspb.ArticlePersistedEvent{
			ArticleId:   uint64(article.ID),
			FeedId:      uint64(article.FeedID),
			Title:       article.Title,
			Content:     article.Content,
			Url:         article.URL,
			Description: article.Description,
			PublishedAt: article.PublishedAt.Unix(),
		}

		if err := producer.PublishArticlePersisted(ctx, event); err != nil {
			fmt.Printf("  ✗ #%d failed: %v\n", article.ID, err)
			continue
		}
		fmt.Printf("  ✓ #%d sent to queue\n", article.ID)
	}

	return nil
}

func confirmAction() bool {
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "yes"
}

