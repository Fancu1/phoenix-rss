package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config is the main config for the application
type Config struct {
	Server      ServerConfig      `mapstructure:"server"`
	Database    DatabaseConfig    `mapstructure:"database"`
	Redis       RedisConfig       `mapstructure:"redis"`
	Auth        AuthConfig        `mapstructure:"auth"`
	Kafka       KafkaConfig       `mapstructure:"kafka"`
	UserService UserServiceConfig `mapstructure:"user_service"`
}

// ServerConfig is the config for the server
type ServerConfig struct {
	Port int `mapstructure:"port"`
}

// DatabaseConfig is the config for the database
type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
	SSLMode  string `mapstructure:"sslmode"`
}

type RedisConfig struct {
	Address string `mapstructure:"address"`
}

type AuthConfig struct {
	JWTSecret string `mapstructure:"jwt_secret"`
}

// KafkaConfig holds Kafka connectivity and topics
type KafkaConfig struct {
	Brokers []string `mapstructure:"brokers"`
	Topic   string   `mapstructure:"topic"`
	GroupID string   `mapstructure:"group_id"`
}

type UserServiceConfig struct {
	Address string `mapstructure:"address"`
}

// LoadConfig loads the configuration from file and environment variables
// Environment variables take precedence over file values
// Environment variables should be prefixed with PHOENIX_RSS_ (e.g., PHOENIX_RSS_DATABASE_HOST)
func LoadConfig() (*Config, error) {
	v := viper.New()

	// Set configuration file search parameters
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	// Add multiple search paths for flexibility
	// This allows the application to work from different working directories
	v.AddConfigPath("./configs")        // Standard location
	v.AddConfigPath(".")                // Current directory
	v.AddConfigPath("/etc/phoenix-rss") // System-wide config (for production)

	// Configure environment variable support
	v.SetEnvPrefix("PHOENIX_RSS")
	v.AutomaticEnv()

	// Replace dots with underscores in environment variable names
	// This allows database.host to be set via PHOENIX_RSS_DATABASE_HOST
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set default values
	setDefaults(v)

	// Try to read configuration file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Configuration file not found; this is acceptable in container environments
			// where all configuration comes from environment variables
		} else {
			// Configuration file was found but another error was produced
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode config into struct: %w", err)
	}

	// Validate and apply post-processing
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// setDefaults configures default values for the application
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.port", 8080)

	// Database defaults
	v.SetDefault("database.host", "127.0.0.1")
	v.SetDefault("database.port", "15432")
	v.SetDefault("database.user", "postgres")
	v.SetDefault("database.password", "password")
	v.SetDefault("database.database", "phoenix_rss")
	v.SetDefault("database.sslmode", "disable")

	// Redis defaults
	v.SetDefault("redis.address", "127.0.0.1:6379")

	// Auth defaults
	v.SetDefault("auth.jwt_secret", "phoenix-rss-default-secret-please-change-in-production")

	// Kafka defaults
	v.SetDefault("kafka.brokers", []string{"127.0.0.1:19092"})
	v.SetDefault("kafka.topic", "feed.fetch")
	v.SetDefault("kafka.group_id", "phoenix-rss-worker")

	// User Service defaults
	v.SetDefault("user_service.address", "127.0.0.1:50051")
}

// validate performs basic validation on the loaded configuration
func (c *Config) validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.Database.Host == "" {
		return fmt.Errorf("database host cannot be empty")
	}

	if c.Database.Database == "" {
		return fmt.Errorf("database name cannot be empty")
	}

	if c.Redis.Address == "" {
		return fmt.Errorf("redis address cannot be empty")
	}

	if c.Auth.JWTSecret == "" {
		return fmt.Errorf("JWT secret cannot be empty")
	}

	if len(c.Kafka.Brokers) == 0 {
		return fmt.Errorf("kafka brokers cannot be empty")
	}
	if c.Kafka.Topic == "" {
		return fmt.Errorf("kafka topic for feed fetch cannot be empty")
	}
	if c.Kafka.GroupID == "" {
		return fmt.Errorf("kafka group id cannot be empty")
	}

	if c.UserService.Address == "" {
		return fmt.Errorf("user service address cannot be empty")
	}

	// Warn about default JWT secret in a production environment
	if c.Auth.JWTSecret == "phoenix-rss-default-secret-please-change-in-production" {
		// Note: In a real application, you might want to use a logger here
		// For now, this serves as documentation of the requirement
	}

	return nil
}
