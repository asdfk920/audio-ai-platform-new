-- 回滚 008_user_center_schema（删除视图与新表；移除 user_auth 扩展列）

DROP VIEW IF EXISTS sys_user;
DROP VIEW IF EXISTS sys_user_role;
DROP VIEW IF EXISTS sys_role;

DROP TABLE IF EXISTS jwt_blacklist;
DROP TABLE IF EXISTS user_login_log;
DROP TABLE IF EXISTS user_member;
DROP TABLE IF EXISTS user_verify_code;

DROP INDEX IF EXISTS idx_user_auth_user_id_auth_type;

ALTER TABLE user_auth DROP COLUMN IF EXISTS updated_at;
ALTER TABLE user_auth DROP COLUMN IF EXISTS avatar;
ALTER TABLE user_auth DROP COLUMN IF EXISTS nickname;
ALTER TABLE user_auth DROP COLUMN IF EXISTS unionid;
