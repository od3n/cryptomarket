#!/usr/bin/env python3
"""
Price Reconciliation Tool

Compares market data between providers to detect price discrepancies.
Part of the SRE toolkit for the Crypto Market Data Platform.

Usage:
    python reconcile_prices.py --symbols BTC,ETH,SOL --threshold-percent 2.0 --format json
    python reconcile_prices.py --api-url http://localhost:8080 --symbols BTC,ETH
"""

import argparse
import json
import sys
from dataclasses import asdict, dataclass
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen


@dataclass
class PriceComparison:
    """Result of comparing prices between two providers."""
    symbol: str
    provider_a: str
    provider_b: str
    price_a: float
    price_b: float
    absolute_diff: float
    percent_diff: float
    exceeds_threshold: bool


@dataclass
class ReconciliationResult:
    """Overall reconciliation result."""
    symbols_checked: int
    discrepancies_found: int
    threshold_percent: float
    comparisons: list
    status: str  # "ok", "warning", "error"


def fetch_from_api(api_url: str, symbol: str) -> dict | None:
    """Fetch market data for a symbol from the platform API."""
    url = f"{api_url.rstrip('/')}/coins/{symbol.upper()}"
    try:
        req = Request(url, headers={"Accept": "application/json"})
        with urlopen(req, timeout=10) as response:
            return json.loads(response.read().decode())
    except (URLError, HTTPError) as e:
        print(f"Warning: Failed to fetch {symbol} from API: {e}", file=sys.stderr)
        return None


def fetch_coingecko_price(symbol: str) -> float | None:
    """Fetch price from CoinGecko API (for direct comparison)."""
    # Map common symbols to CoinGecko IDs
    symbol_map = {
        "BTC": "bitcoin",
        "ETH": "ethereum",
        "SOL": "solana",
        "ADA": "cardano",
        "DOT": "polkadot",
        "AVAX": "avalanche-2",
        "MATIC": "matic-network",
        "LINK": "chainlink",
    }
    
    cg_id = symbol_map.get(symbol.upper(), symbol.lower())
    url = f"https://api.coingecko.com/api/v3/simple/price?ids={cg_id}&vs_currencies=usd"
    
    try:
        req = Request(url, headers={"Accept": "application/json"})
        with urlopen(req, timeout=10) as response:
            data = json.loads(response.read().decode())
            return data.get(cg_id, {}).get("usd")
    except (URLError, HTTPError) as e:
        print(f"Warning: Failed to fetch {symbol} from CoinGecko: {e}", file=sys.stderr)
        return None


def fetch_coincap_price(symbol: str) -> float | None:
    """Fetch price from CoinCap API (for direct comparison)."""
    # Map common symbols to CoinCap IDs
    symbol_map = {
        "BTC": "bitcoin",
        "ETH": "ethereum",
        "SOL": "solana",
        "ADA": "cardano",
        "DOT": "polkadot",
        "AVAX": "avalanche",
        "MATIC": "polygon",
        "LINK": "chainlink",
    }
    
    cc_id = symbol_map.get(symbol.upper(), symbol.lower())
    url = f"https://api.coincap.io/v2/assets/{cc_id}"
    
    try:
        req = Request(url, headers={"Accept": "application/json"})
        with urlopen(req, timeout=10) as response:
            data = json.loads(response.read().decode())
            price_str = data.get("data", {}).get("priceUsd")
            return float(price_str) if price_str else None
    except (URLError, HTTPError, ValueError) as e:
        print(f"Warning: Failed to fetch {symbol} from CoinCap: {e}", file=sys.stderr)
        return None


def compare_prices(
    symbol: str,
    price_a: float,
    price_b: float,
    provider_a: str,
    provider_b: str,
    threshold_percent: float
) -> PriceComparison:
    """Compare two prices and calculate differences."""
    absolute_diff = abs(price_a - price_b)
    avg_price = (price_a + price_b) / 2
    percent_diff = (absolute_diff / avg_price) * 100 if avg_price > 0 else 0
    
    return PriceComparison(
        symbol=symbol,
        provider_a=provider_a,
        provider_b=provider_b,
        price_a=price_a,
        price_b=price_b,
        absolute_diff=round(absolute_diff, 2),
        percent_diff=round(percent_diff, 4),
        exceeds_threshold=percent_diff > threshold_percent
    )


def reconcile(
    symbols: list,
    threshold_percent: float,
    api_url: str | None = None,
    use_direct_providers: bool = False
) -> ReconciliationResult:
    """
    Reconcile prices for the given symbols.
    
    If api_url is provided, fetches from the platform API.
    If use_direct_providers is True, fetches directly from CoinGecko and CoinCap.
    """
    comparisons = []
    
    for symbol in symbols:
        symbol = symbol.strip().upper()
        if not symbol:
            continue
            
        if use_direct_providers:
            # Fetch directly from providers
            price_cg = fetch_coingecko_price(symbol)
            price_cc = fetch_coincap_price(symbol)
            
            if price_cg is not None and price_cc is not None:
                comparison = compare_prices(
                    symbol, price_cg, price_cc,
                    "coingecko", "coincap",
                    threshold_percent
                )
                comparisons.append(comparison)
            else:
                print(f"Warning: Could not fetch prices for {symbol}", file=sys.stderr)
        elif api_url:
            # Fetch from platform API
            data = fetch_from_api(api_url, symbol)
            if data and data.get("price_usd"):
                # For API mode, we just report the current state
                # In a full implementation, you'd compare stored provider data
                print(f"Info: {symbol} price from API: ${data['price_usd']} (provider: {data.get('provider', 'unknown')})", file=sys.stderr)
    
    discrepancies = sum(1 for c in comparisons if c.exceeds_threshold)
    
    if discrepancies > 0:
        status = "warning"
    elif len(comparisons) == 0:
        status = "error"
    else:
        status = "ok"
    
    return ReconciliationResult(
        symbols_checked=len(symbols),
        discrepancies_found=discrepancies,
        threshold_percent=threshold_percent,
        comparisons=[asdict(c) for c in comparisons],
        status=status
    )


def format_text(result: ReconciliationResult) -> str:
    """Format result as human-readable text."""
    lines = [
        "=" * 60,
        "Price Reconciliation Report",
        "=" * 60,
        f"Symbols checked: {result.symbols_checked}",
        f"Threshold: {result.threshold_percent}%",
        f"Discrepancies found: {result.discrepancies_found}",
        f"Status: {result.status.upper()}",
        "-" * 60,
    ]
    
    if result.comparisons:
        lines.append(f"{'Symbol':<8} {'CoinGecko':>12} {'CoinCap':>12} {'Diff %':>10} {'Status':>10}")
        lines.append("-" * 60)
        
        for c in result.comparisons:
            status = "ALERT" if c["exceeds_threshold"] else "OK"
            lines.append(
                f"{c['symbol']:<8} "
                f"${c['price_a']:>10,.2f} "
                f"${c['price_b']:>10,.2f} "
                f"{c['percent_diff']:>9.4f}% "
                f"{status:>10}"
            )
    else:
        lines.append("No comparisons available.")
    
    lines.append("=" * 60)
    return "\n".join(lines)


def main():
    parser = argparse.ArgumentParser(
        description="Compare market data prices between providers"
    )
    parser.add_argument(
        "--symbols",
        default="BTC,ETH",
        help="Comma-separated list of symbols to check (default: BTC,ETH)"
    )
    parser.add_argument(
        "--threshold-percent",
        type=float,
        default=2.0,
        help="Percentage difference threshold for alerts (default: 2.0)"
    )
    parser.add_argument(
        "--format",
        choices=["text", "json"],
        default="text",
        help="Output format (default: text)"
    )
    parser.add_argument(
        "--api-url",
        help="Platform API URL to fetch data from"
    )
    parser.add_argument(
        "--direct",
        action="store_true",
        help="Fetch directly from CoinGecko and CoinCap APIs"
    )
    
    args = parser.parse_args()
    symbols = [s.strip() for s in args.symbols.split(",") if s.strip()]
    
    if not symbols:
        print("Error: No symbols specified", file=sys.stderr)
        sys.exit(1)
    
    result = reconcile(
        symbols=symbols,
        threshold_percent=args.threshold_percent,
        api_url=args.api_url,
        use_direct_providers=args.direct
    )
    
    if args.format == "json":
        print(json.dumps(asdict(result), indent=2))
    else:
        print(format_text(result))
    
    # Exit codes: 0=ok, 1=discrepancies found, 2=error
    if result.status == "ok":
        sys.exit(0)
    elif result.status == "warning":
        sys.exit(1)
    else:
        sys.exit(2)


if __name__ == "__main__":
    main()
