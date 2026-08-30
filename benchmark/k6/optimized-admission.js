import http from 'k6/http';
import { check, sleep } from 'k6';

const baseURL = __ENV.BASE_URL || 'http://localhost:8081';
const activityID = __ENV.ACTIVITY_ID || '1';
const vus = Number(__ENV.K6_VUS || 100);
const duration = __ENV.K6_DURATION || '60s';
const password = __ENV.K6_PASSWORD || 'optimized-load-password';

export const options = { scenarios: { admission: { executor: 'constant-vus', vus, duration, gracefulStop: '10s' } }, thresholds: { http_req_failed: ['rate<0.20'], http_req_duration: ['p(95)<1000'] } };
let token = '';
function login() {
  if (token) return token;
  const username = `opt_load_${__VU}_${Date.now()}`;
  const h = { headers: { 'Content-Type': 'application/json' } };
  const reg = http.post(`${baseURL}/api/auth/register`, JSON.stringify({ username, password }), h);
  check(reg, { 'register accepted': r => [201, 409].includes(r.status) });
  const res = http.post(`${baseURL}/api/auth/login`, JSON.stringify({ username, password }), h);
  check(res, { 'login succeeded': r => r.status === 200 });
  if (res.status === 200) token = res.json('token');
  return token;
}
export default function () {
  const jwt = login(); if (!jwt) return;
  const requestID = `k6-opt-${__VU}-${__ITER}-${Date.now()}-${Math.random()}`;
  const res = http.post(`${baseURL}/api/seckill/activities/${activityID}/requests`, JSON.stringify({ request_id: requestID }), { headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${jwt}` }, tags: { scenario: 'optimized_admission' } });
  check(res, { 'admission response': r => [202, 409, 503].includes(r.status) });
  sleep(0.1);
}
