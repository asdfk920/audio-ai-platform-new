SET search_path TO public;

DROP INDEX IF EXISTS idx_iot_product_status;
DROP INDEX IF EXISTS uk_iot_product_key_lower;
DROP TABLE IF EXISTS public.iot_product;
