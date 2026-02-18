-- Notification Service Database Schema

-- Notifications table - Core entity for Notification Service
-- Note: user_id and video_id are logical references to other services
CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL, -- Logical reference to Auth Service
    video_id UUID, -- Logical reference to Video Service
    type VARCHAR(50) NOT NULL CHECK (type IN ('email', 'sms', 'push')),
    status VARCHAR(50) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'sent', 'failed')),
    subject VARCHAR(255),
    message TEXT NOT NULL,
    recipient VARCHAR(255) NOT NULL,
    sent_at TIMESTAMP,
    error_message TEXT,
    retry_count INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_notifications_user_id ON notifications(user_id);
CREATE INDEX idx_notifications_status ON notifications(status);
CREATE INDEX idx_notifications_type ON notifications(type);
CREATE INDEX idx_notifications_created_at ON notifications(created_at DESC);

-- Notification templates
CREATE TABLE notification_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_name VARCHAR(100) NOT NULL UNIQUE,
    template_type VARCHAR(50) NOT NULL CHECK (template_type IN ('email', 'sms', 'push')),
    subject_template VARCHAR(255),
    body_template TEXT NOT NULL,
    variables JSONB, -- List of expected variables
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_templates_name ON notification_templates(template_name);
CREATE INDEX idx_templates_type ON notification_templates(template_type);

-- Trigger to update updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_templates_updated_at BEFORE UPDATE ON notification_templates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- View for notification statistics
CREATE OR REPLACE VIEW notification_stats AS
SELECT 
    type,
    COUNT(*) AS total,
    COUNT(CASE WHEN status = 'sent' THEN 1 END) AS sent,
    COUNT(CASE WHEN status = 'failed' THEN 1 END) AS failed,
    COUNT(CASE WHEN status = 'pending' THEN 1 END) AS pending
FROM notifications
WHERE created_at > NOW() - INTERVAL '24 hours'
GROUP BY type;

-- Cleanup function
CREATE OR REPLACE FUNCTION cleanup_old_notifications()
RETURNS void AS $$
BEGIN
    DELETE FROM notifications 
    WHERE status = 'sent' 
    AND sent_at < NOW() - INTERVAL '30 days';
    
    RAISE NOTICE 'Notification cleanup completed';
END;
$$ LANGUAGE plpgsql;

-- Insert default templates
INSERT INTO notification_templates (template_name, template_type, subject_template, body_template, variables)
VALUES 
(
    'video_completed',
    'email',
    'Video Processing Completed - {{video_name}}',
    'Hello {{user_name}},\n\nYour video "{{video_name}}" has been processed successfully!\n\nYou can download it here: {{download_url}}\n\nBest regards,\nG57 Team',
    '{"user_name": "string", "video_name": "string", "download_url": "string"}'::jsonb
),
(
    'video_failed',
    'email',
    'Video Processing Failed - {{video_name}}',
    'Hello {{user_name}},\n\nUnfortunately, your video "{{video_name}}" failed to process.\n\nError: {{error_message}}\n\nPlease try uploading again or contact support.\n\nBest regards,\nG57 Team',
    '{"user_name": "string", "video_name": "string", "error_message": "string"}'::jsonb
);

COMMENT ON TABLE notifications IS 'Stores notification queue and history';
COMMENT ON TABLE notification_templates IS 'Reusable notification templates';


