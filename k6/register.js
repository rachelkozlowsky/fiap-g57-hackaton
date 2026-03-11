import http from 'k6/http';
import { check, sleep } from 'k6';
import { htmlReport } from 'https://raw.githubusercontent.com/benc-uk/k6-reporter/main/dist/bundle.js';
import { textSummary } from 'https://jslib.k6.io/k6-summary/0.0.1/index.js';
import { Counter, Trend } from 'k6/metrics';
import exec from 'k6/execution';

const BASE_URL = __ENV.K6_BASE_URL || 'http://localhost:8080';

const reqDuration = new Trend('http_req_duration', true);
const reqWaiting = new Trend('http_req_waiting', true);
const reqConnecting = new Trend('http_req_connecting', true);
const reqReceiving = new Trend('http_req_receiving', true);
const reqSending = new Trend('http_req_sending', true);
const reqBlocked = new Trend('http_req_blocked', true);
const reqTlsHandshaking = new Trend('http_req_tls_handshaking', true);
const httpReqsCounters = {};
const httpStatusCounters = {};

const tags = ['30vus', '40vus', '60vus', '100vus'];
const time = 10;
const tipo = 's';
const duration = `${time}${tipo}`;

tags.forEach(tag => {
  httpReqsCounters[tag] = new Counter(`http_requests_${tag}`);
  httpStatusCounters[tag] = new Counter(`http_errors_${tag}`);
});

export const options = {
  scenarios: tags.reduce((acc, tag, index) => {
    acc[`scenario_${tag}`] = {
      executor: 'constant-vus',
      vus: parseInt(tag),
      duration: duration,
      startTime: `${index * time}${tipo}`,
      tags: { scenario: tag },
    };
    return acc;
  }, {}),
  thresholds: tags.reduce((acc, tag) => {
    acc[`http_req_failed{scenario:${tag}}`] = [{ threshold: 'rate<0.01', abortOnFail: false }];
    acc[`checks{scenario:${tag}}`] = [{ threshold: 'rate>0.99', abortOnFail: false }];
    acc[`http_req_duration{scenario:${tag}}`] = [{ threshold: 'avg<2000', abortOnFail: false }];
    acc[`http_req_waiting{scenario:${tag}}`] = [{ threshold: 'avg<1500', abortOnFail: false }];
    acc[`http_req_connecting{scenario:${tag}}`] = [{ threshold: 'avg<500', abortOnFail: false }];
    acc[`http_req_sending{scenario:${tag}}`] = [{ threshold: 'avg<500', abortOnFail: false }];
    acc[`http_req_receiving{scenario:${tag}}`] = [{ threshold: 'avg<500', abortOnFail: false }];
    acc[`http_req_blocked{scenario:${tag}}`] = [{ threshold: 'avg<500', abortOnFail: false }];
    acc[`http_req_tls_handshaking{scenario:${tag}}`] = [{ threshold: 'avg<500', abortOnFail: false }];
    return acc;
  }, {}),

  summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(95)', 'p(99)'],
};


export default function () {
  const scenarioName = exec.scenario.name;
  const tag = scenarioName.replace('scenario_', '');

  const uniqueId = `${__VU}-${__ITER}-${Date.now()}`;
  const email = `user_${uniqueId}@test.com`;
  const password = 'password123';
  const payload = JSON.stringify({
    email: email,
    name: `User ${uniqueId}`,
    password: password,
  });

  const res = http.post(`${BASE_URL}/api/v1/auth/register`,
    payload,
    {
      headers: { 'Content-Type': 'application/json' },
      tags: { scenario: tag },
    }
  );

  reqDuration.add(res.timings.duration, { scenario: tag });
  reqWaiting.add(res.timings.waiting, { scenario: tag });
  reqConnecting.add(res.timings.connecting, { scenario: tag });
  reqReceiving.add(res.timings.receiving, { scenario: tag });
  reqSending.add(res.timings.sending, { scenario: tag });
  reqBlocked.add(res.timings.blocked, { scenario: tag });
  reqTlsHandshaking.add(res.timings.tls_handshaking, { scenario: tag });

  httpReqsCounters[tag].add(1);
  if (res.status !== 200 && res.status !== 201) {
    httpStatusCounters[tag].add(1);
  }


  check(res, {
    [`status 200 (${tag})`]: (r) => r.status === 201,
  }, { scenario: tag });
}

export function handleSummary(data) {
  const metricsToRemove = [
    'http_req_blocked',
    'http_req_connecting',
    'http_req_duration',
    'http_req_receiving',
    'http_req_sending',
    'http_req_tls_handshaking',
    'http_req_waiting',
    'http_req_failed',
    'checks'
  ];

  metricsToRemove.forEach(metric => {
    if (data.metrics[metric]) {
      delete data.metrics[metric];
    }
  });

  return {
    'results/register.html': htmlReport(data),
    stdout: textSummary(data, { indent: ' ', enableColors: true }),
  };
}
