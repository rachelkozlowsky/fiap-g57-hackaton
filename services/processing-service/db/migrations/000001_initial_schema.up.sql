-- Processing Service Database Schema

-- Processing jobs table - Core entity for Processing Service
-- Note: video_id references Video Service (different DB, logical reference only)
CREATE TABLE processing_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    video_id UUID NOT NULL, -- Logical reference to Video Service
    user_id UUID NOT NULL, -- Logical reference to Auth Service (for notifications)
    worker_id VARCHAR(100),
    status VARCHAR(50) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'running', 'completed', 'failed', 'timeout')),
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    duration_seconds INT,
    error_message TEXT,
    retry_count INT DEFAULT 0,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_jobs_video_id ON processing_jobs(video_id);
CREATE INDEX idx_jobs_status ON processing_jobs(status);
CREATE INDEX idx_jobs_worker_id ON processing_jobs(worker_id);
CREATE INDEX idx_jobs_user_id ON processing_jobs(user_id);
CREATE INDEX idx_jobs_created_at ON processing_jobs(created_at DESC);

-- System metrics for monitoring
CREATE TABLE system_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    metric_name VARCHAR(100) NOT NULL,
    metric_value DECIMAL(20, 4) NOT NULL,
    labels JSONB,
    recorded_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_metrics_name ON system_metrics(metric_name);
CREATE INDEX idx_metrics_recorded_at ON system_metrics(recorded_at DESC);

-- View for processing health
CREATE OR REPLACE VIEW processing_health AS
SELECT 
    COUNT(CASE WHEN status = 'pending' THEN 1 END) AS pending_jobs,
    COUNT(CASE WHEN status = 'running' THEN 1 END) AS running_jobs,
    COUNT(CASE WHEN status = 'completed' THEN 1 END) AS completed_jobs,
    COUNT(CASE WHEN status = 'failed' THEN 1 END) AS failed_jobs,
    AVG(duration_seconds) AS avg_duration_seconds,
    MAX(created_at) AS last_job_at
FROM processing_jobs
WHERE created_at > NOW() - INTERVAL '1 hour';

-- Cleanup function for old data
CREATE OR REPLACE FUNCTION cleanup_old_data()
RETURNS void AS $$
BEGIN
    DELETE FROM system_metrics WHERE recorded_at < NOW() - INTERVAL '30 days';
    DELETE FROM processing_jobs 
    WHERE status = 'completed' 
    AND completed_at < NOW() - INTERVAL '30 days';
    
    RAISE NOTICE 'Processing cleanup completed';
END;
$$ LANGUAGE plpgsql;

COMMENT ON TABLE processing_jobs IS 'Tracks video processing jobs';
COMMENT ON TABLE system_metrics IS 'System performance metrics';

VACUUM ANALYZE;
