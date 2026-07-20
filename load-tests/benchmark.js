// k6 API Benchmark Suite — Multi-level load characterization
//
// Measures latency, throughput, error rate at 1, 100, 1000, and 5000 concurrent users.
// Results establish performance baselines for SLO tracking.
//
// Usage:
//   k6 run load-tests/benchmark.js
//   k6 run --env API_URL=http://localhost:8080 --env LEVEL=100 load-tests/benchmark.js

import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('benchmark_errors');
const latencyP50 = new Trend('benchmark_p50', true);
const latencyP95 = new Trend('benchmark_p95', true);
const latencyP99 = new Trend('benchmark_p99', true);
const throughput = new Counter('benchmark_requests_total');

const API_URL = __ENV.API_URL || 'http://localhost:8080';
const LEVEL = parseInt(__ENV.LEVEL || '0'); // 0 = run all levels

// Endpoint mix reflecting real traffic patterns
const ENDPOINTS = [
  { path: '/health', weight: 0.10 },
  { path: '/markets', weight: 0.40 },
  { path: '/coins', weight: 0.15 },
  { path: '/coins/BTC', weight: 0.20 },
  { path: '/coins/ETH', weight: 0.10 },
  { path: '/operations/status', weight: 0.05 },
];

function pickEndpoint() {
  const r = Math.random();
  let cumulative = 0;
  for (const ep of ENDPOINTS) {
    cumulative += ep.weight;
    if (r <= cumulative) return ep.path;
  }
  return '/markets';
}

// Scenario configurations for each load level
const scenarios = {};

if (LEVEL === 0 || LEVEL === 1) {
  scenarios['level_1_user'] = {
    executor: 'constant-vus',
    vus: 1,
    duration: '30s',
    tags: { level: '1_user' },
    exec: 'benchmarkRequest',
  };
}

if (LEVEL === 0 || LEVEL === 100) {
  scenarios['level_100_users'] = {
    executor: 'ramping-vus',
    startVUs: 0,
    stages: [
      { duration: '10s', target: 100 },
      { duration: '30s', target: 100 },
      { duration: '5s', target: 0 },
    ],
    startTime: LEVEL === 0 ? '35s' : '0s',
    tags: { level: '100_users' },
    exec: 'benchmarkRequest',
  };
}

if (LEVEL === 0 || LEVEL === 1000) {
  scenarios['level_1000_users'] = {
    executor: 'ramping-vus',
    startVUs: 0,
    stages: [
      { duration: '15s', target: 1000 },
      { duration: '30s', target: 1000 },
      { duration: '10s', target: 0 },
    ],
    startTime: LEVEL === 0 ? '85s' : '0s',
    tags: { level: '1000_users' },
    exec: 'benchmarkRequest',
  };
}

if (LEVEL === 0 || LEVEL === 5000) {
  scenarios['level_5000_users'] = {
    executor: 'ramping-vus',
    startVUs: 0,
    stages: [
      { duration: '20s', target: 5000 },
      { duration: '30s', target: 5000 },
      { duration: '10s', target: 0 },
    ],
    startTime: LEVEL === 0 ? '150s' : '0s',
    tags: { level: '5000_users' },
    exec: 'benchmarkRequest',
  };
}

export const options = {
  scenarios,
  thresholds: {
    benchmark_errors: ['rate<0.05'],
    http_req_duration: ['p(95)<500', 'p(99)<2000'],
    benchmark_p95: ['p(95)<500'],
  },
};

export function benchmarkRequest() {
  const endpoint = pickEndpoint();
  const start = Date.now();

  const res = http.get(`${API_URL}${endpoint}`, {
    headers: { 'Accept-Encoding': 'gzip' },
    timeout: '10s',
  });

  const duration = Date.now() - start;
  latencyP50.add(duration);
  latencyP95.add(duration);
  latencyP99.add(duration);
  throughput.add(1);

  const success = check(res, {
    'status is 200 or 429': (r) => r.status === 200 || r.status === 429,
    'response time < 5s': () => duration < 5000,
    'has body': (r) => r.body && r.body.length > 0,
  });

  errorRate.add(!success);

  // Minimal think time to simulate realistic spacing
  sleep(0.05);
}

export function handleSummary(data) {
  const summary = {
    timestamp: new Date().toISOString(),
    metrics: {
      total_requests: data.metrics.http_reqs ? data.metrics.http_reqs.values.count : 0,
      error_rate: data.metrics.benchmark_errors ? data.metrics.benchmark_errors.values.rate : 0,
      p50_ms: data.metrics.http_req_duration ? data.metrics.http_req_duration.values['p(50)'] : 0,
      p95_ms: data.metrics.http_req_duration ? data.metrics.http_req_duration.values['p(95)'] : 0,
      p99_ms: data.metrics.http_req_duration ? data.metrics.http_req_duration.values['p(99)'] : 0,
      max_ms: data.metrics.http_req_duration ? data.metrics.http_req_duration.values.max : 0,
      rps: data.metrics.http_reqs ? data.metrics.http_reqs.values.rate : 0,
    },
  };

  return {
    stdout: JSON.stringify(summary, null, 2) + '\n',
  };
}
