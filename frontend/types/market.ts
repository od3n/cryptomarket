// Market data types matching the backend event contract

export interface PriceEvent {
  event_id: string;
  event_type: "market.price.updated";
  symbol: string;
  price_usd: string;
  market_cap: string;
  volume_24h: string;
  change_24h: string;
  provider: string;
  observed_at: string;
  published_at: string;
}

export interface MarketData {
  symbol: string;
  name: string;
  price_usd: string;
  market_cap: string;
  volume_24h: string;
  change_24h: string;
  provider: string;
  captured_at: string;
}

export interface Coin {
  id: number;
  symbol: string;
  name: string;
  provider_symbol: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface PriceSnapshot {
  id: number;
  coin_id: number;
  price_usd: string;
  market_cap: string;
  volume_24h: string;
  change_24h: string;
  provider: string;
  captured_at: string;
}

export interface ProviderStatus {
  provider: string;
  last_sync_at: string | null;
  last_duration_ms: number | null;
  last_status: string;
  recent_failures: number;
}

export interface PaginatedResponse<T> {
  data: T[];
  count: number;
}

export type FreshnessState = "fresh" | "delayed" | "stale" | "unknown";

export type ConnectionState =
  | "connecting"
  | "live"
  | "reconnecting"
  | "disconnected"
  | "degraded";

export type SortField = "price_usd" | "market_cap" | "change_24h" | "symbol";
export type SortDirection = "asc" | "desc";
