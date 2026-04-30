SET search_path TO public;

DROP TABLE IF EXISTS public.member_pay_log;
DROP TABLE IF EXISTS public.pay_log;
DROP TABLE IF EXISTS public.order_master;
DROP TABLE IF EXISTS public.member_package;

ALTER TABLE public.users DROP COLUMN IF EXISTS balance_cent;
