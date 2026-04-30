-- 回滚 020：移除 users.avatar
SET search_path TO public;

ALTER TABLE users
  DROP COLUMN IF EXISTS avatar;

