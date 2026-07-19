// Environment-based configuration for the frontend application.
// No hardcoded localhost URLs in application logic.

export const config = {
  // API endpoints (proxied through Next.js rewrites)
  apiBaseUrl: "/api",
  realtimeUrl: "/events/markets",

  // Data freshness thresholds (in seconds)
  freshThreshold: parseInt(
    process.env.NEXT_PUBLIC_FRESH_THRESHOLD || "120",
    10
  ),
  staleThreshold: parseInt(
    process.env.NEXT_PUBLIC_STALE_THRESHOLD || "300",
    10
  ),

  // Application environment
  appEnv: process.env.NEXT_PUBLIC_APP_ENV || "development",

  // Analytics (disabled by default)
  analyticsEnabled: process.env.NEXT_PUBLIC_ANALYTICS_ENABLED === "true",
} as const;

export type Config = typeof config;
