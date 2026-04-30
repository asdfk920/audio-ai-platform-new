-- 将 go-admin 的 sys_user / sys_user_role 视图改为基于 public.sys_admin，与 C 端 public.users 彻底分离。
-- 请先执行 069_sys_admin.sql 与 070_sys_admin_backfill_from_console_users.sql，确保 sys_admin 中已有可登录账号。
SET search_path TO public;
SET client_encoding TO 'UTF8';

-- go-admin SysUser 写入所需的可选列（原 sys_admin 精简表无此字段）
ALTER TABLE public.sys_admin ADD COLUMN IF NOT EXISTS dept_id INTEGER NOT NULL DEFAULT 0;
ALTER TABLE public.sys_admin ADD COLUMN IF NOT EXISTS post_id INTEGER NOT NULL DEFAULT 0;
ALTER TABLE public.sys_admin ADD COLUMN IF NOT EXISTS remark VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE public.sys_admin ADD COLUMN IF NOT EXISTS avatar VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE public.sys_admin ADD COLUMN IF NOT EXISTS salt VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE public.sys_admin ADD COLUMN IF NOT EXISTS update_by BIGINT;

COMMENT ON COLUMN public.sys_admin.dept_id IS 'go-admin 兼容：部门';
COMMENT ON COLUMN public.sys_admin.post_id IS 'go-admin 兼容：岗位';
COMMENT ON COLUMN public.sys_admin.remark IS 'go-admin 兼容：备注';
COMMENT ON COLUMN public.sys_admin.avatar IS 'go-admin 兼容：头像';
COMMENT ON COLUMN public.sys_admin.salt IS 'go-admin 兼容：盐（可为空，bcrypt 主密文在 password）';
COMMENT ON COLUMN public.sys_admin.update_by IS 'go-admin 兼容：更新人 sys_admin.id';

-- C 端 users 上曾挂接的控制台角色，避免平台用户列表再出现 super_admin 等
DELETE FROM user_role_rel ur
USING roles r
WHERE ur.role_id = r.id
  AND LOWER(TRIM(COALESCE(NULLIF(TRIM(r.slug), ''), r.name))) IN ('super_admin', 'admin', 'operator', 'finance');

DROP VIEW IF EXISTS sys_user_role CASCADE;
DROP VIEW IF EXISTS sys_user CASCADE;

CREATE VIEW sys_user AS
SELECT
    sa.id::INT AS user_id,
    sa.username,
    sa.password,
    COALESCE(NULLIF(TRIM(sa.salt), ''), '')::VARCHAR(255) AS salt,
    COALESCE(NULLIF(TRIM(sa.nick_name), ''), '')::VARCHAR(128) AS nick_name,
    COALESCE(NULLIF(TRIM(sa.mobile), ''), '')::VARCHAR(11) AS phone,
    COALESCE(NULLIF(TRIM(sa.email), ''), '')::VARCHAR(128) AS email,
    COALESCE(NULLIF(TRIM(sa.avatar), ''), '')::VARCHAR(255) AS avatar,
    ''::VARCHAR(255) AS sex,
    COALESCE(sa.role_id, 0)::INT AS role_id,
    COALESCE(sa.dept_id, 0)::INT AS dept_id,
    COALESCE(sa.post_id, 0)::INT AS post_id,
    COALESCE(NULLIF(TRIM(sa.remark), ''), '')::VARCHAR(255) AS remark,
    CASE WHEN sa.status = 1 THEN '2' ELSE '1' END::VARCHAR(4) AS status,
    COALESCE(sa.created_by, 0)::INT AS create_by,
    COALESCE(sa.update_by, 0)::INT AS update_by,
    sa.created_at,
    sa.updated_at,
    sa.deleted_at::TIMESTAMPTZ AS deleted_at,
    COALESCE(NULLIF(TRIM(sa.last_login_ip), ''), '')::VARCHAR(50) AS login_ip,
    COALESCE(sa.login_count, 0)::BIGINT AS login_count,
    COALESCE(sa.last_login_at, TIMESTAMPTZ '1970-01-01 00:00:00+00') AS last_login_time
FROM sys_admin sa;

COMMENT ON VIEW sys_user IS 'go-admin 兼容视图；物理表 public.sys_admin（与 C 端 users 分离）';

-- 每位管理员一行；id 为合成主键，满足 GORM 对关联表主键的读取
CREATE VIEW sys_user_role AS
SELECT
    (sa.id * 1000000 + sa.role_id)::BIGINT AS id,
    sa.id::INT AS user_id,
    sa.role_id::INT AS role_id
FROM sys_admin sa
WHERE sa.deleted_at IS NULL;

COMMENT ON VIEW sys_user_role IS 'go-admin 兼容；管理员单一角色，派生自 sys_admin.role_id';
