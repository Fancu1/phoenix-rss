package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config is the main config for the application
type Config struct {
	Server           ServerConfig           `mapstructure:"server"`
	Database         DatabaseConfig         `mapstructure:"database"`
	Redis            RedisConfig            `mapstructure:"redis"`
	Auth             AuthConfig             `mapstructure:"auth"`
	Kafka            KafkaConfig            `mapstructure:"kafka"`
	UserService      UserServiceConfig      `mapstructure:"user_service"`
	FeedService      FeedServiceConfig      `mapstructure:"feed_service"`
	SchedulerService SchedulerServiceConfig `mapstructure:"scheduler_service"`
	AIService        AIServiceConfig        `mapstructure:"ai_service"`
}

// ServerConfig is the config for the server
type ServerConfig struct {
	Port int `mapstructure:"port"`
}

// DatabaseConfig is the config for the database
type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

type RedisConfig struct {
	Address string `mapstructure:"address"`
}

type AuthConfig struct {
	JWTSecret string `mapstructure:"jwt_secret"`
}

// KafkaConfig hold Kafka connectivity and topic configurations
type KafkaConfig struct {
	Brokers      []string                `mapstructure:"brokers"`
	FeedFetch    FeedFetchKafkaConfig    `mapstructure:"feed_fetch"`
	AIProcessing AIProcessingKafkaConfig `mapstructure:"ai_processing"`
}

// FeedFetchKafkaConfig config for feed fetching workflow (scheduler -> feed service)
type FeedFetchKafkaConfig struct {
	Topic              string `mapstructure:"topic"`
	FeedServiceGroupID string `mapstructure:"feed_service_group_id"`
}

// AIProcessingKafkaConfig config for AI processing workflow (feed service -> ai service -> feed service)
type AIProcessingKafkaConfig struct {
	ArticlesNewTopic       string `mapstructure:"articles_new_topic"`
	ArticlesProcessedTopic string `mapstructure:"articles_processed_topic"`
	AIServiceGroupID       string `mapstructure:"ai_service_group_id"`
	FeedServiceAIGroupID   string `mapstructure:"feed_service_ai_group_id"`
}

type UserServiceConfig struct {
	Address string `mapstructure:"address"`
}

type FeedServiceConfig struct {
	Port    int    `mapstructure:"port"`
	Address string `mapstructure:"address"`
}

type SchedulerServiceConfig struct {
	Schedule      string `mapstructure:"schedule"`
	BatchSize     int    `mapstructure:"batch_size"`
	BatchDelay    string `mapstructure:"batch_delay"`
	MaxConcurrent int    `mapstructure:"max_concurrent"`
}

type AIServiceConfig struct {
	LLMBaseURL     string `mapstructure:"llm_base_url"`
	LLMAPIKey      string `mapstructure:"llm_api_key"`
	LLMModel       string `mapstructure:"llm_model"`
	RequestTimeout string `mapstructure:"request_timeout"`
}

// LoadConfig loads the configuration with the following priority:
// 1. Environment variables (e.g., from .env file or system)
// 2. Default values set in the code.
func LoadConfig() (*Config, error) {
	v := viper.New()

	// Step 1: Set default values. This is the lowest priority.
	setDefaults(v)

	// Step 2 (Optional): Load .env file. This will override defaults.
	// We look in the current directory for the .env file.
	v.SetConfigName(".env")
	v.SetConfigType("env")
	v.AddConfigPath(".")
	if err := v.MergeInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Only return an error if the file was found but couldn't be read.
			// If the file is not found, we can proceed with defaults/env vars.
			return nil, fmt.Errorf("error reading .env file: %w", err)
		}
	}

	// Step 3: Enable reading from environment variables.
	// This has the highest priority and will override .env and defaults.
	// e.g., DATABASE_HOST will override the value in .env.
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Bind specific environment variables to their corresponding config keys.
	// This ensures that `v.Unmarshal` works correctly with AutomaticEnv.
	bindEnvironmentVariables(v)

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode config into struct: %w", err)
	}

	// Handle special parsing for complex types
	if err := config.postProcess(v); err != nil {
		return nil, fmt.Errorf("config post-processing failed: %w", err)
	}

	// Validate configuration
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
	v.SetDefault("database.port", 15432)
	v.SetDefault("database.user", "postgres")
	v.SetDefault("database.password", "password")
	v.SetDefault("database.dbname", "phoenix_rss")
	v.SetDefault("database.sslmode", "disable")

	// Redis defaults
	v.SetDefault("redis.address", "127.0.0.1:6379")

	// Auth defaults
	v.SetDefault("auth.jwt_secret", "phoenix-rss-default-secret-please-change-in-production")

	// Kafka defaults
	v.SetDefault("kafka.brokers", []string{"127.0.0.1:19092"})

	// Feed fetch workflow defaults
	v.SetDefault("kafka.feed_fetch.topic", "feed.fetch")
	v.SetDefault("kafka.feed_fetch.feed_service_group_id", "feed-service-group")

	// AI processing workflow defaults
	v.SetDefault("kafka.ai_processing.articles_new_topic", "articles.new")
	v.SetDefault("kafka.ai_processing.articles_processed_topic", "articles.processed")
	v.SetDefault("kafka.ai_processing.ai_service_group_id", "ai-service-group")
	v.SetDefault("kafka.ai_processing.feed_service_ai_group_id", "feed-service-ai-group")

	// User Service defaults
	v.SetDefault("user_service.address", "127.0.0.1:50051")

	// Feed Service defaults
	v.SetDefault("feed_service.port", 50053)
	v.SetDefault("feed_service.address", "127.0.0.1:50053")

	// Scheduler Service defaults
	v.SetDefault("scheduler_service.schedule", "@every 30m")
	v.SetDefault("scheduler_service.batch_size", 20)
	v.SetDefault("scheduler_service.batch_delay", "5s")
	v.SetDefault("scheduler_service.max_concurrent", 5)

	// AI Service defaults
	v.SetDefault("ai_service.llm_base_url", "https://api.openai.com")
	v.SetDefault("ai_service.llm_api_key", "sk-proj-1234567890")
	v.SetDefault("ai_service.llm_model", "gpt-4o-mini")
	v.SetDefault("ai_service.request_timeout", "30s")
}

// validate performs basic validation on the loaded configuration
func (c *Config) validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.Database.Host == "" {
		return fmt.Errorf("database host cannot be empty")
	}

	if c.Database.DBName == "" {
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

	// Validate feed fetch kafka config
	if c.Kafka.FeedFetch.Topic == "" {
		return fmt.Errorf("kafka feed fetch topic cannot be empty")
	}
	if c.Kafka.FeedFetch.FeedServiceGroupID == "" {
		return fmt.Errorf("kafka feed service group ID cannot be empty")
	}

	// Validate AI processing kafka config
	if c.Kafka.AIProcessing.ArticlesNewTopic == "" {
		return fmt.Errorf("kafka articles new topic cannot be empty")
	}
	if c.Kafka.AIProcessing.ArticlesProcessedTopic == "" {
		return fmt.Errorf("kafka articles processed topic cannot be empty")
	}
	if c.Kafka.AIProcessing.AIServiceGroupID == "" {
		return fmt.Errorf("kafka AI service group ID cannot be empty")
	}
	if c.Kafka.AIProcessing.FeedServiceAIGroupID == "" {
		return fmt.Errorf("kafka feed service AI group ID cannot be empty")
	}

	if c.UserService.Address == "" {
		return fmt.Errorf("user service address cannot be empty")
	}

	if c.FeedService.Port <= 0 || c.FeedService.Port > 65535 {
		return fmt.Errorf("invalid feed service port: %d", c.FeedService.Port)
	}

	if c.FeedService.Address == "" {
		return fmt.Errorf("feed service address cannot be empty")
	}

	if c.SchedulerService.Schedule == "" {
		return fmt.Errorf("scheduler service schedule cannot be empty")
	}

	if c.SchedulerService.BatchSize <= 0 {
		return fmt.Errorf("scheduler service batch size must be positive")
	}

	if c.SchedulerService.MaxConcurrent <= 0 {
		return fmt.Errorf("scheduler service max concurrent must be positive")
	}

	if c.SchedulerService.BatchDelay == "" {
		return fmt.Errorf("scheduler service batch delay cannot be empty")
	}

	if c.AIService.LLMBaseURL == "" {
		return fmt.Errorf("AI service LLM base URL cannot be empty")
	}

	if c.AIService.LLMAPIKey == "" {
		return fmt.Errorf("AI service LLM API key cannot be empty")
	}

	if c.AIService.LLMModel == "" {
		return fmt.Errorf("AI service LLM model cannot be empty")
	}

	if c.AIService.RequestTimeout == "" {
		return fmt.Errorf("AI service request timeout cannot be empty")
	}

	// Warn about default JWT secret in a production environment
	if c.Auth.JWTSecret == "phoenix-rss-default-secret-please-change-in-production" {
		// Note: In a real application, you might want to use a logger here
		// For now, this serves as documentation of the requirement
	}

	return nil
}

// bindEnvironmentVariables binds specific environment variables to handle special cases
func bindEnvironmentVariables(v *viper.Viper) {
	// Bind all the key environment variables
	envBindings := []string{
		"server.port",
		"database.host",
		"database.port",
		"database.user",
		"database.password",
		"database.dbname",
		"database.sslmode",
		"redis.address",
		"auth.jwt_secret",
		"kafka.brokers",
		"kafka.feed_fetch.topic",
		"kafka.feed_fetch.feed_service_group_id",
		"kafka.ai_processing.articles_new_topic",
		"kafka.ai_processing.articles_processed_topic",
		"kafka.ai_processing.ai_service_group_id",
		"kafka.ai_processing.feed_service_ai_group_id",
		"user_service.address",
		"feed_service.port",
		"feed_service.address",
		"scheduler_service.schedule",
		"scheduler_service.batch_size",
		"scheduler_service.batch_delay",
		"scheduler_service.max_concurrent",
		"ai_service.llm_base_url",
		"ai_service.llm_api_key",
		"ai_service.llm_model",
		"ai_service.request_timeout",
	}

	for _, key := range envBindings {
		// This will bind "database.host" to "DATABASE_HOST" environment variable
		v.BindEnv(key)
	}
}

// postProcess handles special parsing for complex types like arrays
func (c *Config) postProcess(v *viper.Viper) error {
	// Handle Kafka brokers - can be comma-separated string or array
	if brokersStr := v.GetString("kafka.brokers"); brokersStr != "" {
		// If it's a comma-separated string, split it
		if strings.Contains(brokersStr, ",") {
			c.Kafka.Brokers = strings.Split(brokersStr, ",")
			// Trim whitespace from each broker
			for i, broker := range c.Kafka.Brokers {
				c.Kafka.Brokers[i] = strings.TrimSpace(broker)
			}
		}
	}

	return nil
}
