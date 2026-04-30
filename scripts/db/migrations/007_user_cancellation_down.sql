DROP TABLE IF EXISTS user_cancellation_log;

ALTER TABLE users DROP COLUMN IF EXISTS account_cancelled_at;
ALTER TABLE users DROP COLUMN IF EXISTS cancellation_cooling_until;
