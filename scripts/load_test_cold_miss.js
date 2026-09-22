import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate } from 'k6/metrics';

const coldMisses = new Counter('cold_misses');
const semanticHits = new Counter('unexpected_semantic_hits');
const errorRate = new Rate('errors');

const diversePrompts = [
  'Describe the biochemical process of photosynthesis in C3 plants',
  'Explain the rise and fall of the Byzantine empire under Justinian',
  'Write an SQL query to find top 5 customers with window functions',
  'How does an airplane wing generate aerodynamic lift using Bernoulli principles',
  'Explain the difference between TCP Reno and TCP BBR congestion control',
  'What are the key ingredients in authentic Neapolitan pizza dough',
  'How do black holes produce Hawking radiation through quantum fluctuations',
  'Explain the difference between optimistic and pessimistic locking in databases',
  'How do mRNA vaccines instruct human cells to produce spike proteins',
  'What is the role of mitochondria in cellular respiration and ATP generation',
  'Explain the architectural differences between RISC-V and x86 architectures',
  'How does the immune system differentiate between self and non-self antigens',
  'Explain the mechanics of plate tectonics and subduction zones',
  'What is the mathematical definition of a Hilbert space in functional analysis',
  'Explain the mechanism of action of penicillin on bacterial cell walls',
];

export const options = {
  stages: [
    { duration: '5s', target: 3 },
    { duration: '15s', target: 3 },
    { duration: '5s', target: 0 },
  ],
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export default function () {
  const prompt = `${diversePrompts[__ITER % diversePrompts.length]} [variant ${__VU}_${__ITER}_${Date.now()}]`;
  const url = `${BASE_URL}/v1/chat/completions`;
  const payload = JSON.stringify({
    messages: [
      {
        role: 'user',
        content: prompt,
      },
    ],
    model: 'gemini-3.5-flash-lite',
  });
  const params = {
    headers: {
      'Content-Type': 'application/json',
      'X-Bypass-Exact-Cache': 'true',
    },
    timeout: '30s',
  };
  const res = http.post(url, payload, params);
  const cacheHeader = res.headers['X-Cache'] || res.headers['x-cache'];
  const isMiss = cacheHeader === 'MISS';

  check(res, {
    'status is 200': (r) => r.status === 200,
    'is cold miss': (r) => isMiss,
  });

  if (res.status === 200) {
    if (isMiss) {
      coldMisses.add(1);
    } else {
      semanticHits.add(1);
    }
    errorRate.add(0);
  } else {
    errorRate.add(1);
  }

  sleep(0.5);
}

