package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/crypto-market-platform/internal/cache"
	"github.com/crypto-market-platform/internal/market"
	"github.com/crypto-market-platform/internal/provider"
	"github.com/crypto-market-platform/internal/repository"
	"github.com/crypto-market-platform/internal/telemetry"
)

// Ingestor performs a single ingestion cycle: fetch, validate, persist, cache, publish.
type Ingestor struct {
	coinRepo     repository.CoinRepository
	snapshotRepo repository.SnapshotRepository
	syncLogRepo  repository.SyncLogRepository
	provider     provider.Provider
	orchestrator *provider.FallbackOrchestrator
	cache        *cache.MarketCache
	metrics      *telemetry.Metrics
	logger       *slog.Logger
}

// NewIngestor creates a new Ingestor.
func NewIngestor(
	coinRepo repository.CoinRepository,
	snapshotRepo repository.SnapshotRepository,
	syncLogRepo repository.SyncLogRepository,
	prov provider.Provider,
	cache *cache.MarketCache,
	metrics *telemetry.Metrics,
	logger *slog.Logger,
) *Ingestor {
	return &Ingestor{
		coinRepo:     coinRepo,
		snapshotRepo: snapshotRepo,
		syncLogRepo:  syncLogRepo,
		provider:     prov,
		cache:        cache,
		metrics:      metrics,
		logger:       logger,
	}
}

// NewIngestorWithFallback creates an Ingestor with provider fallback support.
func NewIngestorWithFallback(
	coinRepo repository.CoinRepository,
	snapshotRepo repository.SnapshotRepository,
	syncLogRepo repository.SyncLogRepository,
	orchestrator *provider.FallbackOrchestrator,
	cache *cache.MarketCache,
	metrics *telemetry.Metrics,
	logger *slog.Logger,
) *Ingestor {
	return &Ingestor{
		coinRepo:     coinRepo,
		snapshotRepo: snapshotRepo,
		syncLogRepo:  syncLogRepo,
		orchestrator: orchestrator,
		cache:        cache,
		metrics:      metrics,
		logger:       logger,
	}
}

// RunCycle executes one complete ingestion cycle.
func (ing *Ingestor) RunCycle(ctx context.Context) error {
	start := time.Now()

	// 1. Load active coins.
	coins, err := ing.coinRepo.GetActiveCoins(ctx)
	if err != nil {
		ing.recordFailure(ctx, start, ing.providerName(), err)
		return fmt.Errorf("load active coins: %w", err)
	}
	if len(coins) == 0 {
		ing.logger.Warn("no active coins found, skipping cycle")
		return nil
	}

	// Build provider symbol list and lookup map.
	providerSymbols := make([]string, 0, len(coins))
	symbolToCoin := make(map[string]market.Coin, len(coins))
	for _, c := range coins {
		providerSymbols = append(providerSymbols, c.ProviderSymbol)
		symbolToCoin[c.ProviderSymbol] = c
	}

	// 2. Fetch market data from provider (with fallback if configured).
	providerStart := time.Now()
	var dataList []market.MarketData
	var fetchErr error
	var usedProvider string

	if ing.orchestrator != nil {
		dataList, fetchErr = ing.orchestrator.FetchMarketData(ctx, providerSymbols)
		usedProvider = ing.orchestrator.ActiveProvider()
	} else {
		dataList, fetchErr = ing.provider.FetchMarketData(ctx, providerSymbols)
		usedProvider = ing.provider.Name()
	}
	providerDuration := time.Since(providerStart)

	if fetchErr != nil {
		ing.metrics.ProviderRequestDuration.WithLabelValues(usedProvider, "error").Observe(providerDuration.Seconds())
		ing.recordFailure(ctx, start, usedProvider, fetchErr)
		return fmt.Errorf("fetch market data: %w", fetchErr)
	}
	ing.metrics.ProviderRequestDuration.WithLabelValues(usedProvider, "success").Observe(providerDuration.Seconds())

	// 3. Validate and build snapshots.
	now := time.Now().UTC()
	snapshots := make([]market.PriceSnapshot, 0, len(dataList))
	latestData := make([]market.LatestMarketData, 0, len(dataList))

	for _, md := range dataList {
		if err := md.Validate(); err != nil {
			ing.logger.Warn("validation failed, skipping",
				slog.String("symbol", md.Symbol),
				slog.String("error", err.Error()))
			continue
		}

		// Find the coin by matching provider symbol via uppercase symbol.
		coin, ok := ing.findCoin(coins, md.Symbol)
		if !ok {
			ing.logger.Warn("no matching coin for market data",
				slog.String("symbol", md.Symbol))
			continue
		}

		snapshots = append(snapshots, market.PriceSnapshot{
			CoinID:     coin.ID,
			PriceUSD:   md.PriceUSD,
			MarketCap:  md.MarketCap,
			Volume24h:  md.Volume24h,
			Change24h:  md.Change24h,
			Provider:   md.Provider,
			CapturedAt: now,
		})

		latestData = append(latestData, market.LatestMarketData{
			Symbol:     coin.Symbol,
			Name:       coin.Name,
			PriceUSD:   md.PriceUSD,
			MarketCap:  md.MarketCap,
			Volume24h:  md.Volume24h,
			Change24h:  md.Change24h,
			Provider:   md.Provider,
			CapturedAt: now,
		})
	}

	if len(snapshots) == 0 {
		ing.recordFailure(ctx, start, usedProvider, fmt.Errorf("no valid snapshots after validation"))
		return fmt.Errorf("no valid market data after validation")
	}

	// 4. Store snapshots in PostgreSQL.
	if err := ing.snapshotRepo.InsertBatch(ctx, snapshots); err != nil {
		ing.recordFailure(ctx, start, usedProvider, err)
		return fmt.Errorf("store snapshots: %w", err)
	}

	// 5. Record sync log.
	ing.recordSuccess(ctx, start, usedProvider)

	// 6. Store latest values in Redis and 7. publish events.
	for _, ld := range latestData {
		if err := ing.cache.SetLatest(ctx, ld); err != nil {
			ing.logger.Error("failed to cache latest",
				slog.String("symbol", ld.Symbol),
				slog.String("error", err.Error()))
		}
		if err := ing.cache.PublishEvent(ctx, ld); err != nil {
			ing.logger.Error("failed to publish event",
				slog.String("symbol", ld.Symbol),
				slog.String("error", err.Error()))
		}
	}

	// 8. Log completion.
	duration := time.Since(start)
	ing.metrics.IngestionDuration.Observe(duration.Seconds())
	ing.metrics.IngestionSuccess.Inc()
	ing.metrics.DataFreshness.Set(0)

	ing.logger.Info("ingestion cycle complete",
		slog.Int("items", len(snapshots)),
		slog.Duration("duration", duration))

	return nil
}

func (ing *Ingestor) findCoin(coins []market.Coin, symbol string) (market.Coin, bool) {
	for _, c := range coins {
		if c.Symbol == symbol {
			return c, true
		}
	}
	return market.Coin{}, false
}

func (ing *Ingestor) providerName() string {
	if ing.orchestrator != nil {
		return ing.orchestrator.ActiveProvider()
	}
	if ing.provider != nil {
		return ing.provider.Name()
	}
	return "unknown"
}

func (ing *Ingestor) recordSuccess(ctx context.Context, start time.Time, providerName string) {
	finished := time.Now().UTC()
	log := &market.ProviderSyncLog{
		Provider:          providerName,
		RequestDurationMs: time.Since(start).Milliseconds(),
		Status:            "success",
		StartedAt:         start.UTC(),
		FinishedAt:        &finished,
	}
	if err := ing.syncLogRepo.Insert(ctx, log); err != nil {
		ing.logger.Error("failed to record sync log", slog.String("error", err.Error()))
	}
}

func (ing *Ingestor) recordFailure(ctx context.Context, start time.Time, providerName string, cause error) {
	ing.metrics.IngestionFailure.Inc()
	finished := time.Now().UTC()
	errMsg := cause.Error()
	log := &market.ProviderSyncLog{
		Provider:          providerName,
		RequestDurationMs: time.Since(start).Milliseconds(),
		Status:            "failure",
		ErrorMessage:      &errMsg,
		StartedAt:         start.UTC(),
		FinishedAt:        &finished,
	}
	if err := ing.syncLogRepo.Insert(ctx, log); err != nil {
		ing.logger.Error("failed to record sync log", slog.String("error", err.Error()))
	}
}
