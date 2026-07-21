package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
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

	// Provider selection
	ProviderPrimary  string
	ProviderFallback []string
	ProviderDisabled []string

	// CoinCap provider
	CoinCapBaseURL string

	// Circuit breaker
	CircuitBreakerFailureThreshold int
	CircuitBreakerOpenDuration     time.Duration
	CircuitBreakerSuccessThreshold int

	// Retry
	RetryMaxAttempts int
	RetryBaseDelay   time.Duration
	RetryMaxDelay    time.Duration

	// Freshness
	FreshnessThreshold time.Duration
	StaleThreshold     time.Duration

	// Authentication
	AuthEnabled   bool
	AuthAPIKeys   []string
	AuthJWTSecret string
	AuthJWTIssuer string
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

		ProviderPrimary:  getEnv("PROVIDER_PRIMARY", "coingecko"),
		ProviderFallback: getEnvList("PROVIDER_FALLBACK", []string{"coincap"}),
		ProviderDisabled: getEnvList("PROVIDER_DISABLED", nil),

		CoinCapBaseURL: getEnv("COINCAP_BASE_URL", "https://api.coincap.io"),

		CircuitBreakerFailureThreshold: getEnvInt("CB_FAILURE_THRESHOLD", 5),
		CircuitBreakerOpenDuration:     getEnvDuration("CB_OPEN_DURATION", 30*time.Second),
		CircuitBreakerSuccessThreshold: getEnvInt("CB_SUCCESS_THRESHOLD", 2),

		RetryMaxAttempts: getEnvInt("RETRY_MAX_ATTEMPTS", 3),
		RetryBaseDelay:   getEnvDuration("RETRY_BASE_DELAY", 500*time.Millisecond),
		RetryMaxDelay:    getEnvDuration("RETRY_MAX_DELAY", 10*time.Second),

		FreshnessThreshold: getEnvDuration("FRESHNESS_THRESHOLD", 120*time.Second),
		StaleThreshold:     getEnvDuration("STALE_THRESHOLD", 300*time.Second),

		AuthEnabled:   getEnv("AUTH_ENABLED", "false") == "true",
		AuthAPIKeys:   getEnvList("AUTH_API_KEYS", nil),
		AuthJWTSecret: getEnv("AUTH_JWT_SECRET", ""),
		AuthJWTIssuer: getEnv("AUTH_JWT_ISSUER", "cryptomarket"),
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

func getEnvList(key string, fallback []string) []string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parts := strings.Split(v, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	if len(result) == 0 {
		return fallback
	}
	return result
}
