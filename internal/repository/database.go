package repository

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/Fancu1/phoenix-rss/internal/config"
)

func InitDB(cfg *config.DatabaseConfig) *gorm.DB {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode)

	fmt.Println("--- Database Initialization Start ---")
	fmt.Printf("Final DSN to be used: user=%s password=*** host=%s port=%d dbname=%s sslmode=%s\n", cfg.User, cfg.Host, cfg.Port, cfg.Database, cfg.SSLMode)

	basicDB, err := gorm.Open(postgres.Open(dsn),
		&gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		},
	)
	if err != nil {
		panic("failed to connect database: " + err.Error())
	}

	return basicDB
}
