# Tiltfile — Live-reload development for Crypto Market Platform
# Usage: tilt up
#
# Provides:
# - Auto-rebuild on Go file changes
# - Hot-reload for frontend (Next.js dev server)
# - Port forwarding for all services
# - Log aggregation

# ─── Go Services ──────────────────────────────────────────────────────────────

# Market API
docker_build('market-api', '.', dockerfile='deploy/docker/Dockerfile', build_args={'SERVICE': 'api'})
k8s_yaml(helm('deploy/helm/cryptomarket',
  name='cryptomarket',
  namespace='cryptomarket',
  values=['deploy/helm/cryptomarket/values-dev.yaml'],
  set=['api.image.repository=market-api', 'ingress.enabled=false']
))

# Live update for API (fast rebuild without full Docker build)
live_update('market-api',
  fall_back_on=['go.mod', 'go.sum'],
  steps=[
    sync('.', '/app'),
    run('cd /app && CGO_ENABLED=0 go build -o /usr/local/bin/service ./cmd/api'),
    restart_container(),
  ]
)

# ─── Local Resources ──────────────────────────────────────────────────────────

# Frontend dev server (runs locally, not in K8s)
local_resource('frontend',
  cmd='cd frontend && npm run dev',
  deps=['frontend/'],
  labels=['frontend'],
)

# ─── Port Forwards ────────────────────────────────────────────────────────────

k8s_resource('market-api', port_forwards=['8080:8080'])
k8s_resource('realtime-gateway', port_forwards=['8081:8081'])
k8s_resource('market-frontend', port_forwards=['3000:3000'])

# ─── Configuration ────────────────────────────────────────────────────────────

# Only watch relevant directories
watch_settings(
  ignore=['**/node_modules/**', '**/.next/**', '**/bin/**', 'docs/**']
)
