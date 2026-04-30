-- 用户注册与 OAuth：支持手机/邮箱二选一、密码加盐、第三方登录
-- 1. 邮箱改为可空（支持仅手机号注册）
-- 2. 密码改为可空（OAuth 用户可能无密码）
-- 3. 增加 salt 字段用于密码加盐

ALTER TABLE users ALTER COLUMN email DROP NOT NULL;
ALTER TABLE users ALTER COLUMN password DROP NOT NULL;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name = 'users' AND column_name = 'salt'
  ) THEN
    ALTER TABLE users ADD COLUMN salt VARCHAR(64) DEFAULT NULL;
  END IF;
END $$;
