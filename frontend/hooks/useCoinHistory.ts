"use client";

import { useCallback, useEffect, useState } from "react";
import { MarketData, PaginatedResponse, PriceSnapshot } from "@/types/market";
import { config } from "@/lib/config";

export type TimeRange = "1h" | "24h" | "7d";

const rangeToParams: Record<TimeRange, { from: () => string; limit: number }> = {
  "1h": {
    from: () => new Date(Date.now() - 60 * 60 * 1000).toISOString(),
    limit: 100,
  },
  "24h": {
    from: () => new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
    limit: 200,
  },
  "7d": {
    from: () => new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString(),
    limit: 500,
  },
};

interface UseCoinDetailReturn {
  coin: MarketData | null;
  history: PriceSnapshot[];
  isLoading: boolean;
  isLoadingHistory: boolean;
  error: string | null;
  timeRange: TimeRange;
  setTimeRange: (range: TimeRange) => void;
}

export function useCoinDetail(symbol: string): UseCoinDetailReturn {
  const [coin, setCoin] = useState<MarketData | null>(null);
  const [history, setHistory] = useState<PriceSnapshot[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isLoadingHistory, setIsLoadingHistory] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [timeRange, setTimeRange] = useState<TimeRange>("24h");

  // Fetch current coin data
  const fetchCoin = useCallback(async () => {
    try {
      const response = await fetch(`${config.apiBaseUrl}/coins/${symbol}`);
      if (!response.ok) {
        if (response.status === 404) {
          setError("Coin not found");
          return;
        }
        throw new Error(`API error: ${response.status}`);
      }
      const data: MarketData = await response.json();
      setCoin(data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to fetch coin");
    } finally {
      setIsLoading(false);
    }
  }, [symbol]);

  // Fetch history based on time range
  const fetchHistory = useCallback(async () => {
    setIsLoadingHistory(true);
    try {
      const params = rangeToParams[timeRange];
      const from = params.from();
      const response = await fetch(
        `${config.apiBaseUrl}/coins/${symbol}/history?from=${encodeURIComponent(from)}&limit=${params.limit}`
      );
      if (!response.ok) throw new Error(`API error: ${response.status}`);
      const data: PaginatedResponse<PriceSnapshot> = await response.json();
      // Sort by captured_at ascending for chart
      const sorted = (data.data || []).sort(
        (a, b) => new Date(a.captured_at).getTime() - new Date(b.captured_at).getTime()
      );
      setHistory(sorted);
    } catch (err) {
      console.error("Failed to fetch history:", err);
      setHistory([]);
    } finally {
      setIsLoadingHistory(false);
    }
  }, [symbol, timeRange]);

  useEffect(() => {
    fetchCoin();
  }, [fetchCoin]);

  useEffect(() => {
    fetchHistory();
  }, [fetchHistory]);

  // Refresh coin data periodically
  useEffect(() => {
    const interval = setInterval(fetchCoin, 30000);
    return () => clearInterval(interval);
  }, [fetchCoin]);

  return {
    coin,
    history,
    isLoading,
    isLoadingHistory,
    error,
    timeRange,
    setTimeRange,
  };
}
