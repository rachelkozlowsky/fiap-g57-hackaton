const jwt = require('jsonwebtoken');

const JWT_SECRET = process.env.JWT_SECRET || 'your-super-secret-jwt-key-change-in-production';

const authMiddleware = (req, res, next) => {
    try {
        const authHeader = req.headers.authorization;

        if (!authHeader) {
            return res.status(401).json({
                error: 'Unauthorized',
                message: 'No authorization token provided'
            });
        }

        const parts = authHeader.split(' ');

        if (parts.length !== 2 || parts[0] !== 'Bearer') {
            return res.status(401).json({
                error: 'Unauthorized',
                message: 'Invalid authorization format. Use: Bearer <token>'
            });
        }

        const token = parts[1];

        const decoded = jwt.verify(token, JWT_SECRET);

        req.user = {
            id: decoded.user_id || decoded.sub,
            email: decoded.email,
            role: decoded.role || 'user'
        };

        req.headers['x-user-id'] = req.user.id;
        req.headers['x-user-role'] = req.user.role;

        next();
    } catch (error) {
        if (error.name === 'TokenExpiredError') {
            return res.status(401).json({
                error: 'Unauthorized',
                message: 'Token has expired'
            });
        }

        if (error.name === 'JsonWebTokenError') {
            return res.status(401).json({
                error: 'Unauthorized',
                message: 'Invalid token'
            });
        }

        return res.status(500).json({
            error: 'Internal Server Error',
            message: 'Error validating token'
        });
    }
};

module.exports = authMiddleware;
