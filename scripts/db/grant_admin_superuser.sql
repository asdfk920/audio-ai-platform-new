-- 可选：赋予应用用户 admin 超级用户权限，之后可用 admin 连接直接执行 apply-all-migrations（无需移交表属主）。
-- 生产环境慎用；更推荐 pre_migrate_owner_to_admin.sql 仅移交属主。
--
-- 使用 PostgreSQL 超级用户执行，例如：
--   psql -h localhost -U postgres -d postgres -v ON_ERROR_STOP=1 -f scripts/db/grant_admin_superuser.sql
--
-- 若角色名不是 admin，请修改下一行的角色名。

ALTER ROLE admin WITH SUPERUSER;
