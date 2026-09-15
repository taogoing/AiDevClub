import http from 'k6/http';
import { check } from 'k6';
import { Counter, Rate } from 'k6/metrics';

const baseURL = __ENV.BASE_URL || 'http://localhost:8080';
const mode = __ENV.MODE || 'local';
// `page_size=5` is the source implementation's in-process Top-5 snapshot;
// any other page size bypasses that snapshot and reads the Redis ZSet.
const pageSize = mode === 'local' ? 5 : 6;

export const options = {
  vus: Number(__ENV.VUS || 20),
  duration: __ENV.DURATION || '7s',
  discardResponseBodies: true,
  thresholds: {
    http_req_failed: ['rate<0.01'],
    checks: ['rate>0.99'],
  },
};

const successfulRankingRequests = new Counter('successful_ranking_requests');
const rankingCheckFailure = new Rate('ranking_check_failure');

export function setup() {
  // Warm the target server's Top-5 snapshot before measuring local-cache mode.
  const response = http.get(`${baseURL}/api/v1/articles/ranking?page=1&page_size=${pageSize}`);
  if (response.status !== 200) {
    throw new Error(`warm-up failed: HTTP ${response.status}`);
  }
}

export default function () {
  const response = http.get(`${baseURL}/api/v1/articles/ranking?page=1&page_size=${pageSize}`);
  const ok = check(response, { 'ranking returns HTTP 200': (r) => r.status === 200 });
  rankingCheckFailure.add(!ok);
  if (ok) successfulRankingRequests.add(1);
}
