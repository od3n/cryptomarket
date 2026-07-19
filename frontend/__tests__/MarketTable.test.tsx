import { render, screen, fireEvent } from "@testing-library/react";
import { MarketTable } from "@/components/MarketTable";
import { MarketData } from "@/types/market";

// Mock next/link
jest.mock("next/link", () => {
  return ({ children, href }: { children: React.ReactNode; href: string }) => (
    <a href={href}>{children}</a>
  );
});

// Mock FreshnessBadge to simplify tests
jest.mock("@/components/FreshnessBadge", () => ({
  FreshnessBadge: ({ timestamp }: { timestamp: string }) => (
    <span data-testid="freshness-badge">{timestamp ? "Fresh" : "Unknown"}</span>
  ),
}));

const mockMarkets: MarketData[] = [
  {
    symbol: "BTC",
    name: "Bitcoin",
    price_usd: "65000.50",
    market_cap: "1300000000000",
    volume_24h: "50000000000",
    change_24h: "2.5",
    provider: "coingecko",
    captured_at: "2026-07-20T08:15:00Z",
  },
  {
    symbol: "ETH",
    name: "Ethereum",
    price_usd: "3500.00",
    market_cap: "420000000000",
    volume_24h: "20000000000",
    change_24h: "-1.2",
    provider: "coingecko",
    captured_at: "2026-07-20T08:15:00Z",
  },
  {
    symbol: "SOL",
    name: "Solana",
    price_usd: "150.00",
    market_cap: "65000000000",
    volume_24h: "3000000000",
    change_24h: "5.0",
    provider: "coingecko",
    captured_at: "2026-07-20T08:15:00Z",
  },
];

describe("MarketTable", () => {
  it("renders loading state", () => {
    const { container } = render(<MarketTable markets={[]} isLoading={true} />);
    expect(container.querySelector(".animate-pulse")).toBeInTheDocument();
  });

  it("renders market data", () => {
    render(<MarketTable markets={mockMarkets} isLoading={false} />);
    expect(screen.getByText("BTC")).toBeInTheDocument();
    expect(screen.getByText("ETH")).toBeInTheDocument();
    expect(screen.getByText("SOL")).toBeInTheDocument();
  });

  it("renders empty state when no markets", () => {
    render(<MarketTable markets={[]} isLoading={false} />);
    expect(screen.getByText("No market data available")).toBeInTheDocument();
  });

  it("filters by symbol", () => {
    render(<MarketTable markets={mockMarkets} isLoading={false} />);
    const input = screen.getByPlaceholderText("Filter by symbol or name...");
    fireEvent.change(input, { target: { value: "btc" } });

    expect(screen.getByText("BTC")).toBeInTheDocument();
    expect(screen.queryByText("ETH")).not.toBeInTheDocument();
    expect(screen.queryByText("SOL")).not.toBeInTheDocument();
  });

  it("filters by name", () => {
    render(<MarketTable markets={mockMarkets} isLoading={false} />);
    const input = screen.getByPlaceholderText("Filter by symbol or name...");
    fireEvent.change(input, { target: { value: "ethereum" } });

    expect(screen.getByText("ETH")).toBeInTheDocument();
    expect(screen.queryByText("BTC")).not.toBeInTheDocument();
  });

  it("shows no matching assets message", () => {
    render(<MarketTable markets={mockMarkets} isLoading={false} />);
    const input = screen.getByPlaceholderText("Filter by symbol or name...");
    fireEvent.change(input, { target: { value: "xyz" } });

    expect(screen.getByText("No matching assets found")).toBeInTheDocument();
  });

  it("clears filter when clear button clicked", () => {
    render(<MarketTable markets={mockMarkets} isLoading={false} />);
    const input = screen.getByPlaceholderText("Filter by symbol or name...");
    fireEvent.change(input, { target: { value: "btc" } });

    const clearButton = screen.getByLabelText("Clear filter");
    fireEvent.click(clearButton);

    expect(screen.getByText("BTC")).toBeInTheDocument();
    expect(screen.getByText("ETH")).toBeInTheDocument();
    expect(screen.getByText("SOL")).toBeInTheDocument();
  });

  it("sorts by symbol when header clicked", () => {
    render(<MarketTable markets={mockMarkets} isLoading={false} />);
    const symbolSort = screen.getByLabelText("Sort by Asset");
    // First click sorts descending (SOL, ETH, BTC)
    fireEvent.click(symbolSort);
    // Second click sorts ascending (BTC, ETH, SOL)
    fireEvent.click(symbolSort);

    const rows = screen.getAllByRole("row").slice(1); // skip header
    expect(rows[0]).toHaveTextContent("BTC");
    expect(rows[1]).toHaveTextContent("ETH");
    expect(rows[2]).toHaveTextContent("SOL");
  });

  it("displays correct asset count", () => {
    render(<MarketTable markets={mockMarkets} isLoading={false} />);
    expect(screen.getByText("Showing 3 of 3 assets")).toBeInTheDocument();
  });

  it("shows filtered count", () => {
    render(<MarketTable markets={mockMarkets} isLoading={false} />);
    const input = screen.getByPlaceholderText("Filter by symbol or name...");
    fireEvent.change(input, { target: { value: "btc" } });

    expect(screen.getByText("Showing 1 of 3 assets")).toBeInTheDocument();
  });
});
