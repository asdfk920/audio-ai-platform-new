-- 回滚 users 扩展字段（按需执行）

ALTER TABLE users DROP COLUMN IF EXISTS gender;
ALTER TABLE users DROP COLUMN IF EXISTS birthday;
ALTER TABLE users DROP COLUMN IF EXISTS device_id;
ALTER TABLE users DROP COLUMN IF EXISTS invite_code;
ALTER TABLE users DROP COLUMN IF EXISTS deleted_at;

ALTER TABLE users DROP COLUMN IF EXISTS timezone;
ALTER TABLE users DROP COLUMN IF EXISTS language;
ALTER TABLE users DROP COLUMN IF EXISTS user_type;
ALTER TABLE users DROP COLUMN IF EXISTS login_fail_count;
ALTER TABLE users DROP COLUMN IF EXISTS account_locked_until;

ALTER TABLE users DROP COLUMN IF EXISTS register_channel;
ALTER TABLE users DROP COLUMN IF EXISTS password_changed_at;
ALTER TABLE users DROP COLUMN IF EXISTS last_login_ip;
ALTER TABLE users DROP COLUMN IF EXISTS last_login_at;
ALTER TABLE users DROP COLUMN IF EXISTS register_ip;

DROP INDEX IF EXISTS idx_users_register_channel;
DROP INDEX IF EXISTS idx_users_deleted_at;
