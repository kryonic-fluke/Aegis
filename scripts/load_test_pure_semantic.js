import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Trend } from 'k6/metrics';

const semanticHits = new Counter('semantic_hits');
const embeddingTrend = new Trend('embedding_duration_ms');
const vectorTrend = new Trend('pgvector_duration_ms');

export const options = {
  stages: [
    { duration: '5s', target: 5 },
    { duration: '15s', target: 5 },
    { duration: '5s', target: 0 },
  ],
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export function setup() {
  const url = `${BASE_URL}/v1/chat/completions`;
  const payload = JSON.stringify({
    messages: [
      {
        role: 'user',
        content: 'Explain general relativity simply',
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
        content: "Explain Einstein's relativity theory in easy language",
      },
    ],
    model: 'gemini-3.5-flash-lite',
  });
  const params = {
    headers: {
      'Content-Type': 'application/json',
      'X-Bypass-Exact-Cache': 'true',
    },
  };
  const res = http.post(url, payload, params);
  const isSemanticHit = res.headers['X-Cache'] === 'SEMANTIC_HIT' || res.headers['x-cache'] === 'SEMANTIC_HIT';
  check(res, {
    'status is 200': (r) => r.status === 200,
    'is semantic hit': (r) => isSemanticHit,
  });

  if (isSemanticHit) {
    semanticHits.add(1);
    const embHdr = res.headers['X-Embedding-Duration-Ms'] || res.headers['x-embedding-duration-ms'];
    const vecHdr = res.headers['X-Vector-Duration-Ms'] || res.headers['x-vector-duration-ms'];
    if (embHdr) {
      embeddingTrend.add(parseFloat(embHdr));
    }
    if (vecHdr) {
      vectorTrend.add(parseFloat(vecHdr));
    }
  }
  sleep(0.5);
}

