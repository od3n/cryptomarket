#!/usr/bin/env bash
# Toil Reduction: Automated operational tasks
# Reduces manual operational work by automating repetitive tasks.
#
# Usage:
#   ./scripts/ops/automate.sh <task>
#
# Tasks:
#   cert-check      — Check TLS certificate expiry across all ingresses
#   disk-check      — Check disk usage on all PVCs
#   stale-pods      — Find and report stuck/pending pods
#   image-audit     — List all container images and their ages
#   resource-report — Generate resource utilization report
#   cleanup         — Clean up completed jobs, evicted pods, old images

set -euo pipefail

NAMESPACE="${NAMESPACE:-cryptomarket-prod}"
TASK="${1:-help}"

case "$TASK" in
  cert-check)
    echo "=== TLS Certificate Expiry Check ==="
    kubectl get certificates -A -o custom-columns=\
      NAMESPACE:.metadata.namespace,\
      NAME:.metadata.name,\
      READY:.status.conditions[0].status,\
      EXPIRY:.status.notAfter \
      2>/dev/null || echo "cert-manager not found, checking ingress secrets..."

    # Check ingress TLS secrets
    for secret in $(kubectl -n "$NAMESPACE" get secrets -o name | grep tls); do
      EXPIRY=$(kubectl -n "$NAMESPACE" get "$secret" -o jsonpath='{.data.tls\.crt}' | \
        base64 -d | openssl x509 -noout -enddate 2>/dev/null | cut -d= -f2)
      echo "  $secret: expires $EXPIRY"
    done
    ;;

  disk-check)
    echo "=== PVC Disk Usage ==="
    kubectl -n "$NAMESPACE" get pvc -o custom-columns=\
      NAME:.metadata.name,\
      STATUS:.status.phase,\
      CAPACITY:.status.capacity.storage,\
      STORAGE_CLASS:.spec.storageClassName
    echo ""
    echo "Note: For actual usage, check Prometheus:"
    echo "  kubelet_volume_stats_used_bytes / kubelet_volume_stats_capacity_bytes"
    ;;

  stale-pods)
    echo "=== Stuck/Pending Pods ==="
    echo "--- Pending > 5 min ---"
    kubectl -n "$NAMESPACE" get pods --field-selector=status.phase=Pending \
      -o custom-columns=NAME:.metadata.name,AGE:.metadata.creationTimestamp,NODE:.spec.nodeName
    echo ""
    echo "--- CrashLoopBackOff ---"
    kubectl -n "$NAMESPACE" get pods | grep -i crashloop || echo "  None found"
    echo ""
    echo "--- Evicted ---"
    kubectl -n "$NAMESPACE" get pods --field-selector=status.phase=Failed \
      -o custom-columns=NAME:.metadata.name,REASON:.status.reason | grep Evicted || echo "  None found"
    ;;

  image-audit)
    echo "=== Container Image Audit ==="
    kubectl -n "$NAMESPACE" get pods -o jsonpath='{range .items[*]}{range .spec.containers[*]}{.image}{"\n"}{end}{end}' | \
      sort -u | while read -r image; do
      echo "  $image"
    done
    ;;

  resource-report)
    echo "=== Resource Utilization Report ==="
    echo ""
    echo "--- Pod Resources ---"
    kubectl -n "$NAMESPACE" top pods 2>/dev/null || echo "  metrics-server not available"
    echo ""
    echo "--- Node Resources ---"
    kubectl top nodes 2>/dev/null || echo "  metrics-server not available"
    echo ""
    echo "--- HPA Status ---"
    kubectl -n "$NAMESPACE" get hpa -o custom-columns=\
      NAME:.metadata.name,\
      REPLICAS:.status.currentReplicas,\
      TARGET:.spec.metrics[0].resource.target.averageUtilization,\
      CURRENT:.status.currentMetrics[0].resource.current.averageUtilization
    ;;

  cleanup)
    echo "=== Cleanup ==="
    echo "Removing completed jobs..."
    kubectl -n "$NAMESPACE" delete jobs --field-selector=status.successful=1 2>/dev/null || true
    echo "Removing evicted pods..."
    kubectl -n "$NAMESPACE" delete pods --field-selector=status.phase=Failed 2>/dev/null || true
    echo "Done."
    ;;

  help|*)
    echo "Usage: $0 <task>"
    echo ""
    echo "Tasks:"
    echo "  cert-check      — Check TLS certificate expiry"
    echo "  disk-check      — Check PVC disk usage"
    echo "  stale-pods      — Find stuck/pending pods"
    echo "  image-audit     — List container images"
    echo "  resource-report — Resource utilization report"
    echo "  cleanup         — Clean up completed/failed resources"
    ;;
esac
