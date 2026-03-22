DROP INDEX IF EXISTS idx_notifications_batch_id;

ALTER TABLE notifications
    DROP COLUMN IF EXISTS batch_id,
    DROP COLUMN IF EXISTS attempt_count,
    DROP COLUMN IF EXISTS max_attempts,
    DROP COLUMN IF EXISTS last_error,
    DROP COLUMN IF EXISTS sent_at;
