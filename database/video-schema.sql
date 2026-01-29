-- Video Service Database Schema
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Videos table - Core entity for Video Service
-- Note: user_id is stored but user data comes from Auth Service via HTTP
CREATE TABLE videos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL, -- Foreign key reference is logical, not enforced (different DB)
    filename VARCHAR(255) NOT NULL,
    original_name VARCHAR(255) NOT NULL,
    size_bytes BIGINT NOT NULL,
    duration_seconds DECIMAL(10, 2),
    status VARCHAR(50) NOT NULL DEFAULT 'pending' 
        CHECK (status IN ('pending', 'queued', 'processing', 'completed', 'failed', 'cancelled')),
    storage_path VARCHAR(500),
    zip_path VARCHAR(500),
    zip_size_bytes BIGINT,
    frame_count INT,
    error_message TEXT,
    retry_count INT DEFAULT 0,
    priority INT DEFAULT 5 CHECK (priority BETWEEN 1 AND 10),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    queued_at TIMESTAMP,
    processing_started_at TIMESTAMP,
    processing_completed_at TIMESTAMP
);

CREATE INDEX idx_videos_user_id ON videos(user_id);
CREATE INDEX idx_videos_status ON videos(status);
CREATE INDEX idx_videos_created_at ON videos(created_at DESC);
CREATE INDEX idx_videos_user_status ON videos(user_id, status);
CREATE INDEX idx_videos_user_created ON videos(user_id, created_at DESC);

-- Trigger to update updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_videos_updated_at BEFORE UPDATE ON videos
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- View for video statistics
CREATE OR REPLACE VIEW video_stats AS
SELECT 
    COUNT(*) AS total_videos,
    COUNT(CASE WHEN status = 'completed' THEN 1 END) AS completed_videos,
    COUNT(CASE WHEN status = 'failed' THEN 1 END) AS failed_videos,
    COUNT(CASE WHEN status = 'processing' THEN 1 END) AS processing_videos,
    COUNT(CASE WHEN status = 'pending' THEN 1 END) AS pending_videos,
    AVG(CASE WHEN status = 'completed' 
        THEN EXTRACT(EPOCH FROM (processing_completed_at - processing_started_at)) 
        END) AS avg_processing_time_seconds,
    MAX(created_at) AS last_upload_at
FROM videos
WHERE created_at > NOW() - INTERVAL '24 hours';

-- Function to get user videos stats
CREATE OR REPLACE FUNCTION get_user_video_stats(p_user_id UUID)
RETURNS TABLE (
    total_videos BIGINT,
    completed_videos BIGINT,
    failed_videos BIGINT,
    total_storage_mb DECIMAL,
    avg_processing_time_seconds DECIMAL
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        COUNT(*)::BIGINT,
        COUNT(CASE WHEN status = 'completed' THEN 1 END)::BIGINT,
        COUNT(CASE WHEN status = 'failed' THEN 1 END)::BIGINT,
        (SUM(size_bytes) / 1024.0 / 1024.0)::DECIMAL(10,2),
        AVG(EXTRACT(EPOCH FROM (processing_completed_at - processing_started_at)))::DECIMAL(10,2)
    FROM videos
    WHERE user_id = p_user_id;
END;
$$ LANGUAGE plpgsql;

COMMENT ON TABLE videos IS 'Stores video metadata and processing status';

VACUUM ANALYZE;
