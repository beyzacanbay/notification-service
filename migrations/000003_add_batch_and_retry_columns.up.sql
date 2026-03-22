ALTER TABLE notifications
    ADD COLUMN batch_id UUID,
    ADD COLUMN attempt_count INT NOT NULL DEFAULT 0,
    ADD COLUMN max_attempts INT NOT NULL DEFAULT 3,
    ADD COLUMN last_error TEXT,
    ADD COLUMN sent_at TIMESTAMPTZ;

CREATE INDEX idx_notifications_batch_id ON notifications(batch_id) WHERE batch_id IS NOT NULL;
