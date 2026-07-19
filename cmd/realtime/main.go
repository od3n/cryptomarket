package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/crypto-market-platform/internal/config"
	"github.com/crypto-market-platform/internal/realtime"
	"github.com/crypto-market-platform/internal/stream"
	"github.com/crypto-market-platform/internal/subscriber"
	"github.com/crypto-market-platform/internal/telemetry"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "configuration error: %v\n", err)
		os.Exit(1)
	}

	logger := telemetry.NewLogger(cfg.LogLevel, "realtime-gateway")
	slog.SetDefault(logger)

	logger.Info("starting realtime gateway",
		slog.String("redis_address", cfg.RedisAddress),
		slog.Int("http_port", cfg.HTTPPort),
		slog.String("log_level", cfg.LogLevel))

	// Connect to Redis.
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddress,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	defer redisClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Error("failed to connect to redis", slog.String("error", err.Error()))
		os.Exit(1)
	}
	cancel()
	logger.Info("connected to redis", slog.String("address", cfg.RedisAddress))

	// Initialize metrics.
	metrics := realtime.NewMetrics()

	// Initialize subscriber hub.
	hubCfg := subscriber.DefaultHubConfig()
	hub := subscriber.NewHub(hubCfg, metrics, logger)

	// Initialize realtime server.
	server := realtime.NewServer(hub, redisClient, metrics, logger)

	// Initialize stream consumer.
	consumerCfg := stream.DefaultConsumerConfig()
	consumer := stream.NewConsumer(redisClient, consumerCfg, server.HandleEvent, metrics, logger)

	// Ensure consumer group exists.
	initCtx, initCancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := consumer.EnsureGroup(initCtx); err != nil {
		logger.Error("failed to initialize consumer group", slog.String("error", err.Error()))
		os.Exit(1)
	}
	initCancel()

	// Create HTTP server.
	addr := fmt.Sprintf(":%d", cfg.HTTPPort)
	srv := &http.Server{
		Addr:         addr,
		Handler:      server.Router(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 0, // SSE requires no write timeout.
		IdleTimeout:  120 * time.Second,
	}

	// Start stream consumer in background.
	runCtx, runCancel := context.WithCancel(context.Background())
	defer runCancel()

	go func() {
		if err := consumer.Run(runCtx); err != nil && err != context.Canceled {
			logger.Error("stream consumer error", slog.String("error", err.Error()))
		}
	}()

	// Start HTTP server in background.
	go func() {
		logger.Info("starting HTTP server", slog.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	// Graceful shutdown.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down realtime gateway")
	runCancel() // Stop consumer.

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown error", slog.String("error", err.Error()))
	}

	hub.CloseAll()
	logger.Info("realtime gateway stopped gracefully")
}
