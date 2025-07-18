package repository

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/Fancu1/phoenix-rss/internal/config"
	"github.com/Fancu1/phoenix-rss/internal/models"
)

func InitDB(cfg *config.DatabaseConfig) *gorm.DB {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		cfg.Host, cfg.User, cfg.Password, cfg.Database, cfg.Port, cfg.SSLMode)
	fmt.Printf("Connecting to database: %s\n", dsn)

	basicDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	return basicDB
}

func InitSchema(db *gorm.DB) error {
	return db.AutoMigrate(&models.Feed{}, &models.Article{}, &models.User{}, &models.Subscription{})
}
