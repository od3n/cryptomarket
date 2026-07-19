#!/usr/bin/env bash
# Incident Demo Script
# Demonstrates the full resilience lifecycle:
#   1. Healthy baseline
#   2. Inject provider rate limiting (429)
#   3. Observe fallback activation and degraded mode
#   4. Recovery and circuit breaker close
#
# Prerequisites:
#   - Docker Compose stack running (make up)
#   - Mock provider configured as primary or fallback
#   - ALLOW_FAILURE_INJECTION=true
#
# Usage:
#   ./scripts/incident-demo.sh
#   make incident-demo

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
MOCK_URL="${MOCK_PROVIDER_URL:-http://localhost:8082}"
API_URL="${API_URL:-http://localhost:8080}"
TOOLKIT="$PROJECT_DIR/sre-toolkit/inject_failures.py"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_step() {
    echo -e "\n${BLUE}━━━ STEP $1: $2 ━━━${NC}\n"
}

log_ok() {
    echo -e "${GREEN}✓ $1${NC}"
}

log_warn() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

log_fail() {
    echo -e "${RED}✗ $1${NC}"
}

check_health() {
    local url="$1"
    local name="$2"
    if curl -sf "$url/health" > /dev/null 2>&1; then
        log_ok "$name is healthy"
        return 0
    else
        log_fail "$name is not responding"
        return 1
    fi
}

check_status() {
    local status
    status=$(curl -sf "$API_URL/operations/status" 2>/dev/null | python3 -c "import sys,json; print(json.load(sys.stdin).get('status','unknown'))" 2>/dev/null || echo "unknown")
    echo "$status"
}

wait_for_status() {
    local expected="$1"
    local timeout="${2:-30}"
    local elapsed=0
    while [ $elapsed -lt $timeout ]; do
        local current
        current=$(check_status)
        if [ "$current" = "$expected" ]; then
            return 0
        fi
        sleep 2
        elapsed=$((elapsed + 2))
    done
    return 1
}

# ─── Preflight Checks ────────────────────────────────────────────────────────

echo -e "${BLUE}╔══════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║   Crypto Market Platform - Incident Demo       ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════╝${NC}"

log_step 0 "Preflight Checks"

check_health "$API_URL" "API" || { log_fail "Start the stack first: make up"; exit 1; }
check_health "$MOCK_URL" "Mock Provider" || log_warn "Mock provider not running (some steps may skip)"

export ALLOW_FAILURE_INJECTION=true

# ─── Step 1: Healthy Baseline ────────────────────────────────────────────────

log_step 1 "Healthy Baseline"

echo "Current platform status:"
curl -sf "$API_URL/operations/status" | python3 -m json.tool 2>/dev/null || echo "  (status endpoint not available)"

STATUS=$(check_status)
if [ "$STATUS" = "healthy" ]; then
    log_ok "Platform is healthy"
else
    log_warn "Platform status: $STATUS (continuing anyway)"
fi

echo ""
echo "Fetching market data..."
MARKETS=$(curl -sf "$API_URL/markets" 2>/dev/null | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'  {d.get(\"count\", 0)} symbols available')" 2>/dev/null || echo "  (markets endpoint not available)")
echo "$MARKETS"

sleep 3

# ─── Step 2: Inject Rate Limiting ────────────────────────────────────────────

log_step 2 "Inject Provider Rate Limiting (429)"

echo "Injecting provider_429 failure..."
python3 "$TOOLKIT" --scenario provider_429 2>/dev/null || log_warn "Injection failed (mock provider may not be running)"

echo "Waiting for circuit breaker to detect failures..."
sleep 10

# ─── Step 3: Observe Degraded State ──────────────────────────────────────────

log_step 3 "Observe Fallback & Degraded Mode"

echo "Platform status after injection:"
curl -sf "$API_URL/operations/status" | python3 -m json.tool 2>/dev/null || echo "  (status endpoint not available)"

STATUS=$(check_status)
case "$STATUS" in
    degraded)
        log_ok "Platform correctly reports DEGRADED state"
        ;;
    stale)
        log_warn "Platform reports STALE (fallback may not be configured)"
        ;;
    unavailable)
        log_fail "Platform reports UNAVAILABLE (all providers down)"
        ;;
    healthy)
        log_warn "Platform still reports HEALTHY (injection may not have taken effect)"
        ;;
    *)
        log_warn "Platform status: $STATUS"
        ;;
esac

echo ""
echo "Checking Prometheus alerts..."
ALERTS=$(curl -sf "http://localhost:9090/api/v1/alerts" 2>/dev/null | python3 -c "
import sys, json
data = json.load(sys.stdin)
alerts = data.get('data', {}).get('alerts', [])
active = [a for a in alerts if a.get('state') == 'firing']
if active:
    for a in active:
        print(f'  FIRING: {a[\"labels\"].get(\"alertname\", \"unknown\")} ({a[\"labels\"].get(\"severity\", \"?\")})')
else:
    print('  No active alerts (may need more time to fire)')
" 2>/dev/null || echo "  (Prometheus not available)")
echo "$ALERTS"

sleep 5

# ─── Step 4: Recovery ────────────────────────────────────────────────────────

log_step 4 "Recovery - Restore Provider"

echo "Cleaning up injected failure..."
python3 "$TOOLKIT" --scenario provider_429 --cleanup 2>/dev/null || log_warn "Cleanup failed"

echo "Waiting for circuit breaker to close and provider to recover..."
sleep 15

# ─── Step 5: Verify Recovery ─────────────────────────────────────────────────

log_step 5 "Verify Recovery"

echo "Platform status after recovery:"
curl -sf "$API_URL/operations/status" | python3 -m json.tool 2>/dev/null || echo "  (status endpoint not available)"

STATUS=$(check_status)
if [ "$STATUS" = "healthy" ]; then
    log_ok "Platform recovered to HEALTHY state"
else
    log_warn "Platform status: $STATUS (may need more time to recover)"
fi

# ─── Summary ──────────────────────────────────────────────────────────────────

echo ""
echo -e "${BLUE}━━━ DEMO COMPLETE ━━━${NC}"
echo ""
echo "Timeline:"
echo "  1. Baseline: Platform healthy, primary provider active"
echo "  2. Injection: Provider returns 429, circuit breaker opens"
echo "  3. Degraded: Fallback provider activated, alerts fire"
echo "  4. Recovery: Provider restored, circuit breaker closes"
echo ""
echo "Key observations:"
echo "  - Circuit breaker prevented cascading failures"
echo "  - Fallback provider maintained data flow"
echo "  - Alerts notified operators of degradation"
echo "  - Automatic recovery without manual intervention"
echo ""
echo "To reset: make incident-reset"
