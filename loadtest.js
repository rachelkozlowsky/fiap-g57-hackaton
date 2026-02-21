import http from 'k6/http';
import { check, sleep } from 'k6';
export const options = {
  stages: [
    { duration: '30s', target: 50 },  // Ramp-up rápido
    { duration: '2m', target: 200 },  // Carga pesada
    { duration: '1m', target: 0 },    // Ramp-down
  ],
  thresholds: {
    http_req_duration: ['p(95)<2000'],
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
  // 1. Registro
  const registerRes = http.post('http://api-gateway:8080/api/v1/auth/register', payload, { headers: headers });
  // Se der sucesso ou se ja existir (409 conflict), tenta login
  if (registerRes.status === 201 || registerRes.status === 409) {
    // 2. Login (Gera carga CPU bcrypt)
    const loginRes = http.post('http://api-gateway:8080/api/v1/auth/login', payload, { headers: headers });
    check(loginRes, { 'login status 200': (r) => r.status === 200 });
  }
  sleep(0.1); // Sleep curto para maximizar RPS
}
