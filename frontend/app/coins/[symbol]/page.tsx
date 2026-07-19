"use client";

import { use } from "react";
import Link from "next/link";
import { useCoinDetail, TimeRange } from "@/hooks/useCoinHistory";
import { PriceChart } from "@/components/PriceChart";
import { FreshnessBadge } from "@/components/FreshnessBadge";

function formatPrice(value: string): string {
  const num = parseFloat(value);
  if (isNaN(num)) return value;
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(num);
}

function formatLargeNumber(value: string): string {
  const num = parseFloat(value);
  if (isNaN(num)) return value;
  if (num >= 1e12) return `$${(num / 1e12).toFixed(2)}T`;
  if (num >= 1e9) return `$${(num / 1e9).toFixed(2)}B`;
  if (num >= 1e6) return `$${(num / 1e6).toFixed(2)}M`;
  return formatPrice(value);
}

const timeRanges: TimeRange[] = ["1h", "24h", "7d"];

export default function CoinDetailPage({
  params,
}: {
  params: Promise<{ symbol: string }>;
}) {
  const { symbol } = use(params);
  const upperSymbol = symbol.toUpperCase();
  const {
    coin,
    history,
    isLoading,
    isLoadingHistory,
    error,
    timeRange,
    setTimeRange,
  } = useCoinDetail(upperSymbol);

  if (isLoading) {
    return (
      <main className="min-h-screen bg-neutral-950 flex items-center justify-center">
        <div className="animate-pulse text-neutral-500">Loading...</div>
      </main>
    );
  }

  if (error || !coin) {
    return (
      <main className="min-h-screen bg-neutral-950 flex flex-col items-center justify-center gap-4">
        <p className="text-neutral-400">{error || "Coin not found"}</p>
        <Link
          href="/"
          className="text-blue-400 hover:text-blue-300 transition-colors"
        >
          Back to Dashboard
        </Link>
      </main>
    );
  }

  const change24h = parseFloat(coin.change_24h) || 0;
  const isPositive = change24h >= 0;

  return (
    <main className="min-h-screen bg-neutral-950">
      {/* Header */}
      <header className="border-b border-neutral-800 bg-neutral-900/50">
        <div className="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
          <div className="flex items-center justify-between">
            <Link
              href="/"
              className="text-neutral-400 hover:text-neutral-200 transition-colors text-sm"
            >
              ← Back to Dashboard
            </Link>
            <FreshnessBadge timestamp={coin.captured_at} showAge />
          </div>
        </div>
      </header>

      <div className="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
        {/* Coin header */}
        <div className="mb-6">
          <div className="flex items-baseline gap-3">
            <h1 className="text-3xl font-bold text-neutral-100">
              {coin.symbol}
            </h1>
            <span className="text-neutral-500">{coin.name}</span>
          </div>
          <div className="flex items-baseline gap-4 mt-2">
            <span className="text-4xl font-mono text-neutral-100">
              {formatPrice(coin.price_usd)}
            </span>
            <span
              className={`text-lg font-mono ${
                isPositive ? "text-green-400" : "text-red-400"
              }`}
            >
              {isPositive ? "+" : ""}
              {change24h.toFixed(2)}%
            </span>
          </div>
        </div>

        {/* Stats grid */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
          <div className="bg-neutral-900 rounded-lg border border-neutral-800 p-4">
            <p className="text-xs text-neutral-500 mb-1">Market Cap</p>
            <p className="text-lg font-mono text-neutral-200">
              {formatLargeNumber(coin.market_cap)}
            </p>
          </div>
          <div className="bg-neutral-900 rounded-lg border border-neutral-800 p-4">
            <p className="text-xs text-neutral-500 mb-1">24h Volume</p>
            <p className="text-lg font-mono text-neutral-200">
              {formatLargeNumber(coin.volume_24h)}
            </p>
          </div>
          <div className="bg-neutral-900 rounded-lg border border-neutral-800 p-4">
            <p className="text-xs text-neutral-500 mb-1">Provider</p>
            <p className="text-lg text-neutral-200 capitalize">
              {coin.provider}
            </p>
          </div>
          <div className="bg-neutral-900 rounded-lg border border-neutral-800 p-4">
            <p className="text-xs text-neutral-500 mb-1">Last Update</p>
            <p className="text-sm text-neutral-200">
              {coin.captured_at
                ? new Date(coin.captured_at).toLocaleString()
                : "Unknown"}
            </p>
          </div>
        </div>

        {/* Chart section */}
        <div className="mb-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-medium text-neutral-200">
              Price History
            </h2>
            <div className="flex gap-1 bg-neutral-800 rounded-lg p-1">
              {timeRanges.map((range) => (
                <button
                  key={range}
                  onClick={() => setTimeRange(range)}
                  className={`px-3 py-1 rounded text-sm transition-colors ${
                    timeRange === range
                      ? "bg-neutral-700 text-neutral-100"
                      : "text-neutral-400 hover:text-neutral-200"
                  }`}
                >
                  {range}
                </button>
              ))}
            </div>
          </div>
          <PriceChart
            data={history}
            symbol={coin.symbol}
            isLoading={isLoadingHistory}
          />
        </div>
      </div>
    </main>
  );
}
