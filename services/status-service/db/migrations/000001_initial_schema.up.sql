-- Status Service Database Schema

-- Status cache table - for caching aggregated status information
-- This service primarily reads from Video Service via HTTP and caches results
CREATE TABLE status_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    cache_key VARCHAR(255) NOT NULL UNIQUE,
    cache_data JSONB NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_status_cache_user_id ON status_cache(user_id);
CREATE INDEX idx_status_cache_key ON status_cache(cache_key);
CREATE INDEX idx_status_cache_expires ON status_cache(expires_at);

-- Query logs for analytics
CREATE TABLE query_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    query_type VARCHAR(100) NOT NULL,
    response_time_ms INT,
    cache_hit BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_query_logs_user_id ON query_logs(user_id);
CREATE INDEX idx_query_logs_created_at ON query_logs(created_at DESC);

-- Trigger to update updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_status_cache_updated_at BEFORE UPDATE ON status_cache
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Cleanup function for expired cache
CREATE OR REPLACE FUNCTION cleanup_expired_cache()
RETURNS void AS $$
BEGIN
    DELETE FROM status_cache WHERE expires_at < NOW();
    DELETE FROM query_logs WHERE created_at < NOW() - INTERVAL '7 days';
    RAISE NOTICE 'Status cache cleanup completed';
END;
$$ LANGUAGE plpgsql;

-- View for cache statistics
CREATE OR REPLACE VIEW cache_stats AS
SELECT 
    COUNT(*) AS total_entries,
    COUNT(CASE WHEN expires_at > NOW() THEN 1 END) AS valid_entries,
    COUNT(CASE WHEN expires_at <= NOW() THEN 1 END) AS expired_entries,
    AVG(EXTRACT(EPOCH FROM (expires_at - created_at))) AS avg_ttl_seconds
FROM status_cache;

COMMENT ON TABLE status_cache IS 'Caches status data from Video Service';
COMMENT ON TABLE query_logs IS 'Logs status queries for analytics';


