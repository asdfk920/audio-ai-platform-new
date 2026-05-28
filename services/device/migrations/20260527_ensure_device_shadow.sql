-- 设备影子表（与 scripts/db/migrations/026 + 057 对齐；device 微服务启动/注册时可幂等执行）
CREATE TABLE IF NOT EXISTS public.device_shadow (
    id BIGSERIAL PRIMARY KEY,
    device_id BIGINT NOT NULL REFERENCES public.device(id) ON DELETE CASCADE,
    sn VARCHAR(64) NOT NULL,
    reported JSONB DEFAULT '{}'::jsonb,
    desired JSONB DEFAULT '{}'::jsonb,
    metadata JSONB DEFAULT '{}'::jsonb,
    version BIGINT NOT NULL DEFAULT 0,
    last_report_time TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_device_shadow_device_id UNIQUE (device_id)
);

CREATE INDEX IF NOT EXISTS idx_device_shadow_sn ON public.device_shadow(sn);

ALTER TABLE public.device_shadow ADD COLUMN IF NOT EXISTS version BIGINT NOT NULL DEFAULT 0;
ALTER TABLE public.device_shadow ADD COLUMN IF NOT EXISTS metadata JSONB DEFAULT '{}'::jsonb;

COMMENT ON TABLE public.device_shadow IS '设备影子表：desired/reported 持久化，与 Redis Hash device:shadow:{SN} 同步';
