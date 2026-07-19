package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"

	"github.com/crypto-market-platform/internal/cache"
	"github.com/crypto-market-platform/internal/config"
	"github.com/crypto-market-platform/internal/provider"
	"github.com/crypto-market-platform/internal/repository"
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

	prov := provider.NewCoinGeckoProvider(cfg.ProviderBaseURL, cfg.ProviderTimeout)

	ingestor := worker.NewIngestor(
		coinRepo,
		snapshotRepo,
		syncLogRepo,
		prov,
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

	logger.Info("starting ingestor",
		slog.Duration("interval", cfg.IngestionInterval),
		slog.String("provider", prov.Name()),
	)

	sched.Start(runCtx)
	logger.Info("ingestor stopped gracefully")
}
