-- 手工建过 public.users 后，用迁移用户（默认 admin）跑 001 会因「非属主」无法在 users 上建索引而失败。
-- 请用 PostgreSQL 超级用户执行本文件一次，把 users 及其序列属主交给 admin（若库用户不同，请替换 admin）。
--
--   psql -U postgres -d audio_platform -f scripts/db/grant_users_table_to_admin.sql
--   Docker: docker exec -i audio-platform-postgres psql -U postgres -d audio_platform -f - < scripts/db/grant_users_table_to_admin.sql

ALTER TABLE public.users OWNER TO admin;

-- BIGSERIAL 默认序列名；若 id 序列改名请手工 ALTER SEQUENCE … OWNER TO admin
ALTER SEQUENCE IF EXISTS public.users_id_seq OWNER TO admin;
