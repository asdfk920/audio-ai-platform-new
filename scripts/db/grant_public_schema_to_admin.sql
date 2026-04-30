-- PostgreSQL 15+ 可能默认禁止普通用户在 public 下 CREATE，导致迁移报「对模式 public 权限不够」。
-- 用超级用户执行一次（库名、用户名按实际修改）：
--
--   psql -U postgres -d audio_platform -f scripts/db/grant_public_schema_to_admin.sql

GRANT USAGE, CREATE ON SCHEMA public TO admin;
