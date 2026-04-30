-- 回滚 004（若 content_play_records 存在 device_id IS NULL 则 SET NOT NULL 会失败）
DROP INDEX IF EXISTS idx_users_mobile_unique;
-- ALTER TABLE content_play_records ALTER COLUMN device_id SET NOT NULL;
