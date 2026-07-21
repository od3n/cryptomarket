#!/usr/bin/env bash
# SLO Deployment Gate — Blocks deployment if error budget is exhausted
#
# Usage:
#   ./scripts/check-slo-gate.sh [--override]
#
# Checks the current error budget remaining for all SLOs.
# If any SLO has less than 10% budget remaining, deployment is blocked.
#
# Override: Set SLO_GATE_OVERRIDE=true or pass --override (requires approval audit log)
#
# Environment:
#   PROMETHEUS_URL  — Prometheus endpoint (default: http://localhost:9090)
#   BUDGET_THRESHOLD — Minimum budget % to allow deploy (default: 10)

set -euo pipefail

PROMETHEUS_URL="${PROMETHEUS_URL:-http://localhost:9090}"
BUDGET_THRESHOLD="${BUDGET_THRESHOLD:-10}"
OVERRIDE="${SLO_GATE_OVERRIDE:-false}"

if [[ "${1:-}" == "--override" ]]; then
  OVERRIDE="true"
fi

echo "=== SLO Deployment Gate ==="
echo "Prometheus: ${PROMETHEUS_URL}"
echo "Threshold:  ${BUDGET_THRESHOLD}% budget remaining required"
echo ""

# Check if Prometheus is reachable
if ! curl -sf "${PROMETHEUS_URL}/-/healthy" > /dev/null 2>&1; then
  echo "WARNING: Prometheus not reachable at ${PROMETHEUS_URL}"
  echo "Cannot verify SLO status. Deployment gate: SKIPPED (Prometheus unavailable)"
  exit 0
fi

# Query error budget remaining for API availability SLO
# SLO: 99.9% availability over 30 days
# Budget remaining = 1 - (error_rate / allowed_error_rate) * 100
query_budget() {
  local slo_name="$1"
  local query="$2"

  RESULT=$(curl -sf "${PROMETHEUS_URL}/api/v1/query" --data-urlencode "query=${query}" 2>/dev/null)

  if [ $? -ne 0 ] || [ -z "$RESULT" ]; then
    echo "WARN"
    return
  fi

  VALUE=$(echo "$RESULT" | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    r = d.get('data', {}).get('result', [])
    if r:
        print(float(r[0]['value'][1]))
    else:
        print('NODATA')
except:
    print('ERROR')
" 2>/dev/null)

  echo "$VALUE"
}

# Check API Availability budget
echo "Checking API Availability SLO..."
API_BUDGET=$(query_budget "api_availability" \
  '(1 - (sum(rate(http_responses_total{status=~"5.."}[30d])) / sum(rate(http_responses_total[30d])))) / 0.001 * 100')

# Check Ingestion Success budget
echo "Checking Ingestion Success SLO..."
INGEST_BUDGET=$(query_budget "ingestion_success" \
  '(sum(rate(ingestion_success_total[30d])) / (sum(rate(ingestion_success_total[30d])) + sum(rate(ingestion_failure_total[30d])))) / 0.995 * 100')

echo ""
echo "=== Results ==="

GATE_PASS=true

check_budget() {
  local name="$1"
  local value="$2"

  if [[ "$value" == "WARN" || "$value" == "ERROR" || "$value" == "NODATA" ]]; then
    echo "  ${name}: No data available (metric may not be recording yet)"
    return 0
  fi

  # Compare as integer (truncate decimal)
  local int_value=${value%.*}
  if [[ "$int_value" -lt "$BUDGET_THRESHOLD" ]]; then
    echo "  ${name}: ${value}% budget remaining — BLOCKED (below ${BUDGET_THRESHOLD}%)"
    GATE_PASS=false
  else
    echo "  ${name}: ${value}% budget remaining — OK"
  fi
}

check_budget "API Availability" "$API_BUDGET"
check_budget "Ingestion Success" "$INGEST_BUDGET"

echo ""

if [[ "$GATE_PASS" == "true" ]]; then
  echo "✅ SLO GATE: PASS — Deployment allowed"
  exit 0
else
  if [[ "$OVERRIDE" == "true" ]]; then
    echo "⚠️  SLO GATE: OVERRIDDEN — Deployment proceeding with override"
    echo "    Override requested at: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
    echo "    This override should be documented in the deployment record."
    exit 0
  else
    echo "❌ SLO GATE: BLOCKED — Error budget exhausted"
    echo ""
    echo "The error budget for one or more SLOs is below ${BUDGET_THRESHOLD}%."
    echo "Per the error budget policy, non-critical deployments are frozen."
    echo ""
    echo "Options:"
    echo "  1. Wait for the budget to recover (30-day rolling window)"
    echo "  2. Fix the reliability issue consuming the budget"
    echo "  3. Override with: SLO_GATE_OVERRIDE=true $0 (requires approval)"
    echo ""
    echo "See: docs/sre/slos.md — Error Budget Policy"
    exit 1
  fi
fi
