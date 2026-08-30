import http from 'k6/http';
import { check } from 'k6';
const baseURL = __ENV.BASE_URL || 'http://localhost:8081'; const activityID = __ENV.ACTIVITY_ID || '1'; const password = __ENV.K6_PASSWORD || 'optimized-load-password';
export const options = { vus: Number(__ENV.K6_VUS || 50), duration: __ENV.K6_DURATION || '30s' };
let token = ''; let requestID = '';
export default function () {
  if (!token) { const username = `opt_dup_${__VU}`; const h = { headers: { 'Content-Type': 'application/json' } }; http.post(`${baseURL}/api/auth/register`, JSON.stringify({ username, password }), h); const l = http.post(`${baseURL}/api/auth/login`, JSON.stringify({ username, password }), h); if (l.status !== 200) return; token = l.json('token'); requestID = `k6-duplicate-${__VU}`; }
  const r = http.post(`${baseURL}/api/seckill/activities/${activityID}/requests`, JSON.stringify({ request_id: requestID }), { headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` }, tags: { scenario: 'optimized_duplicate' } });
  check(r, { 'idempotent response': x => [202, 409].includes(x.status) });
}
