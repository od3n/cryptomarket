package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration.
type Config struct {
	AppEnv      string
	ServiceName string
	HTTPPort    int
	LogLevel    string

	PostgresDSN string

	RedisAddress  string
	RedisPassword string
	RedisDB       int

	IngestionInterval time.Duration
	ProviderBaseURL   string
	ProviderTimeout   time.Duration
}

// Load reads configuration from environment variables and returns a validated Config.
// It fails fast if mandatory configuration is missing or invalid.
func Load() (*Config, error) {
	cfg := &Config{
		AppEnv:            getEnv("APP_ENV", "development"),
		ServiceName:       getEnv("SERVICE_NAME", "market-api"),
		HTTPPort:          getEnvInt("HTTP_PORT", 8080),
		LogLevel:          getEnv("LOG_LEVEL", "info"),
		PostgresDSN:       getEnv("POSTGRES_DSN", ""),
		RedisAddress:      getEnv("REDIS_ADDRESS", "localhost:6379"),
		RedisPassword:     getEnv("REDIS_PASSWORD", ""),
		RedisDB:           getEnvInt("REDIS_DB", 0),
		IngestionInterval: getEnvDuration("INGESTION_INTERVAL", 60*time.Second),
		ProviderBaseURL:   getEnv("PROVIDER_BASE_URL", "https://api.coingecko.com/api/v3"),
		ProviderTimeout:   getEnvDuration("PROVIDER_TIMEOUT", 10*time.Second),
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config validation: %w", err)
	}

	return cfg, nil
}

// validate checks mandatory configuration values.
func (c *Config) validate() error {
	if c.PostgresDSN == "" {
		return fmt.Errorf("POSTGRES_DSN is required")
	}
	if c.HTTPPort < 1 || c.HTTPPort > 65535 {
		return fmt.Errorf("HTTP_PORT must be between 1 and 65535, got %d", c.HTTPPort)
	}
	if c.IngestionInterval < 5*time.Second {
		return fmt.Errorf("INGESTION_INTERVAL must be at least 5s, got %s", c.IngestionInterval)
	}
	if c.ProviderTimeout < 1*time.Second {
		return fmt.Errorf("PROVIDER_TIMEOUT must be at least 1s, got %s", c.ProviderTimeout)
	}
	if c.RedisAddress == "" {
		return fmt.Errorf("REDIS_ADDRESS is required")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
