-- 创建用户设备绑定操作日志表
SET search_path TO public;

CREATE TABLE IF NOT EXISTS public.user_device_bind_log (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    device_id BIGINT NOT NULL,
    sn VARCHAR(64) NOT NULL,
    operator VARCHAR(64),
    action VARCHAR(32) NOT NULL,
    action_time TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_bind_log_user_id
    ON public.user_device_bind_log(user_id);

CREATE INDEX IF NOT EXISTS idx_bind_log_device_id
    ON public.user_device_bind_log(device_id);

COMMENT ON TABLE public.user_device_bind_log IS '用户设备绑定操作日志表';
COMMENT ON COLUMN public.user_device_bind_log.operator IS '操作人标识';
COMMENT ON COLUMN public.user_device_bind_log.action IS '操作类型，如 bind/unbind';
