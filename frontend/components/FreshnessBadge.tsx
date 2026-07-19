"use client";

import { FreshnessState } from "@/types/market";
import { calculateFreshness, getDataAgeSeconds, formatDuration } from "@/lib/freshness";

interface FreshnessBadgeProps {
  timestamp: string | null | undefined;
  showAge?: boolean;
}

const stateConfig: Record<FreshnessState, { label: string; className: string }> = {
  fresh: { label: "Fresh", className: "bg-green-900/50 text-green-400 border-green-700" },
  delayed: { label: "Delayed", className: "bg-yellow-900/50 text-yellow-400 border-yellow-700" },
  stale: { label: "Stale", className: "bg-red-900/50 text-red-400 border-red-700" },
  unknown: { label: "Unknown", className: "bg-neutral-800 text-neutral-400 border-neutral-600" },
};

export function FreshnessBadge({ timestamp, showAge = false }: FreshnessBadgeProps) {
  const state = calculateFreshness(timestamp);
  const { label, className } = stateConfig[state];
  const ageSeconds = getDataAgeSeconds(timestamp);

  return (
    <span
      className={`inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium border ${className}`}
      title={timestamp ? `Last update: ${new Date(timestamp).toLocaleString()}` : "No timestamp"}
    >
      {label}
      {showAge && ageSeconds !== null && (
        <span className="opacity-75">({formatDuration(ageSeconds)})</span>
      )}
    </span>
  );
}
