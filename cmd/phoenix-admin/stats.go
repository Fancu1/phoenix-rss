package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Fancu1/phoenix-rss/internal/feed-service/models"
)

func newStatsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show statistics",
		Long:  `Display overall statistics for feeds and articles.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStats()
		},
	}

	return cmd
}

func runStats() error {
	ctx := context.Background()

	// Count feeds
	var feedCount int64
	if err := db.WithContext(ctx).Model(&models.Feed{}).Count(&feedCount).Error; err != nil {
		return fmt.Errorf("failed to count feeds: %w", err)
	}

	// Count articles
	var articleCount int64
	if err := db.WithContext(ctx).Model(&models.Article{}).Count(&articleCount).Error; err != nil {
		return fmt.Errorf("failed to count articles: %w", err)
	}

	// Count processed articles
	var processedCount int64
	if err := db.WithContext(ctx).Model(&models.Article{}).Where("processed_at IS NOT NULL").Count(&processedCount).Error; err != nil {
		return fmt.Errorf("failed to count processed articles: %w", err)
	}

	pendingCount := articleCount - processedCount

	// Calculate percentages
	var processedPercent, pendingPercent float64
	if articleCount > 0 {
		processedPercent = float64(processedCount) / float64(articleCount) * 100
		pendingPercent = float64(pendingCount) / float64(articleCount) * 100
	}

	// Print statistics
	fmt.Println()
	fmt.Println("=== Phoenix RSS Statistics ===")
	fmt.Println()
	fmt.Printf("Feeds:     %d\n", feedCount)
	fmt.Printf("Articles:  %d\n", articleCount)
	fmt.Println()
	fmt.Println("AI Processing:")
	fmt.Printf("  ✓ Processed:  %d (%.1f%%)\n", processedCount, processedPercent)
	fmt.Printf("  ⏳ Pending:   %d (%.1f%%)\n", pendingCount, pendingPercent)
	fmt.Println()

	return nil
}

