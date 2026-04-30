-- 一次性：将 public 下表与序列的属主改为 admin（应用连接用户）。
-- 在出现「必须是表 users 的属主」且当前库由 postgres 等超级用户建表、而迁移用 admin 连接时执行。
--
-- 使用 PostgreSQL 超级用户执行，例如：
--   psql -U postgres -d audio_platform -f scripts/db/pre_migrate_owner_to_admin.sql
--   或 Docker: docker exec -i audio-platform-postgres psql -U admin -d audio_platform -f - < scripts/db/pre_migrate_owner_to_admin.sql
--
-- 若应用库用户不是 admin，请先在本文件中全局替换 admin 为你的用户名。

DO $$
DECLARE
  r RECORD;
BEGIN
  FOR r IN
    SELECT tablename FROM pg_tables WHERE schemaname = 'public'
  LOOP
    EXECUTE format('ALTER TABLE public.%I OWNER TO admin', r.tablename);
  END LOOP;
  FOR r IN
    SELECT sequence_name FROM information_schema.sequences WHERE sequence_schema = 'public'
  LOOP
    EXECUTE format('ALTER SEQUENCE public.%I OWNER TO admin', r.sequence_name);
  END LOOP;
END $$;

ALTER SCHEMA public OWNER TO admin;
