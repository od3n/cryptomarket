"use client";

import {
  Area,
  AreaChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { PriceSnapshot } from "@/types/market";

interface PriceChartProps {
  data: PriceSnapshot[];
  symbol: string;
  isLoading: boolean;
}

interface ChartDataPoint {
  time: string;
  price: number;
  timestamp: string;
}

function formatPrice(value: number): string {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(value);
}

function formatTime(timestamp: string): string {
  const date = new Date(timestamp);
  return date.toLocaleTimeString("en-US", {
    hour: "2-digit",
    minute: "2-digit",
  });
}

function CustomTooltip({
  active,
  payload,
}: {
  active?: boolean;
  payload?: Array<{ payload: ChartDataPoint }>;
}) {
  if (!active || !payload?.length) return null;

  const data = payload[0].payload;
  return (
    <div className="bg-neutral-800 border border-neutral-700 rounded-lg px-3 py-2 shadow-lg">
      <p className="text-sm font-medium text-neutral-200">
        {formatPrice(data.price)}
      </p>
      <p className="text-xs text-neutral-500">
        {new Date(data.timestamp).toLocaleString()}
      </p>
    </div>
  );
}

export function PriceChart({ data, symbol, isLoading }: PriceChartProps) {
  if (isLoading) {
    return (
      <div className="h-64 flex items-center justify-center bg-neutral-900 rounded-lg border border-neutral-800">
        <div className="animate-pulse text-neutral-500">Loading chart...</div>
      </div>
    );
  }

  if (data.length === 0) {
    return (
      <div className="h-64 flex items-center justify-center bg-neutral-900 rounded-lg border border-neutral-800">
        <p className="text-neutral-500">No historical data available</p>
      </div>
    );
  }

  const chartData: ChartDataPoint[] = data.map((snapshot) => ({
    time: formatTime(snapshot.captured_at),
    price: parseFloat(snapshot.price_usd) || 0,
    timestamp: snapshot.captured_at,
  }));

  // Calculate price change for color
  const firstPrice = chartData[0]?.price || 0;
  const lastPrice = chartData[chartData.length - 1]?.price || 0;
  const isPositive = lastPrice >= firstPrice;
  const strokeColor = isPositive ? "#22c55e" : "#ef4444";
  const fillColor = isPositive ? "#22c55e" : "#ef4444";

  return (
    <div className="bg-neutral-900 rounded-lg border border-neutral-800 p-4">
      <div className="h-64" role="img" aria-label={`Price chart for ${symbol}`}>
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart
            data={chartData}
            margin={{ top: 5, right: 5, left: 5, bottom: 5 }}
          >
            <defs>
              <linearGradient id="priceGradient" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor={fillColor} stopOpacity={0.3} />
                <stop offset="95%" stopColor={fillColor} stopOpacity={0} />
              </linearGradient>
            </defs>
            <XAxis
              dataKey="time"
              tick={{ fill: "#737373", fontSize: 12 }}
              tickLine={false}
              axisLine={{ stroke: "#404040" }}
              minTickGap={50}
            />
            <YAxis
              tick={{ fill: "#737373", fontSize: 12 }}
              tickLine={false}
              axisLine={false}
              tickFormatter={(value) => `$${value.toLocaleString()}`}
              width={80}
              domain={["auto", "auto"]}
            />
            <Tooltip content={<CustomTooltip />} />
            <Area
              type="monotone"
              dataKey="price"
              stroke={strokeColor}
              strokeWidth={2}
              fill="url(#priceGradient)"
              dot={false}
              activeDot={{ r: 4, fill: strokeColor }}
            />
          </AreaChart>
        </ResponsiveContainer>
      </div>

      {/* Accessible summary */}
      <p className="sr-only">
        Price chart for {symbol} showing {data.length} data points. Starting
        price: {formatPrice(firstPrice)}, ending price: {formatPrice(lastPrice)}.
      </p>
    </div>
  );
}
