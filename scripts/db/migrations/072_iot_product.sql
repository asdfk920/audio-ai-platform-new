-- 后台产品线（product_key 元数据），与 ota_firmware / device 通过 product_key 字符串关联
SET search_path TO public;
SET client_encoding TO 'UTF8';

CREATE TABLE IF NOT EXISTS public.iot_product (
    id BIGSERIAL PRIMARY KEY,
    product_key VARCHAR(64) NOT NULL,
    name VARCHAR(128) NOT NULL DEFAULT '',
    category VARCHAR(64) NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    communication JSONB NOT NULL DEFAULT '[]'::jsonb,
    device_type VARCHAR(64) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'draft',
    created_by BIGINT NOT NULL DEFAULT 0,
    updated_by BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ NULL,
    CONSTRAINT iot_product_status_check CHECK (status IN ('draft', 'published', 'disabled'))
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_iot_product_key_lower
    ON public.iot_product (LOWER(TRIM(product_key)))
    WHERE deleted_at IS NULL AND product_key IS NOT NULL AND BTRIM(product_key) <> '';

CREATE INDEX IF NOT EXISTS idx_iot_product_status ON public.iot_product (status) WHERE deleted_at IS NULL;

COMMENT ON TABLE public.iot_product IS 'IoT 产品线：product_key 与 ota_firmware/device 对齐';
COMMENT ON COLUMN public.iot_product.product_key IS '产品标识（认证/固件关联），未删除行唯一（忽略大小写）';
COMMENT ON COLUMN public.iot_product.status IS 'draft=草稿 published=已发布 disabled=禁用';
