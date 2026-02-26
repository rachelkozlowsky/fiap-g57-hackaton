require('dotenv').config();
const express = require('express');
const cors = require('cors');
const helmet = require('helmet');
const morgan = require('morgan');
const rateLimit = require('express-rate-limit');
const { createProxyMiddleware } = require('http-proxy-middleware');
const winston = require('winston');

const logger = winston.createLogger({
  level: 'info',
  format: winston.format.combine(
    winston.format.timestamp(),
    winston.format.json()
  ),
  transports: [
    new winston.transports.Console({
      format: winston.format.combine(
        winston.format.colorize(),
        winston.format.simple()
      )
    })
  ]
});

const app = express();
const PORT = process.env.PORT || 8080;

app.use(helmet());
app.use(cors());
app.use(morgan('combined', { stream: { write: message => logger.info(message.trim()) } }));


let limiter;
if (!process.env.DISABLE_RATE_LIMIT && process.env.NODE_ENV !== 'loadtest') {
  limiter = rateLimit({
    windowMs: parseInt(process.env.RATE_LIMIT_WINDOW_MS) || 60000,
    max: parseInt(process.env.RATE_LIMIT_MAX_REQUESTS) || 100,
    message: 'Too many requests from this IP, please try again later.',
    standardHeaders: true,
    legacyHeaders: false,
  });
  app.use(limiter);
} else {
  logger.warn('rate limiter disabled (load test or DISABLE_RATE_LIMIT)');
}

app.get('/health', (req, res) => {
  res.json({
    status: 'ok',
    service: 'api-gateway',
    version: '1.0.0',
    timestamp: Date.now()
  });
});

app.get('/health/live', (req, res) => {
  res.json({ status: 'up' });
});

app.get('/health/ready', (req, res) => {
  res.json({ status: 'ready' });
});

const client = require('prom-client');
const collectDefaultMetrics = client.collectDefaultMetrics;
collectDefaultMetrics({ register: client.register });

const httpRequestDurationMicroseconds = new client.Histogram({
  name: 'http_request_duration_seconds',
  help: 'Duration of HTTP requests in seconds',
  labelNames: ['method', 'route', 'code'],
  buckets: [0.1, 0.3, 0.5, 0.7, 1, 3, 5, 7, 10]
});

const httpRequestsTotal = new client.Counter({
  name: 'http_requests_total',
  help: 'Total number of HTTP requests',
  labelNames: ['method', 'route', 'code']
});

app.get('/metrics', async (req, res) => {
  res.set('Content-Type', client.register.contentType);
  res.send(await client.register.metrics());
});

app.use((req, res, next) => {
  if (req.path === '/metrics') return next();

  const start = process.hrtime();
  res.on('finish', () => {
    const durationCount = process.hrtime(start);
    const durationSeconds = (durationCount[0] * 1e9 + durationCount[1]) / 1e9;
    const route = req.path;

    httpRequestsTotal.inc({
      method: req.method,
      route: route,
      code: res.statusCode
    });

    httpRequestDurationMicroseconds.observe({
      method: req.method,
      route: route,
      code: res.statusCode
    }, durationSeconds);
  });
  next();
});

const authMiddleware = require('./middleware/auth');

const AUTH_SERVICE_URL = process.env.AUTH_SERVICE_URL || 'http://localhost:8081';
const VIDEO_SERVICE_URL = process.env.VIDEO_SERVICE_URL || 'http://localhost:8082';
const STATUS_SERVICE_URL = process.env.STATUS_SERVICE_URL || 'http://localhost:8083';

const proxyOptions = {
  timeout: 10000,          // 10 s
  proxyTimeout: 10000,
  changeOrigin: true,
  logLevel: 'debug',  timeout: parseInt(process.env.PROXY_TIMEOUT_MS) || 10000,      // espera 10s por padrão
  proxyTimeout: parseInt(process.env.PROXY_TIMEOUT_MS) || 10000,  onProxyReq: (proxyReq, req, res) => {

    const requestId = req.headers['x-request-id'] || generateRequestId();
    proxyReq.setHeader('X-Request-ID', requestId);
    logger.info(`Proxying request: ${req.method} ${req.path} -> ${proxyReq.path}`);
  },
  onProxyRes: (proxyRes, req, res) => {
    logger.info(`Received response: ${proxyRes.statusCode} for ${req.method} ${req.path}`);
  },
  onError: (err, req, res) => {
    logger.error(`Proxy error: ${err.message}`);
    res.status(502).json({
      error: 'Bad Gateway',
      message: 'Service temporarily unavailable'
    });
  }
};

app.use('/api/v1/auth', createProxyMiddleware({
  target: AUTH_SERVICE_URL,
  ...proxyOptions
}));

app.use('/api/v1/videos/:id/download', createProxyMiddleware({
  target: VIDEO_SERVICE_URL,
  ...proxyOptions
}));

app.use('/api/v1/videos', authMiddleware, createProxyMiddleware({
  target: VIDEO_SERVICE_URL,
  ...proxyOptions
}));

app.use('/api/v1/videos/status', authMiddleware, createProxyMiddleware({
  target: STATUS_SERVICE_URL,
  ...proxyOptions
}));

app.use('/api/v1/stats', authMiddleware, createProxyMiddleware({
  target: STATUS_SERVICE_URL,
  ...proxyOptions
}));

app.use((req, res) => {
  res.status(404).json({
    error: 'Not Found',
    message: 'The requested endpoint does not exist'
  });
});

app.use((err, req, res, next) => {
  logger.error(`Error: ${err.message}`, { stack: err.stack });
  res.status(err.status || 500).json({
    error: 'Internal Server Error',
    message: process.env.NODE_ENV === 'production' ? 'An error occurred' : err.message
  });
});

let server;

if (require.main === module) {
  server = app.listen(PORT, () => {
    logger.info(`API Gateway listening on port ${PORT}`);
    logger.info(`Environment: ${process.env.NODE_ENV || 'development'}`);
    logger.info(`Auth Service: ${AUTH_SERVICE_URL}`);
    logger.info(`Video Service: ${VIDEO_SERVICE_URL}`);
    logger.info(`Status Service: ${STATUS_SERVICE_URL}`);
  });
}

process.on('SIGTERM', () => {
  logger.info('SIGTERM signal received: closing HTTP server');
  if (server) {
    server.close(() => {
      logger.info('HTTP server closed');
      process.exit(0);
    });
  } else {
    process.exit(0);
  }
});

function generateRequestId() {
  return `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
}

module.exports = app;
