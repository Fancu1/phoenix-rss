This file is a merged representation of a subset of the codebase, containing specifically included files, combined into a single document by Repomix.

# File Summary

## Purpose
This file contains a packed representation of the entire repository's contents.
It is designed to be easily consumable by AI systems for analysis, code review,
or other automated processes.

## File Format
The content is organized as follows:
1. This summary section
2. Repository information
3. Directory structure
4. Repository files (if enabled)
5. Multiple file entries, each consisting of:
  a. A header with the file path (## File: path/to/file)
  b. The full contents of the file in a code block

## Usage Guidelines
- This file should be treated as read-only. Any changes should be made to the
  original repository files, not this packed version.
- When processing this file, use the file path to distinguish
  between different files in the repository.
- Be aware that this file may contain sensitive information. Handle it with
  the same level of security as you would the original repository.

## Notes
- Some files may have been excluded based on .gitignore rules and Repomix's configuration
- Binary files are not included in this packed representation. Please refer to the Repository Structure section for a complete list of file paths, including binary files
- Only files matching these patterns are included: internal/config/config.go, internal/repository/database.go, configs/config.yaml
- Files matching patterns in .gitignore are excluded
- Files matching default ignore patterns are excluded
- Files are sorted by Git change count (files with more changes are at the bottom)

# Directory Structure
```
configs/
  config.yaml
internal/
  config/
    config.go
  repository/
    database.go
```

# Files

## File: internal/repository/database.go
```go
package repository

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/Fancu1/phoenix-rss/internal/config"
)

func InitDB(cfg *config.DatabaseConfig) *gorm.DB {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode)

	pgxConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		panic("failed to parse pgx config: " + err.Error())
	}

	sqlDB := stdlib.OpenDB(*pgxConfig.ConnConfig)

	fmt.Println("Database connection successful")

	db, err := gorm.Open(postgres.New(postgres.Config{
		Conn: sqlDB,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		panic("failed to connect database: " + err.Error())
	}

	if _, err = db.DB(); err != nil {
		panic("failed to get underlying sql.DB: " + err.Error())
	}

	err = db.Raw("SELECT 1").Error
	if err != nil {
		panic("database connection check failed: " + err.Error())
	}

	return db
}
```

## File: configs/config.yaml
```yaml
server:
  port: 8080

database:
  host: "127.0.0.1"
  port: 5433
  user: "myuser"
  password: "mysecretpassword"
  database: "phoenix-rss"
  sslmode: "disable"
```

## File: internal/config/config.go
```go
package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
	SSLMode  string `yaml:"sslmode"`
}

// Config is the main config for the application
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
}

// ServerConfig is the config for the server
type ServerConfig struct {
	Port int `yaml:"port"`
}

// LoadConfig loads the config from the file
func LoadConfig(path string) (*Config, error) {
	// read the config file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// parse the config
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
```
