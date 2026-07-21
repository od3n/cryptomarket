// k6 Scale Test — 5000 VU Mixed Workload
//
// Validates the platform handles 5000 concurrent users with:
// - 70% read traffic (markets, coins)
// - 20% SSE connections (realtime)
// - 10% burst/health checks
//
// Usage:
//   k6 run load-tests/scale.js
//   k6 run --env API_URL=http://localhost:8080 load-tests/scale.js

import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate, Trend, Counter, Gauge } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const apiLatency = new Trend('api_latency', true);
const sseConnections = new Gauge('sse_active_connections');
const throughput = new Counter('total_requests');

const API_URL = __ENV.API_URL || 'http://localhost:8080';
const REALTIME_URL = __ENV.REALTIME_URL || 'http://localhost:8081';

export const options = {
  scenarios: {
    // 70% read traffic — markets and coins
    read_traffic: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 3500 },  // Ramp to 3500
        { duration: '2m', target: 3500 },   // Hold at 3500
        { duration: '30s', target: 0 },     // Ramp down
      ],
      exec: 'readTraffic',
      tags: { scenario: 'read' },
    },

    // 20% SSE connections
    sse_connections: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 1000 },
        { duration: '2m', target: 1000 },
        { duration: '30s', target: 0 },
      ],
      exec: 'sseTraffic',
      tags: { scenario: 'sse' },
    },

    // 10% burst/health
    burst_traffic: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '10s', target: 500 },
        { duration: '30s', target: 500 },
        { duration: '10s', target: 0 },
        { duration: '30s', target: 0 },   // Pause
        { duration: '10s', target: 500 },  // Second burst
        { duration: '30s', target: 500 },
        { duration: '10s', target: 0 },
      ],
      exec: 'burstTraffic',
      tags: { scenario: 'burst' },
    },
  },

  thresholds: {
    errors: ['rate<0.05'],                    // <5% error rate at scale
    api_latency: ['p(95)<500', 'p(99)<1000'], // p95 < 500ms, p99 < 1s
    http_req_duration: ['p(99)<2000'],         // Overall p99 < 2s
  },
};

// ─── Read Traffic (70%) ──────────────────────────────────────────────────────

export function readTraffic() {
  const endpoints = [
    { path: '/markets', weight: 40 },
    { path: '/coins', weight: 20 },
    { path: '/coins/BTC', weight: 15 },
    { path: '/coins/ETH', weight: 10 },
    { path: '/coins/BTC/history?limit=10', weight: 10 },
    { path: '/operations/status', weight: 5 },
  ];

  const endpoint = weightedRandom(endpoints);
  const start = Date.now();
  const res = http.get(`${API_URL}${endpoint.path}`);
  const duration = Date.now() - start;

  apiLatency.add(duration);
  throughput.add(1);

  const success = check(res, {
    'read: status 200': (r) => r.status === 200,
    'read: has body': (r) => r.body && r.body.length > 0,
    'read: fast response': () => duration < 1000,
  });

  errorRate.add(!success);
  sleep(Math.random() * 2 + 0.5); // 0.5-2.5s think time
}

// ─── SSE Traffic (20%) ───────────────────────────────────────────────────────

export function sseTraffic() {
  // Simulate SSE connection by hitting the health endpoint
  // (actual SSE would use a persistent connection)
  const res = http.get(`${REALTIME_URL}/health`);
  sseConnections.add(1);

  check(res, {
    'sse: gateway healthy': (r) => r.status === 200,
  });

  // Simulate connection duration
  sleep(Math.random() * 10 + 5); // 5-15s connection
  sseConnections.add(-1);
}

// ─── Burst Traffic (10%) ─────────────────────────────────────────────────────

export function burstTraffic() {
  const endpoints = ['/health', '/markets', '/coins', '/providers/status'];
  const endpoint = endpoints[Math.floor(Math.random() * endpoints.length)];

  const start = Date.now();
  const res = http.get(`${API_URL}${endpoint}`);
  const duration = Date.now() - start;

  apiLatency.add(duration);
  throughput.add(1);

  const success = check(res, {
    'burst: status 200 or 429': (r) => r.status === 200 || r.status === 429,
    'burst: response < 2s': () => duration < 2000,
  });

  errorRate.add(!success);
  sleep(0.1); // Minimal think time for burst
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

function weightedRandom(items) {
  const totalWeight = items.reduce((sum, item) => sum + item.weight, 0);
  let random = Math.random() * totalWeight;

  for (const item of items) {
    random -= item.weight;
    if (random <= 0) return item;
  }
  return items[items.length - 1];
}

export function teardown() {
  const res = http.get(`${API_URL}/health`);
  console.log(`Final health: ${res.status}`);
}
