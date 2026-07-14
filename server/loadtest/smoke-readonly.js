import http from 'k6/http';
import { check, sleep } from 'k6';

const BASE_URL = (__ENV.BASE_URL || 'http://localhost:8080').replace(/\/+$/, '');

export const options = {
  vus: 1,
  duration: '15s',
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<500'],
    checks: ['rate>0.99'],
  },
};

export default function () {
  const healthResponse = http.get(`${BASE_URL}/health`, {
    tags: { endpoint: 'health' },
  });

  check(healthResponse, {
    'health returns 200': (response) => response.status === 200,
  });

  const rankingsResponse = http.get(`${BASE_URL}/api/rankings?stage_id=1`, {
    tags: { endpoint: 'rankings' },
  });

  check(rankingsResponse, {
    'rankings returns 200': (response) => response.status === 200,
  });

  const statsResponse = http.get(`${BASE_URL}/api/stats?stage_id=1`, {
    tags: { endpoint: 'stats' },
  });

  check(statsResponse, {
    'stats returns 200': (response) => response.status === 200,
  });

  sleep(1);
}
