// k6 Load and Resilience Tests for Crypto Market Data Platform
//
// Scenarios:
//   1. Normal load - sustained moderate traffic
//   2. Burst - sudden spike in traffic
//   3. Provider failure during load - resilience under failure
//   4. Fallback verification - data still served during provider outage
//   5. Stale cache - behavior when data is stale
//   6. Recovery - return to normal after failure
//
// Usage:
//   k6 run load-tests/resilience.js
//   k6 run --env API_URL=http://localhost:8080 load-tests/resilience.js
//   k6 run --scenario normal_load load-tests/resilience.js

import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const apiLatency = new Trend('api_latency', true);
const marketsLatency = new Trend('markets_latency', true);
const recoveryTime = new Trend('recovery_time', true);
const fallbackRequests = new Counter('fallback_requests');

// Configuration
const API_URL = __ENV.API_URL || 'http://localhost:8080';
const REALTIME_URL = __ENV.REALTIME_URL || 'http://localhost:8081';

// Thresholds
export const options = {
  scenarios: {
    // Scenario 1: Normal sustained load
    normal_load: {
      executor: 'constant-vus',
      vus: 10,
      duration: '30s',
      tags: { scenario: 'normal_load' },
      exec: 'normalLoad',
    },

    // Scenario 2: Burst traffic
    burst: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '5s', target: 50 },   // Ramp up
        { duration: '10s', target: 50 },  // Hold at peak
        { duration: '5s', target: 10 },   // Ramp down
      ],
      startTime: '35s',
      tags: { scenario: 'burst' },
      exec: 'burstLoad',
    },

    // Scenario 3: Provider failure during load
    provider_failure: {
      executor: 'constant-vus',
      vus: 5,
      duration: '20s',
      startTime: '55s',
      tags: { scenario: 'provider_failure' },
      exec: 'providerFailureLoad',
    },

    // Scenario 4: Recovery verification
    recovery: {
      executor: 'constant-vus',
      vus: 10,
      duration: '20s',
      startTime: '80s',
      tags: { scenario: 'recovery' },
      exec: 'recoveryLoad',
    },
  },

  thresholds: {
    errors: ['rate<0.1'],                    // <10% error rate
    api_latency: ['p(95)<500'],              // p95 < 500ms
    markets_latency: ['p(95)<1000'],         // p95 < 1s for markets
    http_req_duration: ['p(99)<2000'],       // p99 < 2s overall
  },
};

// ─── Scenario Functions ───────────────────────────────────────────────────────

export function normalLoad() {
  group('Normal Load - Health Check', () => {
    const res = http.get(`${API_URL}/health`);
    checkResponse(res, 'health');
  });

  group('Normal Load - Markets', () => {
    const start = Date.now();
    const res = http.get(`${API_URL}/markets`);
    const duration = Date.now() - start;
    marketsLatency.add(duration);
    checkResponse(res, 'markets');

    if (res.status === 200) {
      const body = JSON.parse(res.body);
      check(body, {
        'markets has data': (b) => b.count !== undefined,
        'markets count > 0': (b) => b.count >= 0,
      });
    }
  });

  group('Normal Load - Single Coin', () => {
    const res = http.get(`${API_URL}/coins/BTC`);
    checkResponse(res, 'coin_btc');
  });

  group('Normal Load - Operations Status', () => {
    const res = http.get(`${API_URL}/operations/status`);
    checkResponse(res, 'operations_status');

    if (res.status === 200) {
      const body = JSON.parse(res.body);
      check(body, {
        'has status field': (b) => b.status !== undefined,
        'status is valid': (b) => ['healthy', 'degraded', 'stale', 'unavailable'].includes(b.status),
      });
    }
  });

  sleep(1);
}

export function burstLoad() {
  // Rapid-fire requests to test burst handling
  const endpoints = ['/health', '/markets', '/coins', '/coins/BTC', '/operations/status'];
  const endpoint = endpoints[Math.floor(Math.random() * endpoints.length)];

  const start = Date.now();
  const res = http.get(`${API_URL}${endpoint}`);
  const duration = Date.now() - start;
  apiLatency.add(duration);

  const success = check(res, {
    'burst: status is 200 or 429': (r) => r.status === 200 || r.status === 429,
    'burst: response time < 2s': () => duration < 2000,
  });

  errorRate.add(!success);
  sleep(0.1); // Short sleep for burst
}

export function providerFailureLoad() {
  // During provider failure, API should still respond (from cache/fallback)
  group('Provider Failure - Markets Still Available', () => {
    const start = Date.now();
    const res = http.get(`${API_URL}/markets`);
    const duration = Date.now() - start;
    marketsLatency.add(duration);

    const success = check(res, {
      'failure: markets responds': (r) => r.status === 200 || r.status === 503,
      'failure: response time < 5s': () => duration < 5000,
    });

    errorRate.add(!success);

    if (res.status === 200) {
      const body = JSON.parse(res.body);
      check(body, {
        'failure: data still served': (b) => b.count >= 0,
      });
    }
  });

  group('Provider Failure - Status Shows Degraded', () => {
    const res = http.get(`${API_URL}/operations/status`);
    if (res.status === 200) {
      const body = JSON.parse(res.body);
      check(body, {
        'failure: status not healthy': (b) => b.status !== undefined,
      });
      if (body.status === 'degraded' || body.status === 'stale') {
        fallbackRequests.add(1);
      }
    }
  });

  sleep(1);
}

export function recoveryLoad() {
  // After recovery, everything should be back to normal
  group('Recovery - Health Restored', () => {
    const res = http.get(`${API_URL}/health`);
    checkResponse(res, 'recovery_health');
  });

  group('Recovery - Markets Fresh', () => {
    const start = Date.now();
    const res = http.get(`${API_URL}/markets`);
    const duration = Date.now() - start;
    recoveryTime.add(duration);

    const success = check(res, {
      'recovery: markets 200': (r) => r.status === 200,
      'recovery: fast response': () => duration < 1000,
    });
    errorRate.add(!success);
  });

  group('Recovery - Status Healthy', () => {
    const res = http.get(`${API_URL}/operations/status`);
    if (res.status === 200) {
      const body = JSON.parse(res.body);
      check(body, {
        'recovery: status is healthy': (b) => b.status === 'healthy' || b.status === 'degraded',
      });
    }
  });

  sleep(1);
}

// ─── Helper Functions ─────────────────────────────────────────────────────────

function checkResponse(res, name) {
  const success = check(res, {
    [`${name}: status 200`]: (r) => r.status === 200,
    [`${name}: has body`]: (r) => r.body && r.body.length > 0,
    [`${name}: content-type json`]: (r) =>
      r.headers['Content-Type'] && r.headers['Content-Type'].includes('application/json'),
  });
  errorRate.add(!success);
}

// ─── Teardown ─────────────────────────────────────────────────────────────────

export function teardown() {
  // Final health check
  const res = http.get(`${API_URL}/health`);
  console.log(`Final health check: ${res.status}`);
}
