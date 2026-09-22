import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Trend } from 'k6/metrics';

const semanticHits = new Counter('semantic_cache_hits');
const exactHits = new Counter('exact_cache_hits');
const semanticMisses = new Counter('semantic_cache_misses');

export const options = {
  stages: [
    { duration: '10s', target: 10 },
    { duration: '30s', target: 10 },
    { duration: '10s', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(95)<1000'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

const queries = [
  'Can you explain general relativity simply?',
  "Describe Einstein's theory of general relativity for beginners",
  'What is general relativity in layman\'s terms?',
  'How does general relativity work in simple words?',
  'Break down general relativity for someone who doesn\'t know physics',
];

export function setup() {
  const url = `${BASE_URL}/v1/chat/completions`;
  const payload = JSON.stringify({
    messages: [
      {
        role: 'user',
        content: 'Explain the theory of general relativity in simple terms',
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
  const query = queries[__ITER % 5];
  const url = `${BASE_URL}/v1/chat/completions`;
  const payload = JSON.stringify({
    messages: [
      {
        role: 'user',
        content: query,
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
    'status is 200': (r) => r.status === 200,
  });
  const cacheStatus = res.headers['X-Cache'] || res.headers['x-cache'];
  if (cacheStatus === 'SEMANTIC_HIT') {
    semanticHits.add(1);
  } else if (cacheStatus === 'EXACT_HIT') {
    exactHits.add(1);
  } else {
    semanticMisses.add(1);
  }
  sleep(0.5);
}
