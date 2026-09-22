import http from 'k6/http';
import { check } from 'k6';
import { Counter, Trend } from 'k6/metrics';

const rateLimited = new Counter('rate_limited_requests');
const allowed = new Counter('allowed_requests');
const serverErrors = new Counter('server_errors');

export const options = {
  vus: 50,
  duration: '10s',
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export default function () {
  const url = `${BASE_URL}/v1/chat/completions`;
  const payload = JSON.stringify({
    messages: [
      {
        role: 'user',
        content: 'ping',
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
  if (res.status === 429) {
    rateLimited.add(1);
  } else if (res.status === 200) {
    allowed.add(1);
  } else if (res.status >= 500) {
    serverErrors.add(1);
  }
}
