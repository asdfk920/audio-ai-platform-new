-- 平台用户管理：展示「姓名」与资料编辑（与实名认证流水可并存）
SET search_path TO public;

ALTER TABLE users ADD COLUMN IF NOT EXISTS real_name VARCHAR(64) NULL;
COMMENT ON COLUMN users.real_name IS '真实姓名（运营维护；可与 user_real_name_auth 实名流水并存）';
