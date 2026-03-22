CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS notifications (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    batch_id        UUID,
    channel         VARCHAR(10) NOT NULL CHECK (channel IN ('sms', 'email', 'push')),
    recipient       VARCHAR(255) NOT NULL,
    content         TEXT NOT NULL,
    priority        SMALLINT NOT NULL DEFAULT 1 CHECK (priority BETWEEN 0 AND 2),
    status          VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'queued', 'processing', 'sent', 'failed', 'cancelled')),
    attempt_count   INT NOT NULL DEFAULT 0,
    max_attempts    INT NOT NULL DEFAULT 3,
    last_error      TEXT,
    sent_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notifications_batch_id ON notifications(batch_id) WHERE batch_id IS NOT NULL;
CREATE INDEX idx_notifications_status ON notifications(status);
CREATE INDEX idx_notifications_channel_status ON notifications(channel, status);
CREATE INDEX idx_notifications_created_at ON notifications(created_at);

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_notifications_updated_at
    BEFORE UPDATE ON notifications
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
