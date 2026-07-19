// Shared freshness calculation utility.
// Determines data freshness state based on configurable thresholds.
//
// Assumptions:
// - Timestamps are ISO 8601 / RFC 3339 format (UTC)
// - "fresh" means data is recent enough to be considered current
// - "delayed" means data is slightly stale but still usable
// - "stale" means data is too old to be reliable
// - "unknown" means no timestamp is available

import { FreshnessState } from "@/types/market";
import { config } from "./config";

/**
 * Calculate the freshness state of data based on its timestamp.
 *
 * @param timestamp - ISO 8601 timestamp string or null/undefined
 * @param freshThreshold - Seconds within which data is considered fresh (default from config)
 * @param staleThreshold - Seconds after which data is considered stale (default from config)
 * @returns FreshnessState: "fresh" | "delayed" | "stale" | "unknown"
 */
export function calculateFreshness(
  timestamp: string | null | undefined,
  freshThreshold: number = config.freshThreshold,
  staleThreshold: number = config.staleThreshold
): FreshnessState {
  if (!timestamp) {
    return "unknown";
  }

  const dataTime = new Date(timestamp).getTime();
  if (isNaN(dataTime)) {
    return "unknown";
  }

  const ageSeconds = (Date.now() - dataTime) / 1000;

  if (ageSeconds <= freshThreshold) {
    return "fresh";
  }
  if (ageSeconds <= staleThreshold) {
    return "delayed";
  }
  return "stale";
}

/**
 * Get the age of data in seconds.
 */
export function getDataAgeSeconds(timestamp: string | null | undefined): number | null {
  if (!timestamp) {
    return null;
  }

  const dataTime = new Date(timestamp).getTime();
  if (isNaN(dataTime)) {
    return null;
  }

  return Math.floor((Date.now() - dataTime) / 1000);
}

/**
 * Format seconds into a human-readable duration.
 */
export function formatDuration(seconds: number): string {
  if (seconds < 60) {
    return `${seconds}s ago`;
  }
  if (seconds < 3600) {
    const minutes = Math.floor(seconds / 60);
    return `${minutes}m ago`;
  }
  const hours = Math.floor(seconds / 3600);
  return `${hours}h ago`;
}
