-- 初始化数据库脚本
-- 创建用户表（支持邮箱/手机号注册、密码加盐、OAuth 无密码）
-- 部分客户端 search_path 为空时会报「创建中没有选择模式」，显式落到 public
SET search_path TO public;

CREATE TABLE IF NOT EXISTS public.users (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE,
    mobile VARCHAR(20),
    password VARCHAR(255),
    salt VARCHAR(64),
    nickname VARCHAR(100),
    avatar VARCHAR(500),
    status SMALLINT DEFAULT 1,
    register_ip VARCHAR(45),
    last_login_at TIMESTAMP NULL,
    last_login_ip VARCHAR(45),
    password_changed_at TIMESTAMP NULL,
    register_channel VARCHAR(20),
    account_locked_until TIMESTAMP NULL,
    login_fail_count SMALLINT NOT NULL DEFAULT 0,
    user_type SMALLINT NOT NULL DEFAULT 1,
    language VARCHAR(10) NOT NULL DEFAULT 'zh-CN',
    timezone VARCHAR(50) NOT NULL DEFAULT 'Asia/Shanghai',
    deleted_at TIMESTAMP NULL,
    invite_code VARCHAR(32),
    device_id VARCHAR(128),
    birthday DATE NULL,
    gender SMALLINT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_email ON public.users(email);
CREATE INDEX IF NOT EXISTS idx_users_mobile ON public.users(mobile);

-- 创建用户认证表
CREATE TABLE IF NOT EXISTS public.user_auth (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    auth_type VARCHAR(20) NOT NULL,
    auth_id VARCHAR(255) NOT NULL,
    refresh_token TEXT,
    expired_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(auth_type, auth_id)
);

CREATE INDEX IF NOT EXISTS idx_user_auth_user_id ON public.user_auth(user_id);

-- 创建角色表
CREATE TABLE IF NOT EXISTS public.roles (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    permissions JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 创建用户角色关联表
CREATE TABLE IF NOT EXISTS public.user_role_rel (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    role_id BIGINT NOT NULL REFERENCES public.roles(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, role_id)
);

-- 设备主数据见迁移 026 的 public.device；不再创建历史表 devices / device_user_rel / device_commands。

-- 创建原始内容表
CREATE TABLE IF NOT EXISTS public.raw_contents (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    file_key VARCHAR(500) NOT NULL,
    file_size BIGINT,
    duration INTEGER,
    upload_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status VARCHAR(20) DEFAULT 'uploaded'
);

CREATE INDEX IF NOT EXISTS idx_raw_contents_user_id ON public.raw_contents(user_id);
CREATE INDEX IF NOT EXISTS idx_raw_contents_status ON public.raw_contents(status);

-- 创建处理后内容表
CREATE TABLE IF NOT EXISTS public.processed_contents (
    id BIGSERIAL PRIMARY KEY,
    raw_content_id BIGINT NOT NULL REFERENCES public.raw_contents(id) ON DELETE CASCADE,
    ai_model_version VARCHAR(50),
    file_key VARCHAR(500) NOT NULL,
    cdn_url VARCHAR(500),
    duration INTEGER,
    bitrate INTEGER,
    status VARCHAR(20) DEFAULT 'processing',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_processed_contents_raw_content_id ON public.processed_contents(raw_content_id);
CREATE INDEX IF NOT EXISTS idx_processed_contents_status ON public.processed_contents(status);

-- 创建最终内容表
CREATE TABLE IF NOT EXISTS public.contents (
    id BIGSERIAL PRIMARY KEY,
    processed_content_id BIGINT NOT NULL REFERENCES public.processed_contents(id) ON DELETE CASCADE,
    title VARCHAR(200),
    duration INTEGER,
    status VARCHAR(20) DEFAULT 'online',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_contents_status ON public.contents(status);

-- 创建播放记录表
CREATE TABLE IF NOT EXISTS public.content_play_records (
    id BIGSERIAL PRIMARY KEY,
    content_id BIGINT NOT NULL REFERENCES public.contents(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    device_id BIGINT,
    play_start_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    play_end_time TIMESTAMP,
    play_duration INTEGER
);

CREATE INDEX IF NOT EXISTS idx_content_play_records_content_id ON public.content_play_records(content_id);
CREATE INDEX IF NOT EXISTS idx_content_play_records_user_id ON public.content_play_records(user_id);
CREATE INDEX IF NOT EXISTS idx_content_play_records_device_id ON public.content_play_records(device_id);

-- 插入默认角色
INSERT INTO public.roles (name, permissions) VALUES
('admin', '{"all": true}'::jsonb),
('user', '{"read": true, "write": false}'::jsonb)
ON CONFLICT (name) DO NOTHING;
