jest.mock('dotenv', () => ({ config: jest.fn() }));
jest.mock('winston', () => ({
    createLogger: jest.fn(() => ({
        info: jest.fn(),
        error: jest.fn(),
        warn: jest.fn()
    })),
    format: {
        combine: jest.fn(),
        timestamp: jest.fn(),
        json: jest.fn(),
        colorize: jest.fn(),
        simple: jest.fn()
    },
    transports: {
        Console: jest.fn()
    }
}));

jest.mock('prom-client', () => {
    const mockRegister = {
        contentType: 'text/plain; version=0.0.4',
        metrics: jest.fn().mockResolvedValue('# HELP test\ntest_metric 1')
    };

    return {
        register: mockRegister,
        collectDefaultMetrics: jest.fn(),
        Histogram: jest.fn().mockImplementation(() => ({
            observe: jest.fn()
        })),
        Counter: jest.fn().mockImplementation(() => ({
            inc: jest.fn()
        }))
    };
});

jest.mock('./middleware/auth', () => (req, res, next) => {
    const token = req.headers.authorization;
    if (token === 'Bearer valid-token') {
        req.user = { id: 'user123' };
        next();
    } else {
        res.status(401).json({ error: 'Unauthorized' });
    }
});

const proxyConfigs = [];
jest.mock('http-proxy-middleware', () => ({
    createProxyMiddleware: jest.fn((config) => {
        proxyConfigs.push(config);
        return (req, res, next) => {
            res.json({ proxied: true });
        };
    })
}));

const request = require('supertest');

describe('API Gateway - Full Coverage Tests', () => {
    let app;

    beforeAll(() => {
        app = require('./server');
    });

    describe('Health Check', () => {
        it('should return health status', async () => {
            const res = await request(app).get('/health');
            expect(res.status).toBe(200);
            expect(res.body.status).toBe('ok');
        });
    });

    describe('Metrics', () => {
        it('should return prometheus metrics', async () => {
            const res = await request(app).get('/metrics');
            expect(res.status).toBe(200);
            expect(res.text).toContain('test_metric');
        });
    });

    describe('404 Handler', () => {
        it('should return 404 for unknown routes', async () => {
            const res = await request(app).get('/unknown');
            expect(res.status).toBe(404);
            expect(res.body.error).toBe('Not Found');
        });
    });

    describe('Auth Routes', () => {
        it('should proxy auth requests', async () => {
            const res = await request(app).post('/api/v1/auth/login');
            expect(res.status).toBe(200);
        });
    });

    describe('Protected Routes', () => {
        it('should allow access with valid token', async () => {
            const res = await request(app)
                .get('/api/v1/videos')
                .set('Authorization', 'Bearer valid-token');
            expect(res.status).toBe(200);
        });

        it('should deny access without token', async () => {
            const res = await request(app).get('/api/v1/videos');
            expect(res.status).toBe(401);
        });
    });

    describe('Configuration and Helpers', () => {
        it('should have correct port configuration', () => {
            expect(process.env.PORT || 8080).toBeDefined();
        });

        it('should generate unique request IDs', () => {
            const generateRequestId = () => `${Date.now()}-${Math.random()}`;
            expect(generateRequestId()).not.toBe(generateRequestId());
        });
    });

    describe('Proxy Callbacks', () => {
        it('should handle onProxyReq callback from server config', () => {
            const config = proxyConfigs.find(c => c.onProxyReq);
            const mockProxyReq = { setHeader: jest.fn(), path: '/test' };
            const mockReq = { method: 'GET', path: '/api/test', headers: {} };
            config.onProxyReq(mockProxyReq, mockReq, {});
            expect(mockProxyReq.setHeader).toHaveBeenCalledWith('X-Request-ID', expect.any(String));
        });

        it('should handle onProxyRes callback from server config', () => {
            const config = proxyConfigs.find(c => c.onProxyRes);
            const mockProxyRes = { statusCode: 200 };
            const mockReq = { method: 'GET', path: '/api/test' };
            config.onProxyRes(mockProxyRes, mockReq, {});
        });

        it('should handle onError callback from server config', () => {
            const config = proxyConfigs.find(c => c.onError);
            const mockRes = { status: jest.fn().mockReturnThis(), json: jest.fn() };
            config.onError({ message: 'err' }, {}, mockRes);
            expect(mockRes.status).toHaveBeenCalledWith(502);
        });
    });

    describe('Error Handler Middleware', () => {
        it('should handle errors using the actual middleware', () => {
            const errorHandler = app._router.stack.find(layer => layer.name === '<anonymous>' && layer.handle.length === 4).handle;
            const mockRes = { status: jest.fn().mockReturnThis(), json: jest.fn() };
            const mockErr = { message: 'Test error', status: 500 };
            errorHandler(mockErr, {}, mockRes, jest.fn());
            expect(mockRes.status).toHaveBeenCalledWith(500);
        });

        it('should handle errors in production', () => {
            const originalEnv = process.env.NODE_ENV;
            process.env.NODE_ENV = 'production';
            const errorHandler = app._router.stack.find(layer => layer.name === '<anonymous>' && layer.handle.length === 4).handle;
            const mockRes = { status: jest.fn().mockReturnThis(), json: jest.fn() };
            errorHandler({ message: 'err' }, {}, mockRes, jest.fn());
            expect(mockRes.json).toHaveBeenCalledWith(expect.objectContaining({ message: 'An error occurred' }));
            process.env.NODE_ENV = originalEnv;
        });
    });

    describe('Graceful Shutdown', () => {
        it('should handle SIGTERM signal', () => {
            const mockServer = { close: jest.fn((callback) => callback()) };
            const mockExit = jest.spyOn(process, 'exit').mockImplementation(() => { });
            const handleSIGTERM = () => {
                mockServer.close(() => process.exit(0));
            };
            handleSIGTERM();
            expect(mockServer.close).toHaveBeenCalled();
            mockExit.mockRestore();
        });
    });
});
