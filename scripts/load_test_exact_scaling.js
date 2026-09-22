import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate } from 'k6/metrics';

const cacheHits = new Counter('cache_hits');
const errorRate = new Rate('errors');

export const options = {
  stages: [
    { duration: '10s', target: 100 },
    { duration: '20s', target: 100 },
    { duration: '10s', target: 0 },
  ],
  thresholds: {
    errors: ['rate<0.01'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export function setup() {
  const url = `${BASE_URL}/v1/chat/completions`;
  const payload = JSON.stringify({
    messages: [
      {
        role: 'user',
        content: 'What is the capital of France?',
      },
    ],
    model: 'gemini-3.5-flash-lite',
  });
  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
  };
  const res = http.post(url, payload, params);
  check(res, {
    'setup status is 200': (r) => r.status === 200,
  });
}

export default function () {
  const url = `${BASE_URL}/v1/chat/completions`;
  const payload = JSON.stringify({
    messages: [
      {
        role: 'user',
        content: 'What is the capital of France?',
      },
    ],
    model: 'gemini-3.5-flash-lite',
  });
  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
  };
  const res = http.post(url, payload, params);
  const success = check(res, {
    'status is 200': (r) => r.status === 200,
    'exact hit header': (r) => r.headers['X-Cache'] === 'EXACT_HIT' || r.headers['x-cache'] === 'EXACT_HIT',
  });
  if (success) {
    cacheHits.add(1);
    errorRate.add(0);
  } else {
    errorRate.add(1);
  }
  sleep(0.05);
}

