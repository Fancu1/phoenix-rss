package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Fancu1/phoenix-rss/internal/feed-service/models"
)

func newFeedsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "feeds",
		Short: "Manage feeds",
		Long:  `List and view feed information.`,
	}

	cmd.AddCommand(newFeedsListCmd())
	cmd.AddCommand(newFeedsShowCmd())

	return cmd
}

func newFeedsListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all feeds",
		Long:  `List all feeds in the database with article counts.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFeedsList()
		},
	}

	return cmd
}

func newFeedsShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show [feed_id]",
		Short: "Show feed details",
		Long:  `Show detailed information about a specific feed.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			feedID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid feed ID: %w", err)
			}
			return runFeedsShow(uint(feedID))
		},
	}

	return cmd
}

func runFeedsList() error {
	ctx := context.Background()

	type FeedWithStats struct {
		models.Feed
		ArticleCount   int64 `gorm:"column:article_count"`
		ProcessedCount int64 `gorm:"column:processed_count"`
	}

	var feeds []FeedWithStats
	err := db.WithContext(ctx).
		Table("feeds").
		Select(`feeds.*, 
			(SELECT COUNT(*) FROM articles WHERE articles.feed_id = feeds.id) as article_count,
			(SELECT COUNT(*) FROM articles WHERE articles.feed_id = feeds.id AND processed_at IS NOT NULL) as processed_count`).
		Order("feeds.id").
		Find(&feeds).Error

	if err != nil {
		return fmt.Errorf("failed to list feeds: %w", err)
	}

	// Print header
	fmt.Println()
	fmt.Printf("%-4s | %-30s | %-30s | %s\n", "ID", "Title", "URL", "Articles (Processed)")
	fmt.Println(strings.Repeat("-", 100))

	// Print feeds
	for _, f := range feeds {
		title := truncateString(f.Title, 30)
		url := truncateString(f.URL, 30)
		fmt.Printf("%-4d | %-30s | %-30s | %d (%d)\n",
			f.ID, title, url, f.ArticleCount, f.ProcessedCount)
	}

	fmt.Println()
	fmt.Printf("Total: %d feeds\n", len(feeds))

	return nil
}

func runFeedsShow(feedID uint) error {
	ctx := context.Background()

	// Get feed
	var feed models.Feed
	if err := db.WithContext(ctx).First(&feed, feedID).Error; err != nil {
		return fmt.Errorf("feed not found: %w", err)
	}

	// Get article stats
	var totalCount int64
	var processedCount int64

	db.WithContext(ctx).Model(&models.Article{}).Where("feed_id = ?", feedID).Count(&totalCount)
	db.WithContext(ctx).Model(&models.Article{}).Where("feed_id = ? AND processed_at IS NOT NULL", feedID).Count(&processedCount)

	// Print feed details
	fmt.Println()
	fmt.Printf("=== Feed #%d ===\n\n", feed.ID)
	fmt.Printf("Title:       %s\n", feed.Title)
	fmt.Printf("URL:         %s\n", feed.URL)
	fmt.Printf("Description: %s\n", truncateString(feed.Description, 60))
	fmt.Printf("Status:      %s\n", feed.Status)
	fmt.Printf("Created:     %s\n", feed.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated:     %s\n", feed.UpdatedAt.Format("2006-01-02 15:04:05"))

	// Print article stats
	fmt.Println()
	fmt.Println("--- Articles ---")
	fmt.Printf("Total:       %d\n", totalCount)
	fmt.Printf("Processed:   %d\n", processedCount)
	fmt.Printf("Pending:     %d\n", totalCount-processedCount)

	if totalCount > 0 {
		percentage := float64(processedCount) / float64(totalCount) * 100
		fmt.Printf("Progress:    %.1f%%\n", percentage)
	}

	fmt.Println()
	return nil
}

