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

const uploadTrend = new Trend('video_upload_duration', true);
const processingTrend = new Trend('video_processing_total_time', true);
const pollCounts = {};

const tags = ['5vus', '10vus', '20vus', '50vus', '80vus', '150vus'];
const time = 10;
const tipo = 's';
const duration = `${time}${tipo}`;

tags.forEach(tag => {
    pollCounts[tag] = new Counter(`status_poll_count_${tag}`);
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
        acc[`http_req_failed{scenario:${tag}}`] = [{ threshold: 'rate<0.15', abortOnFail: false }];
        acc[`http_req_duration{scenario:${tag}}`] = [{ threshold: 'avg<10000', abortOnFail: false }];
        acc[`http_req_waiting{scenario:${tag}}`] = [{ threshold: 'avg<5000', abortOnFail: false }];
        acc[`http_req_connecting{scenario:${tag}}`] = [{ threshold: 'avg<1000', abortOnFail: false }];
        acc[`http_req_sending{scenario:${tag}}`] = [{ threshold: 'avg<3000', abortOnFail: false }];
        acc[`http_req_receiving{scenario:${tag}}`] = [{ threshold: 'avg<1000', abortOnFail: false }];
        acc[`http_req_blocked{scenario:${tag}}`] = [{ threshold: 'avg<1000', abortOnFail: false }];
        acc[`http_req_tls_handshaking{scenario:${tag}}`] = [{ threshold: 'avg<1000', abortOnFail: false }];
        acc[`video_upload_duration{scenario:${tag}}`] = [{ threshold: 'avg<20000', abortOnFail: false }];
        acc[`video_processing_total_time{scenario:${tag}}`] = [{ threshold: 'avg<120000', abortOnFail: false }];
        return acc;
    }, {}),

    summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(95)', 'p(99)'],
};



const binFile = open('./test_video.mp4', 'b');

export default function () {
    const scenarioName = exec.scenario.name;
    const tag = scenarioName.replace('scenario_', '');

    const uniqueId = `${exec.vu.idInTest}-${exec.vu.iterationInInstance}-${Date.now()}`;
    const email = `proc_test_${uniqueId}@test.com`;
    const password = 'password123';

    const authHeaders = { 'Content-Type': 'application/json' };

    let token;
    const registerRes = http.post(`${BASE_URL}/api/v1/auth/register`, JSON.stringify({
        email: email,
        name: `Processing Tester ${uniqueId}`,
        password: password,
    }), {
        headers: authHeaders,
        tags: { scenario: tag }
    });
    addMetrics(registerRes, tag);

    if (registerRes.status === 201) {
        token = registerRes.json('access_token');
    } else if (registerRes.status === 409) {
        const loginRes = http.post(`${BASE_URL}/api/v1/auth/login`, JSON.stringify({
            email: email,
            password: password,
        }), {
            headers: authHeaders,
            tags: { scenario: tag }
        });
        addMetrics(loginRes, tag);

        if (loginRes.status === 200) {
            token = loginRes.json('access_token');
        } else {
            console.error(`Login failed for existing user ${email}. Status: ${loginRes.status}, Body: ${loginRes.body}`);
        }
    } else {
        console.error(`Register failed for ${email}. Status: ${registerRes.status}, Body: ${registerRes.body}`);
    }

    if (!token) {
        console.error(`Failed to obtain token for VU ${exec.vu.idInTest}`);
        return;
    }

    const headers = { 'Authorization': `Bearer ${token}` };

    const uploadData = {
        video: http.file(binFile, 'test_video.mp4', 'video/mp4'),
    };

    const uploadStartTime = Date.now();
    const uploadRes = http.post(`${BASE_URL}/api/v1/videos/upload`, uploadData, {
        headers: { 'Authorization': `Bearer ${token}` },
        tags: { scenario: tag }
    });
    addMetrics(uploadRes, tag);

    const uploadDuration = Date.now() - uploadStartTime;
    uploadTrend.add(uploadDuration, { scenario: tag });

    const isUploadOk = check(uploadRes, {
        'upload status 200': (r) => r.status === 200,
        'has video_id': (r) => r.json('video_id') !== undefined,
    }, { scenario: tag });

    if (!isUploadOk) {
        return;
    }

    const videoId = uploadRes.json('video_id');
    const processStartTime = Date.now();

    let status = 'queued';
    let attempts = 0;
    const maxAttempts = 180;

    while ((status === 'queued' || status === 'processing' || status === 'pending') && attempts < maxAttempts) {
        attempts++;
        pollCounts[tag].add(1);

        sleep(1);

        const statusRes = http.get(`${BASE_URL}/api/v1/videos/${videoId}`, {
            headers: headers,
            tags: { scenario: tag }
        });
        addMetrics(statusRes, tag);

        check(statusRes, {
            'status fetch 200': (r) => r.status === 200,
        }, { scenario: tag });

        if (statusRes.status === 200) {
            status = statusRes.json('status');
            if (status === 'completed' || status === 'failed') {
                break;
            }
        }
    }

    function addMetrics(res, tag) {
        reqDuration.add(res.timings.duration, { scenario: tag });
        reqWaiting.add(res.timings.waiting, { scenario: tag });
        reqConnecting.add(res.timings.connecting, { scenario: tag });
        reqReceiving.add(res.timings.receiving, { scenario: tag });
        reqSending.add(res.timings.sending, { scenario: tag });
        reqBlocked.add(res.timings.blocked, { scenario: tag });
        reqTlsHandshaking.add(res.timings.tls_handshaking, { scenario: tag });
    }


    if (status === 'completed' || status === 'failed') {
        const totalProcessingTime = Date.now() - processStartTime;
        processingTrend.add(totalProcessingTime, { scenario: tag, status: status });
        check(uploadRes, {
            'processing success': () => status === 'completed',
        }, { scenario: tag });
    } else {
        check(uploadRes, {
            'processing timeout': () => false,
        }, { scenario: tag });
    }

    sleep(2);
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
        'video_processing_total_time',
        'video_upload_duration',
        'http_req_failed',
        'checks',
        'iteration_duration',
        'iteration_duration'
    ];

    metricsToRemove.forEach(metric => {
        if (data.metrics[metric]) {
            delete data.metrics[metric];
        }
    });

    return {
        'results/processing.html': htmlReport(data),
        stdout: textSummary(data, { indent: ' ', enableColors: true }),
    };
}

