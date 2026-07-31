"""Tests for the price reconciliation tool."""

import os
import sys
import unittest

# Add parent directory to path for imports
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from reconcile_prices import (
    ReconciliationResult,
    compare_prices,
    format_text,
)


class TestComparePrices(unittest.TestCase):
    """Tests for the compare_prices function."""

    def test_identical_prices(self):
        """Identical prices should have zero difference."""
        result = compare_prices("BTC", 50000.0, 50000.0, "coingecko", "coincap", 2.0)
        
        self.assertEqual(result.symbol, "BTC")
        self.assertEqual(result.absolute_diff, 0.0)
        self.assertEqual(result.percent_diff, 0.0)
        self.assertFalse(result.exceeds_threshold)

    def test_small_difference(self):
        """Small difference below threshold should not exceed."""
        result = compare_prices("BTC", 50000.0, 50500.0, "coingecko", "coincap", 2.0)
        
        self.assertLess(result.percent_diff, 2.0)
        self.assertFalse(result.exceeds_threshold)

    def test_large_difference(self):
        """Large difference above threshold should exceed."""
        result = compare_prices("BTC", 50000.0, 55000.0, "coingecko", "coincap", 2.0)
        
        self.assertGreater(result.percent_diff, 2.0)
        self.assertTrue(result.exceeds_threshold)

    def test_absolute_difference_calculation(self):
        """Absolute difference should be correctly calculated."""
        result = compare_prices("ETH", 3000.0, 3100.0, "coingecko", "coincap", 5.0)
        
        self.assertEqual(result.absolute_diff, 100.0)

    def test_percent_difference_calculation(self):
        """Percent difference should use average as denominator."""
        # Average of 100 and 110 is 105
        # Diff is 10, so percent is 10/105 * 100 = 9.52%
        result = compare_prices("TEST", 100.0, 110.0, "a", "b", 5.0)
        
        expected_percent = (10.0 / 105.0) * 100
        self.assertAlmostEqual(result.percent_diff, round(expected_percent, 4), places=3)


class TestReconciliationResult(unittest.TestCase):
    """Tests for ReconciliationResult formatting."""

    def test_format_text_ok_status(self):
        """Text format should display OK status correctly."""
        result = ReconciliationResult(
            symbols_checked=2,
            discrepancies_found=0,
            threshold_percent=2.0,
            comparisons=[],
            status="ok"
        )
        
        text = format_text(result)
        self.assertIn("OK", text)
        self.assertIn("Symbols checked: 2", text)

    def test_format_text_with_comparisons(self):
        """Text format should include comparison data."""
        result = ReconciliationResult(
            symbols_checked=1,
            discrepancies_found=1,
            threshold_percent=2.0,
            comparisons=[{
                "symbol": "BTC",
                "provider_a": "coingecko",
                "provider_b": "coincap",
                "price_a": 50000.0,
                "price_b": 55000.0,
                "absolute_diff": 5000.0,
                "percent_diff": 9.5238,
                "exceeds_threshold": True
            }],
            status="warning"
        )
        
        text = format_text(result)
        self.assertIn("BTC", text)
        self.assertIn("WARNING", text)
        self.assertIn("ALERT", text)


class TestEdgeCases(unittest.TestCase):
    """Tests for edge cases."""

    def test_zero_prices(self):
        """Zero prices should be handled gracefully."""
        result = compare_prices("TEST", 0.0, 0.0, "a", "b", 2.0)
        
        self.assertEqual(result.percent_diff, 0.0)

    def test_one_zero_price(self):
        """One zero price should result in 100% difference conceptually."""
        result = compare_prices("TEST", 100.0, 0.0, "a", "b", 2.0)
        
        # Average is 50, diff is 100, so 200%
        self.assertTrue(result.exceeds_threshold)


if __name__ == "__main__":
    unittest.main()
