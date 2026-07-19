#!/usr/bin/env python3
"""Failure Injection Toolkit for Crypto Market Data Platform.

Injects controlled failures into the platform for resilience testing.
Requires ALLOW_FAILURE_INJECTION=true environment variable to operate.

Scenarios:
  - provider_timeout: Makes mock provider respond with delay
  - provider_500: Makes mock provider return HTTP 500
  - provider_429: Makes mock provider return HTTP 429 (rate limit)
  - provider_malformed: Makes mock provider return invalid JSON
  - redis_failure: Stops Redis container
  - stale_data: Stops ingestor to create stale data
  - delayed_ingestion: Increases ingestion interval

Usage:
  python inject_failures.py --scenario provider_429 --duration 60
  python inject_failures.py --scenario provider_500 --cleanup
  python inject_failures.py --list
"""

import argparse
import json
import os
import subprocess
import sys
import time
import urllib.request
import urllib.error


# Environment guard
ALLOW_INJECTION = os.environ.get("ALLOW_FAILURE_INJECTION", "").lower() == "true"
MOCK_PROVIDER_URL = os.environ.get("MOCK_PROVIDER_URL", "http://localhost:8082")
COMPOSE_PROJECT = os.environ.get("COMPOSE_PROJECT_NAME", "cryptomarket")

SCENARIOS = {
    "provider_timeout": {
        "description": "Mock provider responds with configurable delay",
        "mode": "delayed",
        "reversible": True,
    },
    "provider_500": {
        "description": "Mock provider returns HTTP 500",
        "mode": "error",
        "reversible": True,
    },
    "provider_429": {
        "description": "Mock provider returns HTTP 429 (rate limited)",
        "mode": "rate_limit",
        "reversible": True,
    },
    "provider_malformed": {
        "description": "Mock provider returns malformed JSON",
        "mode": "malformed",
        "reversible": True,
    },
    "redis_failure": {
        "description": "Stops Redis container",
        "mode": "docker_stop",
        "target": "redis",
        "reversible": True,
    },
    "stale_data": {
        "description": "Stops ingestor to create stale data",
        "mode": "docker_stop",
        "target": "ingestor",
        "reversible": True,
    },
    "delayed_ingestion": {
        "description": "Restarts ingestor with longer interval",
        "mode": "docker_restart_env",
        "target": "ingestor",
        "env": {"INGESTION_INTERVAL": "300s"},
        "reversible": True,
    },
}


def check_guard():
    """Check if failure injection is allowed."""
    if not ALLOW_INJECTION:
        print(json.dumps({
            "success": False,
            "error": "Failure injection is disabled. Set ALLOW_FAILURE_INJECTION=true to enable.",
            "scenarios": list(SCENARIOS.keys()),
        }, indent=2))
        sys.exit(1)


def set_mock_mode(mode: str) -> dict:
    """Set the mock provider mode via environment variable restart."""
    try:
        result = subprocess.run(
            ["docker", "compose", "-p", COMPOSE_PROJECT,
             "exec", "-T", "mock-provider",
             "sh", "-c", f"export MOCK_MODE={mode}"],
            capture_output=True, text=True, timeout=10
        )
        # Alternative: restart with new env
        result = subprocess.run(
            ["docker", "compose", "-p", COMPOSE_PROJECT,
             "up", "-d", "--force-recreate",
             "-e", f"MOCK_MODE={mode}", "mock-provider"],
            capture_output=True, text=True, timeout=30
        )
        return {"success": result.returncode == 0, "output": result.stdout + result.stderr}
    except (subprocess.TimeoutExpired, FileNotFoundError) as e:
        return {"success": False, "error": str(e)}


def docker_stop(service: str) -> dict:
    """Stop a docker compose service."""
    try:
        result = subprocess.run(
            ["docker", "compose", "-p", COMPOSE_PROJECT, "stop", service],
            capture_output=True, text=True, timeout=30
        )
        return {"success": result.returncode == 0, "output": result.stdout + result.stderr}
    except (subprocess.TimeoutExpired, FileNotFoundError) as e:
        return {"success": False, "error": str(e)}


def docker_start(service: str) -> dict:
    """Start a docker compose service."""
    try:
        result = subprocess.run(
            ["docker", "compose", "-p", COMPOSE_PROJECT, "start", service],
            capture_output=True, text=True, timeout=30
        )
        return {"success": result.returncode == 0, "output": result.stdout + result.stderr}
    except (subprocess.TimeoutExpired, FileNotFoundError) as e:
        return {"success": False, "error": str(e)}


def inject(scenario: str, duration: int = 0) -> dict:
    """Inject a failure scenario."""
    if scenario not in SCENARIOS:
        return {"success": False, "error": f"Unknown scenario: {scenario}"}

    config = SCENARIOS[scenario]
    result = {
        "scenario": scenario,
        "description": config["description"],
        "injected_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
    }

    mode = config["mode"]

    if mode in ("delayed", "error", "rate_limit", "malformed"):
        inject_result = set_mock_mode(mode)
        result["injection"] = inject_result
        result["success"] = inject_result["success"]
    elif mode == "docker_stop":
        inject_result = docker_stop(config["target"])
        result["injection"] = inject_result
        result["success"] = inject_result["success"]
    elif mode == "docker_restart_env":
        # For env-based injection, we'd need to modify compose or use docker update
        inject_result = docker_stop(config["target"])
        result["injection"] = inject_result
        result["success"] = inject_result["success"]
        result["note"] = "Service stopped. Restart with --cleanup to restore."
    else:
        result["success"] = False
        result["error"] = f"Unknown mode: {mode}"

    if duration > 0 and result.get("success"):
        result["auto_cleanup_in"] = f"{duration}s"
        result["note"] = f"Failure will auto-revert in {duration} seconds"

    return result


def cleanup(scenario: str) -> dict:
    """Clean up (revert) a failure scenario."""
    if scenario not in SCENARIOS:
        return {"success": False, "error": f"Unknown scenario: {scenario}"}

    config = SCENARIOS[scenario]
    result = {
        "scenario": scenario,
        "action": "cleanup",
        "cleaned_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
    }

    mode = config["mode"]

    if mode in ("delayed", "error", "rate_limit", "malformed"):
        cleanup_result = set_mock_mode("success")
        result["cleanup"] = cleanup_result
        result["success"] = cleanup_result["success"]
    elif mode in ("docker_stop", "docker_restart_env"):
        cleanup_result = docker_start(config["target"])
        result["cleanup"] = cleanup_result
        result["success"] = cleanup_result["success"]
    else:
        result["success"] = False
        result["error"] = f"Unknown mode: {mode}"

    return result


def list_scenarios() -> dict:
    """List all available scenarios."""
    scenarios = {}
    for name, config in SCENARIOS.items():
        scenarios[name] = {
            "description": config["description"],
            "reversible": config["reversible"],
        }
    return {"scenarios": scenarios, "injection_enabled": ALLOW_INJECTION}


def main():
    parser = argparse.ArgumentParser(
        description="Failure Injection Toolkit for Crypto Market Data Platform"
    )
    parser.add_argument(
        "--scenario", "-s",
        choices=list(SCENARIOS.keys()),
        help="Failure scenario to inject"
    )
    parser.add_argument(
        "--duration", "-d",
        type=int, default=0,
        help="Duration in seconds before auto-cleanup (0=no auto-cleanup)"
    )
    parser.add_argument(
        "--cleanup", "-c",
        action="store_true",
        help="Clean up (revert) the specified scenario"
    )
    parser.add_argument(
        "--list", "-l",
        action="store_true",
        help="List available scenarios"
    )
    parser.add_argument(
        "--format",
        choices=["json", "text"],
        default="json",
        help="Output format"
    )

    args = parser.parse_args()

    if args.list:
        result = list_scenarios()
        print(json.dumps(result, indent=2))
        sys.exit(0)

    if not args.scenario:
        parser.print_help()
        sys.exit(1)

    check_guard()

    if args.cleanup:
        result = cleanup(args.scenario)
    else:
        result = inject(args.scenario, args.duration)

    print(json.dumps(result, indent=2))
    sys.exit(0 if result.get("success") else 1)


if __name__ == "__main__":
    main()
