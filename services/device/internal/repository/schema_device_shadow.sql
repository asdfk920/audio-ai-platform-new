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
ALTER TABLE public.device_shadow ADD COLUMN IF NOT EXISTS desired JSONB DEFAULT '{}'::jsonb;
