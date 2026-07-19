"use client";

import { useMarketData } from "@/hooks/useMarketData";
import { ConnectionStatus } from "@/components/ConnectionStatus";
import { MarketTable } from "@/components/MarketTable";
import { ProviderPanel } from "@/components/ProviderPanel";
import { formatDuration, getDataAgeSeconds } from "@/lib/freshness";

export default function DashboardPage() {
  const { markets, isLoading, error, connectionState, lastUpdated } =
    useMarketData();

  const dataAge = getDataAgeSeconds(lastUpdated);

  return (
    <main className="min-h-screen bg-neutral-950">
      {/* Header */}
      <header className="border-b border-neutral-800 bg-neutral-900/50 backdrop-blur-sm sticky top-0 z-10">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
            <div>
              <h1 className="text-xl font-bold text-neutral-100">
                Crypto Market Dashboard
              </h1>
              <p className="text-sm text-neutral-500 mt-0.5">
                Real-time cryptocurrency market data platform
              </p>
            </div>
            <div className="flex items-center gap-4">
              {dataAge !== null && (
                <span className="text-xs text-neutral-500">
                  Updated {formatDuration(dataAge)}
                </span>
              )}
              <ConnectionStatus state={connectionState} />
            </div>
          </div>
        </div>
      </header>

      {/* Main content */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
        <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
          {/* Market table - takes 3 columns */}
          <div className="lg:col-span-3">
            {error && (
              <div className="mb-4 p-4 bg-red-900/20 border border-red-800 rounded-lg">
                <p className="text-sm text-red-400">
                  Error loading market data: {error}
                </p>
                <p className="text-xs text-red-500 mt-1">
                  The dashboard will continue showing the most recent known
                  data.
                </p>
              </div>
            )}
            <MarketTable markets={markets} isLoading={isLoading} />
          </div>

          {/* Sidebar - takes 1 column */}
          <div className="space-y-6">
            <ProviderPanel />

            {/* Info card */}
            <div className="bg-neutral-900 rounded-lg border border-neutral-800 p-4">
              <h2 className="text-sm font-medium text-neutral-400 mb-2">
                About
              </h2>
              <p className="text-xs text-neutral-500 leading-relaxed">
                This dashboard displays real-time cryptocurrency market data
                ingested from public providers. Prices update automatically via
                Server-Sent Events without page refresh.
              </p>
            </div>
          </div>
        </div>
      </div>

      {/* Footer */}
      <footer className="border-t border-neutral-800 mt-12">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
          <p className="text-xs text-neutral-600 text-center">
            Crypto Market Data Platform — Phase 2: Realtime Delivery
          </p>
        </div>
      </footer>
    </main>
  );
}
