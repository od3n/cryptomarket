"use client";

import { ConnectionState } from "@/types/market";

interface ConnectionStatusProps {
  state: ConnectionState;
}

const stateConfig: Record<
  ConnectionState,
  { label: string; color: string; pulse: boolean }
> = {
  connecting: { label: "Connecting", color: "bg-yellow-500", pulse: true },
  live: { label: "Live", color: "bg-green-500", pulse: false },
  reconnecting: { label: "Reconnecting", color: "bg-yellow-500", pulse: true },
  disconnected: { label: "Disconnected", color: "bg-red-500", pulse: false },
  degraded: { label: "Degraded", color: "bg-orange-500", pulse: true },
};

export function ConnectionStatus({ state }: ConnectionStatusProps) {
  const { label, color, pulse } = stateConfig[state];

  return (
    <div
      className="flex items-center gap-2 px-3 py-1.5 rounded-full bg-neutral-800 text-sm"
      role="status"
      aria-label={`Connection status: ${label}`}
    >
      <span className="relative flex h-2.5 w-2.5">
        {pulse && (
          <span
            className={`animate-ping absolute inline-flex h-full w-full rounded-full ${color} opacity-75`}
          />
        )}
        <span
          className={`relative inline-flex rounded-full h-2.5 w-2.5 ${color}`}
        />
      </span>
      <span className="text-neutral-300">{label}</span>
    </div>
  );
}
