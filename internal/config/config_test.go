package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad_ValidConfig(t *testing.T) {
	os.Setenv("POSTGRES_DSN", "postgres://user:pass@localhost:5432/db?sslmode=disable")
	os.Setenv("HTTP_PORT", "9090")
	os.Setenv("INGESTION_INTERVAL", "30s")
	os.Setenv("PROVIDER_TIMEOUT", "5s")
	defer func() {
		os.Unsetenv("POSTGRES_DSN")
		os.Unsetenv("HTTP_PORT")
		os.Unsetenv("INGESTION_INTERVAL")
		os.Unsetenv("PROVIDER_TIMEOUT")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.HTTPPort != 9090 {
		t.Errorf("expected HTTPPort 9090, got %d", cfg.HTTPPort)
	}
	if cfg.IngestionInterval != 30*time.Second {
		t.Errorf("expected IngestionInterval 30s, got %s", cfg.IngestionInterval)
	}
	if cfg.ProviderTimeout != 5*time.Second {
		t.Errorf("expected ProviderTimeout 5s, got %s", cfg.ProviderTimeout)
	}
}

func TestLoad_MissingPostgresDSN(t *testing.T) {
	os.Unsetenv("POSTGRES_DSN")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing POSTGRES_DSN")
	}
}

func TestLoad_InvalidHTTPPort(t *testing.T) {
	tests := []struct {
		name string
		port string
	}{
		{"zero", "0"},
		{"negative", "-1"},
		{"too large", "99999"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("POSTGRES_DSN", "postgres://user:pass@localhost/db")
			os.Setenv("HTTP_PORT", tt.port)
			defer func() {
				os.Unsetenv("POSTGRES_DSN")
				os.Unsetenv("HTTP_PORT")
			}()

			_, err := Load()
			if err == nil {
				t.Errorf("expected error for port %s", tt.port)
			}
		})
	}
}

func TestLoad_InvalidIngestionInterval(t *testing.T) {
	os.Setenv("POSTGRES_DSN", "postgres://user:pass@localhost/db")
	os.Setenv("INGESTION_INTERVAL", "2s")
	defer func() {
		os.Unsetenv("POSTGRES_DSN")
		os.Unsetenv("INGESTION_INTERVAL")
	}()

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for ingestion interval < 5s")
	}
}

func TestLoad_Defaults(t *testing.T) {
	os.Setenv("POSTGRES_DSN", "postgres://user:pass@localhost/db")
	defer os.Unsetenv("POSTGRES_DSN")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.AppEnv != "development" {
		t.Errorf("expected default AppEnv 'development', got %s", cfg.AppEnv)
	}
	if cfg.ServiceName != "market-api" {
		t.Errorf("expected default ServiceName 'market-api', got %s", cfg.ServiceName)
	}
	if cfg.HTTPPort != 8080 {
		t.Errorf("expected default HTTPPort 8080, got %d", cfg.HTTPPort)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected default LogLevel 'info', got %s", cfg.LogLevel)
	}
	if cfg.RedisAddress != "localhost:6379" {
		t.Errorf("expected default RedisAddress 'localhost:6379', got %s", cfg.RedisAddress)
	}
	if cfg.IngestionInterval != 60*time.Second {
		t.Errorf("expected default IngestionInterval 60s, got %s", cfg.IngestionInterval)
	}
}
