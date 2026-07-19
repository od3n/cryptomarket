"use client";

import { useEffect, useState } from "react";

type PlatformStatus = "healthy" | "degraded" | "stale" | "unavailable";

interface OperationsStatus {
  status: PlatformStatus;
  ingestion?: {
    active: boolean;
    active_provider: string;
    degraded: boolean;
  };
  freshness?: {
    overall_state: string;
    stale: number;
  };
}

const statusConfig: Record<PlatformStatus, { label: string; className: string; show: boolean }> = {
  healthy: { label: "", className: "", show: false },
  degraded: {
    label: "Degraded — operating on fallback provider",
    className: "bg-yellow-900/30 border-yellow-700 text-yellow-400",
    show: true,
  },
  stale: {
    label: "Stale data — market data may be outdated",
    className: "bg-orange-900/30 border-orange-700 text-orange-400",
    show: true,
  },
  unavailable: {
    label: "Service unavailable — data ingestion interrupted",
    className: "bg-red-900/30 border-red-700 text-red-400",
    show: true,
  },
};

export function StatusBanner() {
  const [status, setStatus] = useState<PlatformStatus>("healthy");
  const [provider, setProvider] = useState<string>("");

  useEffect(() => {
    let interval: NodeJS.Timeout;

    const fetchStatus = async () => {
      try {
        const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
        const res = await fetch(`${apiUrl}/operations/status`);
        if (res.ok) {
          const data: OperationsStatus = await res.json();
          setStatus(data.status);
          if (data.ingestion?.active_provider) {
            setProvider(data.ingestion.active_provider);
          }
        }
      } catch {
        // Silently ignore — banner stays at last known state
      }
    };

    fetchStatus();
    interval = setInterval(fetchStatus, 30000);
    return () => clearInterval(interval);
  }, []);

  const config = statusConfig[status];
  if (!config.show) return null;

  return (
    <div className={`border-b px-4 py-2 text-center text-sm font-medium ${config.className}`}>
      {config.label}
      {provider && status === "degraded" && (
        <span className="ml-2 opacity-75">({provider})</span>
      )}
    </div>
  );
}
