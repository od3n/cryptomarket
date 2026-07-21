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
	"github.com/redis/go-redis/v9"

	"github.com/crypto-market-platform/internal/api"
	"github.com/crypto-market-platform/internal/cache"
	"github.com/crypto-market-platform/internal/config"
	"github.com/crypto-market-platform/internal/repository"
	"github.com/crypto-market-platform/internal/telemetry"
)

func main() {
	// Handle migrate/seed subcommands.
	if len(os.Args) > 1 {
		handleSubcommand(os.Args[1])
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "configuration error: %v\n", err)
		os.Exit(1)
	}

	logger := telemetry.NewLogger(cfg.LogLevel, cfg.ServiceName)
	slog.SetDefault(logger)

	// Connect to PostgreSQL.
	db, err := sql.Open("postgres", cfg.PostgresDSN)
	if err != nil {
		logger.Error("failed to open database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer db.Close()

	db.SetMaxOpenConns(25)
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

	handler := api.NewHandler(db, coinRepo, snapshotRepo, syncLogRepo, marketCache, logger)
	middleware := api.NewMiddleware(logger, metrics)
	authCfg := api.NewAuthConfig(cfg.AuthEnabled, cfg.AuthAPIKeys, cfg.AuthJWTSecret, cfg.AuthJWTIssuer)
	router := api.NewRouter(handler, middleware, authCfg)

	// Create HTTP server.
	addr := fmt.Sprintf(":%d", cfg.HTTPPort)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in background.
	go func() {
		logger.Info("starting API server", slog.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	// Graceful shutdown.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down API server")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown error", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Info("server stopped gracefully")
}

func handleSubcommand(cmd string) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "configuration error: %v\n", err)
		os.Exit(1)
	}

	db, err := sql.Open("postgres", cfg.PostgresDSN)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	ctx := context.Background()

	switch cmd {
	case "migrate":
		if err := repository.Migrate(ctx, db, "migrations"); err != nil {
			fmt.Fprintf(os.Stderr, "migration failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("migrations complete")
	case "migrate-down":
		if err := repository.MigrateDown(ctx, db, "migrations"); err != nil {
			fmt.Fprintf(os.Stderr, "migration down failed: %v\n", err)
			os.Exit(1)
		}
	case "seed":
		// Seed is handled by migration 002_seed_coins.up.sql
		if err := repository.Migrate(ctx, db, "migrations"); err != nil {
			fmt.Fprintf(os.Stderr, "seed failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("seed complete")
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		os.Exit(1)
	}
}
