"""Tests for the failure injection toolkit."""

import os
import sys
import unittest
from unittest.mock import patch

# Add parent directory to path
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from inject_failures import (
    SCENARIOS,
    check_guard,
    cleanup,
    inject,
    list_scenarios,
)


class TestListScenarios(unittest.TestCase):
    """Test scenario listing."""

    def test_list_returns_all_scenarios(self):
        result = list_scenarios()
        self.assertIn("scenarios", result)
        self.assertEqual(len(result["scenarios"]), len(SCENARIOS))

    def test_list_includes_descriptions(self):
        result = list_scenarios()
        for info in result["scenarios"].values():
            self.assertIn("description", info)
            self.assertIn("reversible", info)
            self.assertTrue(len(info["description"]) > 0)

    def test_all_scenarios_reversible(self):
        result = list_scenarios()
        for name, info in result["scenarios"].items():
            self.assertTrue(info["reversible"], f"Scenario {name} should be reversible")


class TestEnvironmentGuard(unittest.TestCase):
    """Test environment guard."""

    @patch.dict(os.environ, {"ALLOW_FAILURE_INJECTION": ""})
    def test_guard_blocks_when_disabled(self):
        with self.assertRaises(SystemExit) as ctx:
            check_guard()
        self.assertEqual(ctx.exception.code, 1)

    @patch.dict(os.environ, {"ALLOW_FAILURE_INJECTION": "false"})
    def test_guard_blocks_when_false(self):
        with self.assertRaises(SystemExit) as ctx:
            check_guard()
        self.assertEqual(ctx.exception.code, 1)

    @patch("inject_failures.ALLOW_INJECTION", True)
    def test_guard_allows_when_true(self):
        # Should not raise
        check_guard()


class TestInject(unittest.TestCase):
    """Test failure injection."""

    def test_unknown_scenario(self):
        result = inject("nonexistent_scenario")
        self.assertFalse(result["success"])
        self.assertIn("Unknown scenario", result["error"])

    @patch("inject_failures.set_mock_mode")
    @patch("inject_failures.ALLOW_INJECTION", True)
    def test_inject_provider_429(self, mock_set_mode):
        mock_set_mode.return_value = {"success": True, "output": ""}
        result = inject("provider_429")
        self.assertTrue(result["success"])
        self.assertEqual(result["scenario"], "provider_429")
        mock_set_mode.assert_called_once_with("rate_limit")

    @patch("inject_failures.set_mock_mode")
    @patch("inject_failures.ALLOW_INJECTION", True)
    def test_inject_provider_500(self, mock_set_mode):
        mock_set_mode.return_value = {"success": True, "output": ""}
        result = inject("provider_500")
        self.assertTrue(result["success"])
        mock_set_mode.assert_called_once_with("error")

    @patch("inject_failures.set_mock_mode")
    @patch("inject_failures.ALLOW_INJECTION", True)
    def test_inject_with_duration(self, mock_set_mode):
        mock_set_mode.return_value = {"success": True, "output": ""}
        result = inject("provider_timeout", duration=60)
        self.assertTrue(result["success"])
        self.assertIn("auto_cleanup_in", result)
        self.assertEqual(result["auto_cleanup_in"], "60s")

    @patch("inject_failures.docker_stop")
    @patch("inject_failures.ALLOW_INJECTION", True)
    def test_inject_redis_failure(self, mock_stop):
        mock_stop.return_value = {"success": True, "output": ""}
        result = inject("redis_failure")
        self.assertTrue(result["success"])
        mock_stop.assert_called_once_with("redis")


class TestCleanup(unittest.TestCase):
    """Test failure cleanup."""

    def test_cleanup_unknown_scenario(self):
        result = cleanup("nonexistent_scenario")
        self.assertFalse(result["success"])

    @patch("inject_failures.set_mock_mode")
    def test_cleanup_provider_mode(self, mock_set_mode):
        mock_set_mode.return_value = {"success": True, "output": ""}
        result = cleanup("provider_429")
        self.assertTrue(result["success"])
        self.assertEqual(result["action"], "cleanup")
        mock_set_mode.assert_called_once_with("success")

    @patch("inject_failures.docker_start")
    def test_cleanup_redis_failure(self, mock_start):
        mock_start.return_value = {"success": True, "output": ""}
        result = cleanup("redis_failure")
        self.assertTrue(result["success"])
        mock_start.assert_called_once_with("redis")

    @patch("inject_failures.docker_start")
    def test_cleanup_stale_data(self, mock_start):
        mock_start.return_value = {"success": True, "output": ""}
        result = cleanup("stale_data")
        self.assertTrue(result["success"])
        mock_start.assert_called_once_with("ingestor")


class TestScenarioConfig(unittest.TestCase):
    """Test scenario configuration integrity."""

    def test_all_scenarios_have_description(self):
        for name, config in SCENARIOS.items():
            self.assertIn("description", config, f"{name} missing description")
            self.assertIn("mode", config, f"{name} missing mode")
            self.assertIn("reversible", config, f"{name} missing reversible")

    def test_docker_scenarios_have_target(self):
        for name, config in SCENARIOS.items():
            if config["mode"] in ("docker_stop", "docker_restart_env"):
                self.assertIn("target", config, f"{name} missing target")


if __name__ == "__main__":
    unittest.main()
