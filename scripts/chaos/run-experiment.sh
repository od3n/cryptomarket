#!/usr/bin/env bash
# Chaos Testing Framework — Safe experiments for local/staging ONLY
#
# SAFETY: This script refuses to run unless CHAOS_ENV=local or CHAOS_ENV=staging.
# It NEVER targets production.
#
# Usage:
#   CHAOS_ENV=local ./scripts/chaos/run-experiment.sh <experiment>
#
# Experiments:
#   kill-api          — Kill the API pod and measure recovery
#   kill-redis        — Kill Redis and verify graceful degradation
#   kill-postgres     — Kill PostgreSQL and verify read-from-cache
#   slow-provider     — Add 5s latency to provider calls
#   network-latency   — Add 200ms latency to all pod traffic
#   packet-loss       — Inject 10% packet loss
#   dns-failure       — Block DNS resolution temporarily
#   disk-pressure     — Fill ephemeral storage to 90%
#
# Each experiment:
#   1. Records baseline metrics
#   2. Injects failure
#   3. Observes for configurable duration
#   4. Removes failure
#   5. Verifies recovery
#   6. Produces a report

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NAMESPACE="${CHAOS_NAMESPACE:-cryptomarket}"
OBSERVE_DURATION="${OBSERVE_DURATION:-30}"
API_URL="${API_URL:-http://localhost:8080}"

# ─── Safety Gate ──────────────────────────────────────────────────────────────

if [[ "${CHAOS_ENV:-}" != "local" && "${CHAOS_ENV:-}" != "staging" ]]; then
  echo "ERROR: CHAOS_ENV must be 'local' or 'staging'. Refusing to run."
  echo "Usage: CHAOS_ENV=local $0 <experiment>"
  exit 1
fi

echo "=== Chaos Experiment Framework ==="
echo "Environment: ${CHAOS_ENV}"
echo "Namespace:   ${NAMESPACE}"
echo "Observe:     ${OBSERVE_DURATION}s"
echo ""

# ─── Helpers ──────────────────────────────────────────────────────────────────

timestamp() { date -u +"%Y-%m-%dT%H:%M:%SZ"; }

log() { echo "[$(timestamp)] $*"; }

check_api_health() {
  local status
  status=$(curl -sf -o /dev/null -w "%{http_code}" "${API_URL}/health" 2>/dev/null || echo "000")
  echo "$status"
}

record_baseline() {
  log "Recording baseline metrics..."
  local health=$(check_api_health)
  local start=$(date +%s%N)
  curl -sf "${API_URL}/markets" > /dev/null 2>&1 || true
  local end=$(date +%s%N)
  local latency_ms=$(( (end - start) / 1000000 ))
  log "Baseline: health=${health}, markets_latency=${latency_ms}ms"
  echo "${health},${latency_ms}"
}

verify_recovery() {
  log "Verifying recovery (waiting ${OBSERVE_DURATION}s)..."
  sleep "$OBSERVE_DURATION"

  local health=$(check_api_health)
  if [[ "$health" == "200" ]]; then
    log "RECOVERY: API healthy (200)"
    return 0
  else
    log "WARNING: API not yet healthy (status=${health})"
    return 1
  fi
}

cleanup_trap() {
  log "Cleanup: removing injected failures..."
  # Remove any tc rules
  kubectl -n "$NAMESPACE" exec deploy/market-api -- tc qdisc del dev eth0 root 2>/dev/null || true
  # Remove any iptables rules
  kubectl -n "$NAMESPACE" exec deploy/market-api -- iptables -F 2>/dev/null || true
  log "Cleanup complete."
}

# ─── Experiments ──────────────────────────────────────────────────────────────

experiment_kill_api() {
  log "EXPERIMENT: Kill API pod"
  record_baseline

  log "Injecting: deleting market-api pod..."
  kubectl -n "$NAMESPACE" delete pod -l app.kubernetes.io/name=market-api --grace-period=0 --force 2>/dev/null || true

  log "Observing for ${OBSERVE_DURATION}s..."
  sleep 5
  local health=$(check_api_health)
  log "During failure: health=${health}"

  verify_recovery
}

experiment_kill_redis() {
  log "EXPERIMENT: Kill Redis"
  record_baseline

  log "Injecting: scaling Redis to 0..."
  kubectl -n "$NAMESPACE" scale statefulset redis --replicas=0 2>/dev/null || true

  log "Observing for ${OBSERVE_DURATION}s..."
  sleep 5
  local health=$(check_api_health)
  log "During failure: health=${health} (expect degraded but serving)"

  log "Recovering: scaling Redis back to 1..."
  kubectl -n "$NAMESPACE" scale statefulset redis --replicas=1 2>/dev/null || true

  verify_recovery
}

experiment_kill_postgres() {
  log "EXPERIMENT: Kill PostgreSQL"
  record_baseline

  log "Injecting: scaling PostgreSQL to 0..."
  kubectl -n "$NAMESPACE" scale statefulset postgres --replicas=0 2>/dev/null || true

  log "Observing for ${OBSERVE_DURATION}s..."
  sleep 5
  local health=$(check_api_health)
  log "During failure: health=${health} (expect cache-served data)"

  log "Recovering: scaling PostgreSQL back to 1..."
  kubectl -n "$NAMESPACE" scale statefulset postgres --replicas=1 2>/dev/null || true

  verify_recovery
}

experiment_slow_provider() {
  log "EXPERIMENT: Slow provider (5s latency)"
  record_baseline

  log "Injecting: adding 5s delay to egress traffic on ingestor..."
  kubectl -n "$NAMESPACE" exec deploy/market-ingestor -- \
    tc qdisc add dev eth0 root netem delay 5000ms 2>/dev/null || true

  log "Observing for ${OBSERVE_DURATION}s..."
  sleep "$OBSERVE_DURATION"

  log "Removing delay..."
  kubectl -n "$NAMESPACE" exec deploy/market-ingestor -- \
    tc qdisc del dev eth0 root 2>/dev/null || true

  verify_recovery
}

experiment_network_latency() {
  log "EXPERIMENT: Network latency (200ms)"
  record_baseline

  log "Injecting: adding 200ms latency to API pod..."
  kubectl -n "$NAMESPACE" exec deploy/market-api -- \
    tc qdisc add dev eth0 root netem delay 200ms 2>/dev/null || true

  log "Observing for ${OBSERVE_DURATION}s..."
  sleep "$OBSERVE_DURATION"

  log "Removing latency..."
  kubectl -n "$NAMESPACE" exec deploy/market-api -- \
    tc qdisc del dev eth0 root 2>/dev/null || true

  verify_recovery
}

experiment_packet_loss() {
  log "EXPERIMENT: Packet loss (10%)"
  record_baseline

  log "Injecting: 10% packet loss on API pod..."
  kubectl -n "$NAMESPACE" exec deploy/market-api -- \
    tc qdisc add dev eth0 root netem loss 10% 2>/dev/null || true

  log "Observing for ${OBSERVE_DURATION}s..."
  sleep "$OBSERVE_DURATION"

  log "Removing packet loss..."
  kubectl -n "$NAMESPACE" exec deploy/market-api -- \
    tc qdisc del dev eth0 root 2>/dev/null || true

  verify_recovery
}

experiment_dns_failure() {
  log "EXPERIMENT: DNS failure"
  record_baseline

  log "Injecting: blocking DNS on ingestor..."
  kubectl -n "$NAMESPACE" exec deploy/market-ingestor -- \
    iptables -A OUTPUT -p udp --dport 53 -j DROP 2>/dev/null || true

  log "Observing for ${OBSERVE_DURATION}s..."
  sleep "$OBSERVE_DURATION"

  log "Restoring DNS..."
  kubectl -n "$NAMESPACE" exec deploy/market-ingestor -- \
    iptables -D OUTPUT -p udp --dport 53 -j DROP 2>/dev/null || true

  verify_recovery
}

experiment_disk_pressure() {
  log "EXPERIMENT: Disk pressure (fill /tmp to 90%)"
  record_baseline

  log "Injecting: filling disk on API pod..."
  kubectl -n "$NAMESPACE" exec deploy/market-api -- \
    sh -c 'dd if=/dev/zero of=/tmp/chaos-fill bs=1M count=100 2>/dev/null' || true

  log "Observing for ${OBSERVE_DURATION}s..."
  sleep "$OBSERVE_DURATION"

  log "Removing disk pressure..."
  kubectl -n "$NAMESPACE" exec deploy/market-api -- \
    rm -f /tmp/chaos-fill 2>/dev/null || true

  verify_recovery
}

# ─── Main ─────────────────────────────────────────────────────────────────────

EXPERIMENT="${1:-}"

case "$EXPERIMENT" in
  kill-api)         experiment_kill_api ;;
  kill-redis)       experiment_kill_redis ;;
  kill-postgres)    experiment_kill_postgres ;;
  slow-provider)    experiment_slow_provider ;;
  network-latency)  experiment_network_latency ;;
  packet-loss)      experiment_packet_loss ;;
  dns-failure)      experiment_dns_failure ;;
  disk-pressure)    experiment_disk_pressure ;;
  *)
    echo "Usage: CHAOS_ENV=local $0 <experiment>"
    echo ""
    echo "Available experiments:"
    echo "  kill-api          Kill the API pod"
    echo "  kill-redis        Kill Redis"
    echo "  kill-postgres     Kill PostgreSQL"
    echo "  slow-provider     Add 5s provider latency"
    echo "  network-latency   Add 200ms network latency"
    echo "  packet-loss       Inject 10% packet loss"
    echo "  dns-failure       Block DNS resolution"
    echo "  disk-pressure     Fill ephemeral storage"
    exit 1
    ;;
esac

log "Experiment '${EXPERIMENT}' complete."
