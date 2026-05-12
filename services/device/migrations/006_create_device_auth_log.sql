-- ============================================================
-- Device Auth Log Table - 设备MQTT认证日志表
-- 用于记录所有设备的MQTT连接认证请求
-- 版本: 006
-- 日期: 2026-05-12
-- ============================================================

CREATE TABLE IF NOT EXISTS device_auth_log (
    id BIGSERIAL PRIMARY KEY,
    device_sn VARCHAR(50) NOT NULL,              -- 设备SN编号
    client_id VARCHAR(100),                      -- MQTT ClientID
    auth_result VARCHAR(20) NOT NULL,            -- 认证结果: success | failed
    error_msg TEXT,                              -- 失败原因说明
    client_ip VARCHAR(50),                       -- 客户端IP地址
    user_agent VARCHAR(255),                     -- 客户端标识
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP  -- 认证时间
);

-- 创建索引以优化查询性能
CREATE INDEX IF NOT EXISTS idx_device_auth_log_sn ON device_auth_log(device_sn);
CREATE INDEX IF NOT EXISTS idx_device_auth_log_result ON device_auth_log(auth_result);
CREATE INDEX IF NOT EXISTS idx_device_auth_log_created_at ON device_auth_log(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_device_auth_log_sn_created ON device_auth_log(device_sn, created_at DESC);

-- 添加注释
COMMENT ON TABLE device_auth_log IS '设备MQTT认证日志表 - 记录所有设备的MQTT连接认证请求和结果';
COMMENT ON COLUMN device_auth_log.id IS '主键ID';
COMMENT ON COLUMN device_auth_log.device_sn IS '设备SN编号';
COMMENT ON COLUMN device_auth_log.client_id IS 'MQTT客户端ID';
COMMENT ON COLUMN device_auth_log.auth_result IS '认证结果: success=成功 | failed=失败';
COMMENT ON COLUMN device_auth_log.error_msg IS '失败原因说明（仅失败时填写）';
COMMENT ON COLUMN device_auth_log.client_id IS '客户端IP地址';
COMMENT ON COLUMN device_auth_log.user_agent IS '客户端User-Agent或来源标识';
COMMENT ON COLUMN device_auth_log.created_at IS '认证请求时间';

-- 定期清理策略（保留最近90天的日志）
-- 可通过定时任务执行：DELETE FROM device_auth_log WHERE created_at < NOW() - INTERVAL '90 days'
