import http from 'k6/http';
import { check } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';

const baseURL = __ENV.BASE_URL || 'http://localhost:8080';
const action = __ENV.ACTION || 'like';
const data = JSON.parse(open(__ENV.DATA_FILE));

export const options = {
  scenarios: {
    notification_write: {
      executor: 'constant-vus',
      vus: Number(__ENV.VUS || 20),
      duration: __ENV.DURATION || '30s',
      gracefulStop: '5s',
    },
  },
  summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(90)', 'p(95)', 'p(99)'],
  discardResponseBodies: true,
};

const benchmarkRequests = new Counter('benchmark_requests');
const benchmarkErrors = new Counter('benchmark_errors');
const benchmarkErrorRate = new Rate('benchmark_error_rate');
const benchmarkDuration = new Trend('benchmark_request_duration', true);

export function setup() {
  if (!['like', 'favorite', 'comment'].includes(action)) {
    throw new Error(`unsupported ACTION: ${action}`);
  }
  if (!data.article_id || !data.tokens?.length) {
    throw new Error('DATA_FILE must contain article_id and tokens');
  }
  return data;
}

export default function (testData) {
  const token = testData.tokens[(__VU - 1) % testData.tokens.length];
  const params = {
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    tags: { action },
  };
  let response;
  if (action === 'like') {
    response = http.post(`${baseURL}/api/v1/articles/${testData.article_id}/like`, null, params);
  } else if (action === 'favorite') {
    response = http.post(`${baseURL}/api/v1/articles/${testData.article_id}/favorite`, null, params);
  } else {
    const content = `notification k6 comment vu=${__VU} iteration=${__ITER} time=${Date.now()}`;
    response = http.post(
      `${baseURL}/api/v1/articles/${testData.article_id}/comments`,
      JSON.stringify({ content }),
      params,
    );
  }

  const ok = check(response, { [`${action} endpoint returns HTTP 200`]: (r) => r.status === 200 });
  benchmarkRequests.add(1);
  benchmarkDuration.add(response.timings.duration);
  benchmarkErrorRate.add(!ok);
  if (!ok) benchmarkErrors.add(1);
}
