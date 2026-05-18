-- 为 device 表添加 register_signature 字段
-- 用于存储设备注册时生成的 HMAC-SHA256 签名（HMAC-SHA256(device_secret, sn + timestamp)）
-- 该签名将在后续 WebSocket 认证中使用

-- 检查字段是否存在，不存在则添加
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_name = 'device' 
        AND column_name = 'register_signature'
    ) THEN
        ALTER TABLE device ADD COLUMN register_signature VARCHAR(64);
        
        COMMENT ON COLUMN device.register_signature IS '设备注册签名：HMAC-SHA256(device_secret, sn + timestamp)，用于WebSocket认证';
        
        RAISE NOTICE '字段 register_signature 已成功添加到 device 表';
    ELSE
        RAISE NOTICE '字段 register_signature 已存在，跳过添加';
    END IF;
END $$;
