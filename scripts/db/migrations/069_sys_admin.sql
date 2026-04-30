-- 后台管理员专用账号表（与 C 端 public.users 分离，仅存控制台管理员信息）
SET search_path TO public;
SET client_encoding TO 'UTF8';

CREATE TABLE IF NOT EXISTS sys_admin (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(64) NOT NULL,
    password VARCHAR(255) NOT NULL,
    real_name VARCHAR(64),
    nick_name VARCHAR(64),
    mobile VARCHAR(20),
    email VARCHAR(100),
    role_id BIGINT NOT NULL REFERENCES public.roles(id) ON DELETE RESTRICT,
    role_name VARCHAR(50),
    role_code VARCHAR(50),
    status SMALLINT NOT NULL DEFAULT 1,
    login_count INTEGER NOT NULL DEFAULT 0,
    last_login_at TIMESTAMPTZ,
    last_login_ip VARCHAR(50),
    password_expired_at TIMESTAMPTZ,
    password_changed_at TIMESTAMPTZ,
    created_by BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_sys_admin_username_lower
    ON sys_admin (LOWER(TRIM(username)))
    WHERE deleted_at IS NULL AND username IS NOT NULL AND TRIM(username) <> '';

COMMENT ON TABLE sys_admin IS '控制台管理员账号信息（与 users/sys_user 视图独立；登录链路可按需迁移至此表）';

COMMENT ON COLUMN sys_admin.username IS '登录用户名，唯一（忽略大小写，仅未删除行）';
COMMENT ON COLUMN sys_admin.password IS '密码哈希（如 bcrypt）';
COMMENT ON COLUMN sys_admin.real_name IS '真实姓名';
COMMENT ON COLUMN sys_admin.nick_name IS '昵称';
COMMENT ON COLUMN sys_admin.mobile IS '手机号码';
COMMENT ON COLUMN sys_admin.email IS '邮箱';
COMMENT ON COLUMN sys_admin.role_id IS '关联 public.roles.id';
COMMENT ON COLUMN sys_admin.role_name IS '角色名称冗余展示';
COMMENT ON COLUMN sys_admin.role_code IS '角色标识冗余（与 roles.slug 一致）';
COMMENT ON COLUMN sys_admin.status IS '状态：0 禁用 1 启用';
COMMENT ON COLUMN sys_admin.login_count IS '登录次数';
COMMENT ON COLUMN sys_admin.last_login_at IS '最后登录时间';
COMMENT ON COLUMN sys_admin.last_login_ip IS '最后登录 IP';
COMMENT ON COLUMN sys_admin.password_expired_at IS '密码过期时间';
COMMENT ON COLUMN sys_admin.password_changed_at IS '密码修改时间';
COMMENT ON COLUMN sys_admin.created_by IS '创建人（管理员 id，可选）';

CREATE INDEX IF NOT EXISTS idx_sys_admin_role_id ON sys_admin(role_id);
CREATE INDEX IF NOT EXISTS idx_sys_admin_status ON sys_admin(status);

-- 首次落库且尚无管理员时插入默认控制台账号（username=admin / password=admin123），与 public.users 无任何关联
INSERT INTO sys_admin (username, password, nick_name, role_id, role_name, role_code, status)
SELECT
    'admin',
    '$2a$10$8x3jB7RG9dZqTaEsGBb8IuwcdzmHtYfZ6EwuebWj5XVC4zvBWWd0W',
    'admin',
    r.id,
    LEFT(BTRIM(COALESCE(NULLIF(TRIM(r.name), ''), 'admin')), 50),
    LEFT(BTRIM(COALESCE(NULLIF(TRIM(r.slug), ''), 'admin')), 50),
    1
FROM roles r
WHERE LOWER(TRIM(COALESCE(NULLIF(TRIM(r.slug), ''), r.name))) = 'admin'
  AND NOT EXISTS (SELECT 1 FROM sys_admin WHERE deleted_at IS NULL)
LIMIT 1;
