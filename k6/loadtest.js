import http from 'k6/http';
import { check, sleep } from 'k6';
import { htmlReport } from 'https://raw.githubusercontent.com/benc-uk/k6-reporter/main/dist/bundle.js';

const BASE_URL = __ENV.K6_BASE_URL || 'http://host.docker.internal:8080';


function makeStages() {
  const stages = [];
  stages.push({ duration: '3s', target: 30 });
  stages.push({ duration: '3s', target: 20 });
  for (let v = 20; v <= 2; v += 1) {
    stages.push({ duration: '3s', target: v });
  }
  stages.push({ duration: '3s', target: 0 });
  return stages;
}

export const options = {
  scenarios: {
    dynamic: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: makeStages(),
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<2000'],
    'http_req_failed': ['rate<0.05'], 
  },
};

export default function () {
  const uniqueId = `${__VU}-${__ITER}-${Date.now()}`;
  const email = `user_${uniqueId}@test.com`;
  const password = 'password123';
  const payload = JSON.stringify({
    email: email,
    password: password,
  });
  const headers = { 'Content-Type': 'application/json' };
  const registerRes = http.post(`${BASE_URL}/api/v1/auth/register`, payload, { headers: headers });
  if (registerRes.status === 201 || registerRes.status === 409) {
    const loginRes = http.post(`${BASE_URL}/api/v1/auth/login`, payload, { headers: headers });
    check(loginRes, { 'login status 200': (r) => r.status === 200 });
  }
  sleep(0.1); 
}

export function handleSummary(data) {
    return saidaHtml(data);
}

export function saidaHtml(data) {
    return {
        [loadtest-report.html]: htmlReport(data),
        stdout: textSummary(data, { indent: " ", enableColors: true }),
    };
}


