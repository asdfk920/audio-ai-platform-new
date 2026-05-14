-- 给 users 表字段添加注释
-- 执行时间：2026-05-14
-- 说明：为数据库表字段添加中文注释，提高可读性和维护性

COMMENT ON COLUMN users.id IS '用户主键ID（自增）';
COMMENT ON COLUMN users.email IS '邮箱地址（登录凭证之一）';
COMMENT ON COLUMN users.mobile IS '手机号码（登录凭证之一）';
COMMENT ON COLUMN users.password IS '密码哈希值（bcrypt加密）';
COMMENT ON COLUMN users.salt IS '密码盐值（用于密码加密）';
COMMENT ON COLUMN users.nickname IS '用户昵称/显示名称';
COMMENT ON COLUMN users.avatar IS '头像URL地址（支持本地路径或外部URL）';
COMMENT ON COLUMN users.status IS '账户状态：0=禁用 1=正常 2=待验证 3=已注销';
COMMENT ON COLUMN users.register_ip IS '注册时IP地址（IPv4/IPv6）';
COMMENT ON COLUMN users.last_login_at IS '最后登录时间';
COMMENT ON COLUMN users.last_login_ip IS '最后登录IP地址（IPv4/IPv6）';
COMMENT ON COLUMN users.password_changed_at IS '最后修改密码时间';
COMMENT ON COLUMN users.register_channel IS '注册渠道：web/app/wechat/google等';
COMMENT ON COLUMN users.account_locked_until IS '账户锁定截止时间（多次登录失败后锁定）';
COMMENT ON COLUMN users.login_fail_count IS '连续登录失败次数（达到阈值自动锁定）';
COMMENT ON COLUMN users.user_type IS '用户类型：0=普通用户 1=VIP会员 2=管理员';
COMMENT ON COLUMN users.language IS '语言偏好：zh-CN/en-US/ja-JP等';
COMMENT ON COLUMN users.timezone IS '时区设置：Asia/Shanghai/America/New_York等';
COMMENT ON COLUMN users.deleted_at IS '软删除时间（NULL表示未删除）';
COMMENT ON COLUMN users.invite_code IS '邀请码（用户专属邀请码）';

-- 查看注释是否添加成功
SELECT
    column_name,
    data_type,
    character_maximum_length,
    is_nullable,
    column_default,
    pg_catalog.col_description((SELECT oid FROM pg_class WHERE relname = 'users'), ordinal_position) AS comment
FROM information_schema.columns
WHERE table_name = 'users'
ORDER BY ordinal_position;
