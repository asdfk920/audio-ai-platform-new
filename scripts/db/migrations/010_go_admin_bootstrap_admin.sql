-- 历史：曾向 public.users + user_role_rel 写入默认控制台账号，与 C 端用户耦合。
-- 现控制台账号仅落在 public.sys_admin（见 069 末尾默认行；登录依赖 081 将 sys_user 视图映射至 sys_admin）。
-- 本迁移保留为空操作，便于旧环境迁移序号不变。
SET search_path TO public;

SELECT 1;
