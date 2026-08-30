import http from 'k6/http';
import { check, sleep } from 'k6';
const baseURL = __ENV.BASE_URL || 'http://localhost:8081';
const requestID = __ENV.REQUEST_ID || '';
export const options = { vus: Number(__ENV.K6_VUS || 50), duration: __ENV.K6_DURATION || '60s', thresholds: { http_req_failed: ['rate<0.10'] } };
export default function () {
  if (requestID) { const r = http.get(`${baseURL}/api/seckill/requests/${requestID}`, { tags: { scenario: 'optimized_status_read' } }); check(r, { 'status readable': x => [200, 404].includes(x.status) }); }
  const a = http.get(`${baseURL}/api/seckill/activities/${__ENV.ACTIVITY_ID || '1'}`, { tags: { scenario: 'optimized_activity_read' } }); check(a, { 'activity readable': x => x.status === 200 }); sleep(0.1);
}
