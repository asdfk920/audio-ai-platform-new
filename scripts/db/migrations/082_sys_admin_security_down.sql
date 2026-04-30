SET search_path TO public;

ALTER TABLE IF EXISTS sys_admin
    DROP COLUMN IF EXISTS allowed_ips,
    DROP COLUMN IF EXISTS allowed_login_start,
    DROP COLUMN IF EXISTS allowed_login_end,
    DROP COLUMN IF EXISTS must_change_password,
    DROP COLUMN IF EXISTS last_password_changed_at;
