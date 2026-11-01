import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate } from 'k6/metrics';

const exactHits = new Counter('exact_hits');
const semanticHits = new Counter('semantic_hits');
const coldMisses = new Counter('cold_misses');
const errorRate = new Rate('errors');

export const options = {
  stages: [
    { duration: '5s', target: 5 },
    { duration: '20s', target: 5 },
    { duration: '5s', target: 0 },
  ],
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

const diversePrompts = [
  'Describe the biochemical process of photosynthesis in C3 plants',
  'Explain the rise and fall of the Byzantine empire under Justinian',
  'Write an SQL query to find top 5 customers with window functions',
  'How does an airplane wing generate aerodynamic lift using Bernoulli principles',
  'Explain the difference between TCP Reno and TCP BBR congestion control',
];

export function setup() {
  const url = `${BASE_URL}/v1/chat/completions`;
  const seedExact = JSON.stringify({
    messages: [{ role: 'user', content: 'What is the capital of France?' }],
    model: 'gemini-3.5-flash-lite',
  });
  http.post(url, seedExact, { headers: { 'Content-Type': 'application/json' } });

  const seedSemantic = JSON.stringify({
    messages: [{ role: 'user', content: 'Explain general relativity simply' }],
    model: 'gemini-3.5-flash-lite',
  });
  http.post(url, seedSemantic, { headers: { 'Content-Type': 'application/json' } });
  sleep(2);
}

export default function () {
  const rand = Math.random();
  const url = `${BASE_URL}/v1/chat/completions`;

  if (rand < 0.70) {
    const payload = JSON.stringify({
      messages: [{ role: 'user', content: 'What is the capital of France?' }],
      model: 'gemini-3.5-flash-lite',
    });
    const res = http.post(url, payload, { headers: { 'Content-Type': 'application/json' } });
    const isExact = res.headers['X-Cache'] === 'EXACT_HIT' || res.headers['x-cache'] === 'EXACT_HIT';
    check(res, { 'status is 200': (r) => r.status === 200 });
    if (isExact) {
      exactHits.add(1);
      errorRate.add(0);
    } else {
      errorRate.add(1);
    }
    sleep(0.05);
  } else if (rand < 0.90) {
    const payload = JSON.stringify({
      messages: [{ role: 'user', content: "Explain Einstein's relativity theory in easy language" }],
      model: 'gemini-3.5-flash-lite',
    });
    const res = http.post(url, payload, {
      headers: {
        'Content-Type': 'application/json',
        'X-Bypass-Exact-Cache': 'true',
      },
    });
    const isSemantic = res.headers['X-Cache'] === 'SEMANTIC_HIT' || res.headers['x-cache'] === 'SEMANTIC_HIT';
    check(res, { 'status is 200': (r) => r.status === 200 });
    if (isSemantic) {
      semanticHits.add(1);
      errorRate.add(0);
    } else {
      errorRate.add(1);
    }
    sleep(0.2);
  } else {
    const prompt = `${diversePrompts[__ITER % diversePrompts.length]} [id ${__VU}_${__ITER}_${Date.now()}]`;
    const payload = JSON.stringify({
      messages: [{ role: 'user', content: prompt }],
      model: 'gemini-3.5-flash-lite',
    });
    const res = http.post(url, payload, {
      headers: {
        'Content-Type': 'application/json',
        'X-Bypass-Exact-Cache': 'true',
      },
      timeout: '30s',
    });
    const isMiss = res.headers['X-Cache'] === 'MISS' || res.headers['x-cache'] === 'MISS';
    check(res, { 'status is 200': (r) => r.status === 200 });
    if (res.status === 200 && isMiss) {
      coldMisses.add(1);
      errorRate.add(0);
    } else {
      errorRate.add(1);
    }
    sleep(0.5);
  }
}

