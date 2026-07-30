"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { ConnectionState, PriceEvent } from "@/types/market";
import { config } from "@/lib/config";

interface UseMarketStreamOptions {
  onEvent?: (event: PriceEvent) => void;
  onError?: (error: Error) => void;
}

interface UseMarketStreamReturn {
  connectionState: ConnectionState;
  lastEvent: PriceEvent | null;
  lastEventId: string | null;
}

// Reconnection backoff settings
const INITIAL_RETRY_MS = 1000;
const MAX_RETRY_MS = 30000;
const BACKOFF_MULTIPLIER = 2;

/**
 * Hook for connecting to the SSE market stream with automatic reconnection.
 * Implements bounded exponential backoff for reconnection attempts.
 */
export function useMarketStream({
  onEvent,
  onError,
}: UseMarketStreamOptions = {}): UseMarketStreamReturn {
  const [connectionState, setConnectionState] =
    useState<ConnectionState>("connecting");
  const [lastEvent, setLastEvent] = useState<PriceEvent | null>(null);
  const [lastEventId, setLastEventId] = useState<string | null>(null);

  const eventSourceRef = useRef<EventSource | null>(null);
  const retryCountRef = useRef(0);
  const retryTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const onEventRef = useRef(onEvent);
  const onErrorRef = useRef(onError);

  // Keep refs updated with the latest callbacks after each render.
  useEffect(() => {
    onEventRef.current = onEvent;
    onErrorRef.current = onError;
  });

  const connect = useCallback(() => {
    // Clean up existing connection
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
    }

    const isReconnect = retryCountRef.current > 0;
    setConnectionState(isReconnect ? "reconnecting" : "connecting");

    // Create EventSource with Last-Event-ID support
    const url = config.realtimeUrl;
    const eventSource = new EventSource(url);
    eventSourceRef.current = eventSource;

    eventSource.onopen = () => {
      retryCountRef.current = 0;
      setConnectionState("live");
    };

    eventSource.addEventListener("market.price.updated", (e) => {
      const messageEvent = e as MessageEvent;
      try {
        const event: PriceEvent = JSON.parse(messageEvent.data);
        setLastEvent(event);
        setLastEventId(messageEvent.lastEventId || event.event_id);
        onEventRef.current?.(event);
      } catch (err) {
        console.error("Failed to parse market event:", err);
      }
    });

    eventSource.onerror = () => {
      eventSource.close();
      eventSourceRef.current = null;

      // Calculate backoff delay
      const delay = Math.min(
        INITIAL_RETRY_MS * Math.pow(BACKOFF_MULTIPLIER, retryCountRef.current),
        MAX_RETRY_MS
      );
      retryCountRef.current += 1;

      // After multiple failures, show degraded state
      if (retryCountRef.current > 3) {
        setConnectionState("degraded");
      } else {
        setConnectionState("reconnecting");
      }

      onErrorRef.current?.(new Error("SSE connection error"));

      // Schedule reconnection
      retryTimeoutRef.current = setTimeout(() => {
        connect();
      }, delay);
    };
  }, []);

  useEffect(() => {
    connect();

    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
      if (retryTimeoutRef.current) {
        clearTimeout(retryTimeoutRef.current);
      }
    };
  }, [connect]);

  return { connectionState, lastEvent, lastEventId };
}
