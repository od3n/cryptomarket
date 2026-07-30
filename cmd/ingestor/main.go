package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"

	"github.com/crypto-market-platform/internal/cache"
	"github.com/crypto-market-platform/internal/config"
	"github.com/crypto-market-platform/internal/provider"
	"github.com/crypto-market-platform/internal/repository"
	"github.com/crypto-market-platform/internal/resilience"
	"github.com/crypto-market-platform/internal/scheduler"
	"github.com/crypto-market-platform/internal/telemetry"
	"github.com/crypto-market-platform/internal/worker"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "configuration error: %v\n", err)
		os.Exit(1)
	}

	logger := telemetry.NewLogger(cfg.LogLevel, "market-ingestor")
	slog.SetDefault(logger)

	// Connect to PostgreSQL.
	db, err := sql.Open("postgres", cfg.PostgresDSN)
	if err != nil {
		logger.Error("failed to open database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer db.Close()

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := db.PingContext(ctx); err != nil {
		logger.Error("failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	cancel()

	// Connect to Redis.
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddress,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	defer redisClient.Close()

	// Initialize dependencies.
	metrics := telemetry.NewMetrics()
	marketCache := cache.NewMarketCache(redisClient)
	coinRepo := repository.NewPostgresCoinRepository(db)
	snapshotRepo := repository.NewPostgresSnapshotRepository(db)
	syncLogRepo := repository.NewPostgresSyncLogRepository(db)

	// Build provider chain with fallback support.
	registry := provider.NewRegistry()
	registry.SetBaseURL("coingecko", cfg.ProviderBaseURL)
	registry.SetBaseURL("coincap", cfg.CoinCapBaseURL)

	// Create providers in priority order.
	providerNames := append([]string{cfg.ProviderPrimary}, cfg.ProviderFallback...)
	providers := make([]provider.Provider, 0, len(providerNames))
	for _, name := range providerNames {
		p, err := registry.Create(name, cfg.ProviderBaseURL, cfg.ProviderTimeout)
		if err != nil {
			logger.Warn("failed to create provider, skipping",
				slog.String("provider", name),
				slog.String("error", err.Error()))
			continue
		}
		providers = append(providers, p)
	}

	if len(providers) == 0 {
		logger.Error("no providers available")
		os.Exit(1)
	}

	// Create fallback orchestrator.
	selector := provider.NewSelector(providers, cfg.ProviderDisabled)
	cbManager := resilience.NewManager(resilience.CircuitBreakerConfig{
		FailureThreshold: cfg.CircuitBreakerFailureThreshold,
		OpenDuration:     cfg.CircuitBreakerOpenDuration,
		SuccessThreshold: cfg.CircuitBreakerSuccessThreshold,
	})
	rateTracker := resilience.NewRateLimitTracker()
	retryConfig := resilience.RetryConfig{
		MaxAttempts: cfg.RetryMaxAttempts,
		BaseDelay:   cfg.RetryBaseDelay,
		MaxDelay:    cfg.RetryMaxDelay,
	}

	orchestrator := provider.NewFallbackOrchestrator(
		selector,
		cbManager,
		rateTracker,
		retryConfig,
		logger,
	)

	ingestor := worker.NewIngestorWithFallback(
		coinRepo,
		snapshotRepo,
		syncLogRepo,
		orchestrator,
		marketCache,
		metrics,
		logger,
	)

	// Create scheduler.
	sched := scheduler.New(cfg.IngestionInterval, ingestor.RunCycle, logger)

	// Run scheduler with graceful shutdown.
	runCtx, runCancel := context.WithCancel(context.Background())
	defer runCancel()

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		logger.Info("received shutdown signal")
		runCancel()
	}()

	// Start metrics/health HTTP server for Prometheus scraping.
	metricsPort := os.Getenv("METRICS_PORT")
	if metricsPort == "" {
		metricsPort = "8082"
	}
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	metricsMux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})
	metricsServer := &http.Server{
		Addr:    ":" + metricsPort,
		Handler: metricsMux,
	}
	go func() {
		logger.Info("metrics server started", slog.String("port", metricsPort))
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("metrics server failed", slog.String("error", err.Error()))
		}
	}()

	logger.Info("starting ingestor",
		slog.Duration("interval", cfg.IngestionInterval),
		slog.String("primary_provider", cfg.ProviderPrimary),
		slog.Any("fallback_providers", cfg.ProviderFallback),
	)

	sched.Start(runCtx)

	// Gracefully shut down the metrics server.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("metrics server shutdown failed", slog.Any("error", err))
	}

	logger.Info("ingestor stopped gracefully")
}
