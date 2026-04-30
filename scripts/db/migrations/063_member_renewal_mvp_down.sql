-- 063 down
SET search_path TO public;

ALTER TABLE public.user_member DROP CONSTRAINT IF EXISTS user_member_auto_renew_check;
ALTER TABLE public.user_member DROP CONSTRAINT IF EXISTS user_member_auto_renew_pay_type_check;
ALTER TABLE public.user_member
    DROP COLUMN IF EXISTS auto_renew_updated_at,
    DROP COLUMN IF EXISTS auto_renew_pay_type,
    DROP COLUMN IF EXISTS auto_renew_package_code,
    DROP COLUMN IF EXISTS auto_renew;

ALTER TABLE public.member_package DROP COLUMN IF EXISTS list_price_cent;

ALTER TABLE public.order_master DROP CONSTRAINT IF EXISTS order_master_biz_scene_check;
ALTER TABLE public.order_master
    DROP COLUMN IF EXISTS biz_scene,
    DROP COLUMN IF EXISTS discount_cent,
    DROP COLUMN IF EXISTS original_amount_cent;
