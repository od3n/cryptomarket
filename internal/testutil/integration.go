//go:build integration

// Package integration provides test infrastructure for integration tests
// using real PostgreSQL and Redis instances via testcontainers.
//
// Integration tests are tagged with the "integration" build tag and run
// separately from unit tests:
//
//	go test -tags=integration -run Integration ./...
//
// Or via Makefile:
//
//	make test-integration
package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestEnv holds the test infrastructure for integration tests.
type TestEnv struct {
	PostgresDSN   string
	RedisAddress  string
	postgresCtr   testcontainers.Container
	redisCtr      testcontainers.Container
}

// SetupTestEnv starts PostgreSQL and Redis containers for integration testing.
// Containers are automatically terminated when the test completes.
func SetupTestEnv(t *testing.T) *TestEnv {
	t.Helper()
	ctx := context.Background()
	env := &TestEnv{}

	// Start PostgreSQL
	pgCtr, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("cryptomarket_test"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start PostgreSQL container: %v", err)
	}
	env.postgresCtr = pgCtr

	pgHost, err := pgCtr.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get PostgreSQL host: %v", err)
	}
	pgPort, err := pgCtr.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("failed to get PostgreSQL port: %v", err)
	}
	env.PostgresDSN = fmt.Sprintf(
		"postgres://testuser:testpass@%s:%s/cryptomarket_test?sslmode=disable",
		pgHost, pgPort.Port(),
	)

	// Start Redis
	redisCtr, err := redis.Run(ctx,
		"redis:7-alpine",
		testcontainers.WithWaitStrategy(
			wait.ForLog("Ready to accept connections").
				WithStartupTimeout(15*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start Redis container: %v", err)
	}
	env.redisCtr = redisCtr

	redisHost, err := redisCtr.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get Redis host: %v", err)
	}
	redisPort, err := redisCtr.MappedPort(ctx, "6379")
	if err != nil {
		t.Fatalf("failed to get Redis port: %v", err)
	}
	env.RedisAddress = fmt.Sprintf("%s:%s", redisHost, redisPort.Port())

	// Register cleanup
	t.Cleanup(func() {
		if err := pgCtr.Terminate(ctx); err != nil {
			t.Logf("warning: failed to terminate PostgreSQL container: %v", err)
		}
		if err := redisCtr.Terminate(ctx); err != nil {
			t.Logf("warning: failed to terminate Redis container: %v", err)
		}
	})

	t.Logf("Integration test environment ready:")
	t.Logf("  PostgreSQL: %s", env.PostgresDSN)
	t.Logf("  Redis: %s", env.RedisAddress)

	return env
}
