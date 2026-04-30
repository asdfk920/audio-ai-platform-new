-- 020: users 表补齐 avatar 字段（头像 URL/路径）
SET search_path TO public;

ALTER TABLE users
  ADD COLUMN IF NOT EXISTS avatar VARCHAR(500);

COMMENT ON COLUMN users.avatar IS '头像（URL 或 /static 路径），注册时若未提供则生成默认头像';

