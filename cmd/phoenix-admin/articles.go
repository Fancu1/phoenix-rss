package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Fancu1/phoenix-rss/internal/feed-service/models"
)

func newArticlesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "articles",
		Short: "Manage articles",
		Long:  `List and view articles in the database.`,
	}

	cmd.AddCommand(newArticlesListCmd())
	cmd.AddCommand(newArticlesShowCmd())

	return cmd
}

func newArticlesListCmd() *cobra.Command {
	var limit int
	var feedID uint

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List articles",
		Long:  `List articles with optional filtering by feed ID.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runArticlesList(limit, feedID)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Number of articles to display")
	cmd.Flags().UintVarP(&feedID, "feed-id", "f", 0, "Filter by feed ID")

	return cmd
}

func newArticlesShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show [article_id]",
		Short: "Show article details",
		Long:  `Show detailed information about a specific article, including AI summary.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			articleID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid article ID: %w", err)
			}
			return runArticlesShow(uint(articleID))
		},
	}

	return cmd
}

func runArticlesList(limit int, feedID uint) error {
	ctx := context.Background()

	type ArticleWithFeed struct {
		models.Article
		FeedTitle string `gorm:"column:feed_title"`
	}

	var articles []ArticleWithFeed
	var totalCount int64

	// Build query
	query := db.WithContext(ctx).
		Table("articles").
		Select("articles.*, feeds.title as feed_title").
		Joins("LEFT JOIN feeds ON articles.feed_id = feeds.id")

	countQuery := db.WithContext(ctx).Model(&models.Article{})

	if feedID > 0 {
		query = query.Where("articles.feed_id = ?", feedID)
		countQuery = countQuery.Where("feed_id = ?", feedID)
	}

	// Get total count
	if err := countQuery.Count(&totalCount).Error; err != nil {
		return fmt.Errorf("failed to count articles: %w", err)
	}

	// Get articles
	if err := query.Order("articles.published_at DESC").Limit(limit).Find(&articles).Error; err != nil {
		return fmt.Errorf("failed to list articles: %w", err)
	}

	// Print header
	fmt.Println()
	fmt.Printf("%-6s | %-15s | %-40s | %-11s | %s\n", "ID", "Feed", "Title", "AI Status", "Published")
	fmt.Println(strings.Repeat("-", 100))

	// Print articles
	for _, a := range articles {
		feedTitle := truncateString(a.FeedTitle, 15)
		title := truncateString(a.Title, 40)
		aiStatus := "⏳ Pending"
		if a.ProcessedAt != nil {
			aiStatus = "✓ Processed"
		}
		published := formatDate(a.PublishedAt)

		fmt.Printf("%-6d | %-15s | %-40s | %-11s | %s\n",
			a.ID, feedTitle, title, aiStatus, published)
	}

	fmt.Println()
	fmt.Printf("Showing %d of %d articles\n", len(articles), totalCount)

	return nil
}

func runArticlesShow(articleID uint) error {
	ctx := context.Background()

	type ArticleWithFeed struct {
		models.Article
		FeedTitle string `gorm:"column:feed_title"`
	}

	var article ArticleWithFeed
	err := db.WithContext(ctx).
		Table("articles").
		Select("articles.*, feeds.title as feed_title").
		Joins("LEFT JOIN feeds ON articles.feed_id = feeds.id").
		Where("articles.id = ?", articleID).
		First(&article).Error

	if err != nil {
		return fmt.Errorf("article not found: %w", err)
	}

	// Print article details
	fmt.Println()
	fmt.Printf("=== Article #%d ===\n\n", article.ID)
	fmt.Printf("Title:        %s\n", article.Title)
	fmt.Printf("Feed:         %s (ID: %d)\n", article.FeedTitle, article.FeedID)
	fmt.Printf("URL:          %s\n", article.URL)
	fmt.Printf("Published:    %s\n", article.PublishedAt.Format("2006-01-02 15:04:05"))

	// Print AI Summary section
	fmt.Println()
	fmt.Println("--- AI Summary ---")
	if article.ProcessedAt != nil {
		fmt.Printf("Status:       ✓ Processed\n")
		if article.ProcessingModel != nil {
			fmt.Printf("Model:        %s\n", *article.ProcessingModel)
		}
		fmt.Printf("Processed At: %s\n", article.ProcessedAt.Format("2006-01-02 15:04:05"))
		fmt.Println()
		if article.Summary != nil && *article.Summary != "" {
			fmt.Println(*article.Summary)
		} else {
			fmt.Println("(No summary available)")
		}
	} else {
		fmt.Printf("Status:       ⏳ Pending\n")
	}

	fmt.Println()
	return nil
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func formatDate(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	if diff < 24*time.Hour {
		return "Today"
	} else if diff < 48*time.Hour {
		return "Yesterday"
	} else if diff < 7*24*time.Hour {
		return fmt.Sprintf("%d days ago", int(diff.Hours()/24))
	}
	return t.Format("2006-01-02")
}

