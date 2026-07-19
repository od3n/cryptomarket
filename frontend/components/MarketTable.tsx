"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import { MarketData, SortDirection, SortField } from "@/types/market";
import { FreshnessBadge } from "./FreshnessBadge";

interface MarketTableProps {
  markets: MarketData[];
  isLoading: boolean;
}

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

function formatChange(value: string): { text: string; isPositive: boolean } {
  const num = parseFloat(value);
  if (isNaN(num)) return { text: value, isPositive: false };
  const isPositive = num >= 0;
  return {
    text: `${isPositive ? "+" : ""}${num.toFixed(2)}%`,
    isPositive,
  };
}

const columns: { field: SortField; label: string; align: string }[] = [
  { field: "symbol", label: "Asset", align: "text-left" },
  { field: "price_usd", label: "Price", align: "text-right" },
  { field: "change_24h", label: "24h Change", align: "text-right" },
  { field: "market_cap", label: "Market Cap", align: "text-right" },
];

export function MarketTable({ markets, isLoading }: MarketTableProps) {
  const [sortField, setSortField] = useState<SortField>("market_cap");
  const [sortDirection, setSortDirection] = useState<SortDirection>("desc");
  const [filter, setFilter] = useState("");

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDirection(sortDirection === "asc" ? "desc" : "asc");
    } else {
      setSortField(field);
      setSortDirection("desc");
    }
  };

  const filteredAndSorted = useMemo(() => {
    let result = [...markets];

    // Filter
    if (filter) {
      const lowerFilter = filter.toLowerCase();
      result = result.filter(
        (m) =>
          m.symbol.toLowerCase().includes(lowerFilter) ||
          m.name.toLowerCase().includes(lowerFilter)
      );
    }

    // Sort
    result.sort((a, b) => {
      let comparison = 0;
      if (sortField === "symbol") {
        comparison = a.symbol.localeCompare(b.symbol);
      } else {
        const aVal = parseFloat(a[sortField]) || 0;
        const bVal = parseFloat(b[sortField]) || 0;
        comparison = aVal - bVal;
      }
      return sortDirection === "asc" ? comparison : -comparison;
    });

    return result;
  }, [markets, filter, sortField, sortDirection]);

  if (isLoading) {
    return (
      <div className="animate-pulse space-y-4">
        {[...Array(5)].map((_, i) => (
          <div key={i} className="h-12 bg-neutral-800 rounded" />
        ))}
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Filter input */}
      <div className="relative">
        <input
          type="text"
          placeholder="Filter by symbol or name..."
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          className="w-full px-4 py-2 bg-neutral-800 border border-neutral-700 rounded-lg text-neutral-200 placeholder-neutral-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          aria-label="Filter markets"
        />
        {filter && (
          <button
            onClick={() => setFilter("")}
            className="absolute right-3 top-1/2 -translate-y-1/2 text-neutral-500 hover:text-neutral-300"
            aria-label="Clear filter"
          >
            ✕
          </button>
        )}
      </div>

      {/* Table */}
      <div className="overflow-x-auto rounded-lg border border-neutral-800">
        <table className="w-full text-sm" role="table">
          <thead>
            <tr className="border-b border-neutral-800 bg-neutral-900/50">
              {columns.map((col) => (
                <th
                  key={col.field}
                  className={`px-4 py-3 font-medium text-neutral-400 ${col.align}`}
                >
                  <button
                    onClick={() => handleSort(col.field)}
                    className="inline-flex items-center gap-1 hover:text-neutral-200 transition-colors"
                    aria-label={`Sort by ${col.label}`}
                  >
                    {col.label}
                    {sortField === col.field && (
                      <span className="text-xs">
                        {sortDirection === "asc" ? "↑" : "↓"}
                      </span>
                    )}
                  </button>
                </th>
              ))}
              <th className="px-4 py-3 font-medium text-neutral-400 text-left">
                Status
              </th>
            </tr>
          </thead>
          <tbody>
            {filteredAndSorted.length === 0 ? (
              <tr>
                <td
                  colSpan={5}
                  className="px-4 py-8 text-center text-neutral-500"
                >
                  {markets.length === 0
                    ? "No market data available"
                    : "No matching assets found"}
                </td>
              </tr>
            ) : (
              filteredAndSorted.map((market) => {
                const change = formatChange(market.change_24h);
                return (
                  <tr
                    key={market.symbol}
                    className="border-b border-neutral-800/50 hover:bg-neutral-800/30 transition-colors"
                  >
                    <td className="px-4 py-3">
                      <Link
                        href={`/coins/${market.symbol}`}
                        className="flex items-center gap-2 hover:text-blue-400 transition-colors"
                      >
                        <span className="font-semibold text-neutral-100">
                          {market.symbol}
                        </span>
                        <span className="text-neutral-500 text-xs hidden sm:inline">
                          {market.name}
                        </span>
                      </Link>
                    </td>
                    <td className="px-4 py-3 text-right font-mono text-neutral-200">
                      {formatPrice(market.price_usd)}
                    </td>
                    <td
                      className={`px-4 py-3 text-right font-mono ${
                        change.isPositive ? "text-green-400" : "text-red-400"
                      }`}
                    >
                      {change.text}
                    </td>
                    <td className="px-4 py-3 text-right font-mono text-neutral-300">
                      {formatLargeNumber(market.market_cap)}
                    </td>
                    <td className="px-4 py-3">
                      <FreshnessBadge timestamp={market.captured_at} />
                    </td>
                  </tr>
                );
              })
            )}
          </tbody>
        </table>
      </div>

      {/* Volume row (shown below table on mobile) */}
      <p className="text-xs text-neutral-500">
        Showing {filteredAndSorted.length} of {markets.length} assets
      </p>
    </div>
  );
}
