const authMiddleware = require('./auth');
const jwt = require('jsonwebtoken');

jest.mock('jsonwebtoken');

describe('Auth Middleware', () => {
    let req, res, next;

    beforeEach(() => {
        req = {
            headers: {}
        };
        res = {
            status: jest.fn().mockReturnThis(),
            json: jest.fn().mockReturnThis()
        };
        next = jest.fn();
        jest.clearAllMocks();
    });

    it('should return 401 if no authorization header is present', () => {
        authMiddleware(req, res, next);
        expect(res.status).toHaveBeenCalledWith(401);
        expect(res.json).toHaveBeenCalledWith(expect.objectContaining({ error: 'Unauthorized' }));
    });

    it('should return 401 if authorization format is invalid', () => {
        req.headers.authorization = 'InvalidFormat token';
        authMiddleware(req, res, next);
        expect(res.status).toHaveBeenCalledWith(401);
    });

    it('should call next and set user if token is valid', () => {
        req.headers.authorization = 'Bearer valid-token';
        const decoded = { user_id: 'user123', email: 'test@example.com', role: 'admin' };
        jwt.verify.mockReturnValue(decoded);

        authMiddleware(req, res, next);

        expect(jwt.verify).toHaveBeenCalledWith('valid-token', expect.any(String));
        expect(req.user.id).toBe('user123');
        expect(req.headers['x-user-id']).toBe('user123');
        expect(next).toHaveBeenCalled();
    });

    it('should return 401 if token is expired', () => {
        req.headers.authorization = 'Bearer expired-token';
        const error = new Error('Expired');
        error.name = 'TokenExpiredError';
        jwt.verify.mockImplementation(() => { throw error; });

        authMiddleware(req, res, next);

        expect(res.status).toHaveBeenCalledWith(401);
        expect(res.json).toHaveBeenCalledWith(expect.objectContaining({ message: 'Token has expired' }));
    });

    it('should return 401 if token is invalid', () => {
        req.headers.authorization = 'Bearer invalid-token';
        const error = new Error('Invalid');
        error.name = 'JsonWebTokenError';
        jwt.verify.mockImplementation(() => { throw error; });

        authMiddleware(req, res, next);

        expect(res.status).toHaveBeenCalledWith(401);
        expect(res.json).toHaveBeenCalledWith(expect.objectContaining({ message: 'Invalid token' }));
    });

    it('should return 500 if an unknown error occurs', () => {
        req.headers.authorization = 'Bearer token';
        jwt.verify.mockImplementation(() => { throw new Error('Unknown'); });

        authMiddleware(req, res, next);

        expect(res.status).toHaveBeenCalledWith(500);
    });
});
