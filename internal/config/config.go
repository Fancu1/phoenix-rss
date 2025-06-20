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
