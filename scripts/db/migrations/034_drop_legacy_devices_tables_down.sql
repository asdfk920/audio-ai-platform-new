-- 回滚 034：仅恢复空表结构（不含 content_play_records 外键，避免与现有数据冲突）
SET search_path TO public;

CREATE TABLE IF NOT EXISTS public.devices (
    id BIGSERIAL PRIMARY KEY,
    sn VARCHAR(100) UNIQUE NOT NULL,
    model VARCHAR(50),
    firmware_version VARCHAR(50),
    status VARCHAR(20) DEFAULT 'offline',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_devices_sn ON public.devices(sn);
CREATE INDEX IF NOT EXISTS idx_devices_status ON public.devices(status);

CREATE TABLE IF NOT EXISTS public.device_user_rel (
    id BIGSERIAL PRIMARY KEY,
    device_id BIGINT NOT NULL REFERENCES public.devices(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    bind_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    unbind_time TIMESTAMP,
    is_valid BOOLEAN DEFAULT true
);

CREATE INDEX IF NOT EXISTS idx_device_user_rel_device_id ON public.device_user_rel(device_id);
CREATE INDEX IF NOT EXISTS idx_device_user_rel_user_id ON public.device_user_rel(user_id);

CREATE TABLE IF NOT EXISTS public.device_commands (
    id BIGSERIAL PRIMARY KEY,
    device_id BIGINT NOT NULL REFERENCES public.devices(id) ON DELETE CASCADE,
    command_type VARCHAR(20) NOT NULL,
    command_content JSONB,
    status VARCHAR(20) DEFAULT 'pending',
    create_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    execute_time TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_device_commands_device_id ON public.device_commands(device_id);
CREATE INDEX IF NOT EXISTS idx_device_commands_status ON public.device_commands(status);
