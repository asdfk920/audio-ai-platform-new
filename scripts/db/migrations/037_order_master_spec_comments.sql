-- 037: 订单主表 product 语义注释（表结构由 036 创建；与业务文档字段含义对齐）
-- 说明：金额在库内以「分」整型存储（amount_cent），对应 DECIMAL(10,2) 元金额 * 100。
SET search_path TO public;

COMMENT ON TABLE public.order_master IS '订单主表：订单号唯一，关联用户、套餐编码与支付状态';
COMMENT ON COLUMN public.order_master.order_no IS '订单号（唯一）';
COMMENT ON COLUMN public.order_master.user_id IS '用户ID';
COMMENT ON COLUMN public.order_master.package_code IS '关联套餐编码';
COMMENT ON COLUMN public.order_master.pay_type IS '支付方式：1微信 2支付宝 3余额';
COMMENT ON COLUMN public.order_master.pay_status IS '0待支付 1已支付 2已取消';
COMMENT ON COLUMN public.order_master.amount_cent IS '订单金额（分），等价于元金额 DECIMAL(10,2) 乘 100';
