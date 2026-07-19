"use client";

import { useEffect, useState } from "react";
import { PaginatedResponse, ProviderStatus } from "@/types/market";
import { config } from "@/lib/config";
import { FreshnessBadge } from "./FreshnessBadge";

export function ProviderPanel() {
  const [providers, setProviders] = useState<ProviderStatus[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchStatus = async () => {
      try {
        const response = await fetch(`${config.apiBaseUrl}/providers/status`);
        if (!response.ok) throw new Error("Failed to fetch provider status");
        const data: PaginatedResponse<ProviderStatus> = await response.json();
        setProviders(data.data || []);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Unknown error");
      } finally {
        setIsLoading(false);
      }
    };

    fetchStatus();
    const interval = setInterval(fetchStatus, 60000); // Refresh every minute
    return () => clearInterval(interval);
  }, []);

  if (isLoading) {
    return (
      <div className="bg-neutral-900 rounded-lg border border-neutral-800 p-4">
        <h2 className="text-sm font-medium text-neutral-400 mb-3">
          Provider Status
        </h2>
        <div className="animate-pulse space-y-2">
          <div className="h-8 bg-neutral-800 rounded" />
        </div>
      </div>
    );
  }

  if (error || providers.length === 0) {
    return (
      <div className="bg-neutral-900 rounded-lg border border-neutral-800 p-4">
        <h2 className="text-sm font-medium text-neutral-400 mb-3">
          Provider Status
        </h2>
        <p className="text-sm text-neutral-500">
          {error || "No provider data available"}
        </p>
      </div>
    );
  }

  return (
    <div className="bg-neutral-900 rounded-lg border border-neutral-800 p-4">
      <h2 className="text-sm font-medium text-neutral-400 mb-3">
        Provider Status
      </h2>
      <div className="space-y-3">
        {providers.map((provider) => (
          <div
            key={provider.provider}
            className="flex items-center justify-between py-2 border-b border-neutral-800/50 last:border-0"
          >
            <div className="flex items-center gap-3">
              <span
                className={`w-2 h-2 rounded-full ${
                  provider.last_status === "success"
                    ? "bg-green-500"
                    : "bg-red-500"
                }`}
              />
              <div>
                <p className="text-sm font-medium text-neutral-200 capitalize">
                  {provider.provider}
                </p>
                <p className="text-xs text-neutral-500">
                  {provider.last_duration_ms !== null
                    ? `${provider.last_duration_ms}ms`
                    : "—"}
                  {provider.recent_failures > 0 && (
                    <span className="text-red-400 ml-2">
                      {provider.recent_failures} failures (24h)
                    </span>
                  )}
                </p>
              </div>
            </div>
            <FreshnessBadge timestamp={provider.last_sync_at} showAge />
          </div>
        ))}
      </div>
    </div>
  );
}
