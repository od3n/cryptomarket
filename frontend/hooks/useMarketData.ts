"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import {
  ConnectionState,
  MarketData,
  PaginatedResponse,
  PriceEvent,
} from "@/types/market";
import { config } from "@/lib/config";
import { useMarketStream } from "./useMarketStream";

interface UseMarketDataReturn {
  markets: MarketData[];
  isLoading: boolean;
  error: string | null;
  connectionState: ConnectionState;
  lastUpdated: string | null;
}

/**
 * Hook that manages market data with initial snapshot fetch and live updates.
 *
 * Strategy:
 * 1. Open SSE stream
 * 2. Buffer incoming events briefly
 * 3. Fetch initial snapshot from REST API
 * 4. Merge buffered events (reject duplicates/older by published_at)
 * 5. Continue live processing
 */
export function useMarketData(): UseMarketDataReturn {
  const [markets, setMarkets] = useState<MarketData[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [lastUpdated, setLastUpdated] = useState<string | null>(null);

  // Track latest published_at per symbol for dedup/ordering
  const latestTimestamps = useRef<Map<string, string>>(new Map());
  // Buffer for events received before initial load completes
  const eventBuffer = useRef<PriceEvent[]>([]);
  const isInitialized = useRef(false);

  // Merge a price event into the markets state
  const mergeEvent = useCallback((event: PriceEvent) => {
    const existingTimestamp = latestTimestamps.current.get(event.symbol);

    // Reject older or duplicate events
    if (existingTimestamp && event.published_at <= existingTimestamp) {
      return;
    }

    latestTimestamps.current.set(event.symbol, event.published_at);

    setMarkets((prev) => {
      const index = prev.findIndex((m) => m.symbol === event.symbol);
      const updatedMarket: MarketData = {
        symbol: event.symbol,
        name: index >= 0 ? prev[index].name : event.symbol,
        price_usd: event.price_usd,
        market_cap: event.market_cap,
        volume_24h: event.volume_24h,
        change_24h: event.change_24h,
        provider: event.provider,
        captured_at: event.observed_at || event.published_at,
      };

      if (index >= 0) {
        const updated = [...prev];
        updated[index] = updatedMarket;
        return updated;
      }
      return [...prev, updatedMarket];
    });

    setLastUpdated(event.published_at);
  }, []);

  // Handle incoming SSE events
  const handleEvent = useCallback(
    (event: PriceEvent) => {
      if (!isInitialized.current) {
        // Buffer events until initial load completes
        eventBuffer.current.push(event);
        return;
      }
      mergeEvent(event);
    },
    [mergeEvent]
  );

  const { connectionState } = useMarketStream({
    onEvent: handleEvent,
    onError: (err) => {
      console.error("Stream error:", err);
    },
  });

  // Fetch initial snapshot
  const fetchMarkets = useCallback(async () => {
    try {
      setIsLoading(true);
      setError(null);

      const response = await fetch(`${config.apiBaseUrl}/markets`);
      if (!response.ok) {
        throw new Error(`API error: ${response.status}`);
      }

      const data: PaginatedResponse<MarketData> = await response.json();

      setMarkets(data.data || []);

      // Initialize timestamps from snapshot
      for (const market of data.data || []) {
        latestTimestamps.current.set(market.symbol, market.captured_at);
      }

      // Mark as initialized and process buffered events
      isInitialized.current = true;
      const buffered = eventBuffer.current;
      eventBuffer.current = [];
      for (const event of buffered) {
        mergeEvent(event);
      }

      if (data.data?.length > 0) {
        setLastUpdated(
          data.data.reduce(
            (latest, m) => (m.captured_at > latest ? m.captured_at : latest),
            data.data[0].captured_at
          )
        );
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to fetch markets");
      isInitialized.current = true;
    } finally {
      setIsLoading(false);
    }
  }, [mergeEvent]);

  useEffect(() => {
    fetchMarkets();
  }, [fetchMarkets]);

  // Periodic REST fallback when connection is degraded
  useEffect(() => {
    if (connectionState !== "degraded") return;

    const interval = setInterval(() => {
      fetchMarkets();
    }, 30000); // Poll every 30s in degraded mode

    return () => clearInterval(interval);
  }, [connectionState, fetchMarkets]);

  return {
    markets,
    isLoading,
    error,
    connectionState,
    lastUpdated,
  };
}
