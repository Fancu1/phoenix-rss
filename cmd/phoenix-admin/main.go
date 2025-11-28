package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gorm.io/gorm"

	"github.com/Fancu1/phoenix-rss/internal/config"
	"github.com/Fancu1/phoenix-rss/internal/feed-service/repository"
)

var (
	// Global database connection
	db *gorm.DB
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "phoenix-admin",
		Short: "Phoenix RSS Admin CLI",
		Long:  `A command-line tool for managing Phoenix RSS articles, feeds, and AI processing.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Skip initialization for help commands
			if cmd.Name() == "help" || cmd.Name() == "completion" {
				return nil
			}

			// Load configuration
			cfg, err := config.LoadConfig()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Initialize database connection
			db = repository.InitDB(&cfg.Database)
			return nil
		},
	}

	// Add subcommands
	rootCmd.AddCommand(newArticlesCmd())
	rootCmd.AddCommand(newAICmd())
	rootCmd.AddCommand(newFeedsCmd())
	rootCmd.AddCommand(newStatsCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

