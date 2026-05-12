-- ============================================================
-- 设备影子表字段补全脚本
-- 用途：为现有的 device_shadow 表添加 WILL 消息处理所需的字段
-- 执行方式：在 PostgreSQL 中执行此 SQL
-- 注意：使用 IF NOT EXISTS 避免重复执行报错
-- ============================================================

-- 1️⃣ 添加在线状态字段（核心字段 - 必需）
ALTER TABLE device_shadow 
ADD COLUMN IF NOT EXISTS online_status SMALLINT NOT NULL DEFAULT 0;

COMMENT ON COLUMN device_shadow.online_status IS '在线状态：0-离线、1-在线';

-- 2️⃣ 添加固件版本字段
ALTER TABLE device_shadow 
ADD COLUMN IF NOT EXISTS firmware_version VARCHAR(64) DEFAULT '';

COMMENT ON COLUMN device_shadow.firmware_version IS '固件版本号';

-- 3️⃣ 添加电量字段
ALTER TABLE device_shadow 
ADD COLUMN IF NOT EXISTS battery_level INTEGER;

COMMENT ON COLUMN device_shadow.battery_level IS '电量百分比 (0-100)';

-- 4️⃣ 添加最后指令时间字段
ALTER TABLE device_shadow 
ADD COLUMN IF NOT EXISTS last_command_time TIMESTAMP WITH TIME ZONE;

COMMENT ON COLUMN device_shadow.last_command_time IS '最后指令下发时间';

-- 5️⃣ 添加元数据字段（JSON格式）
ALTER TABLE device_shadow 
ADD COLUMN IF NOT EXISTS metadata JSONB DEFAULT '{}';

COMMENT ON COLUMN device_shadow.metadata IS '影子元数据（版本信息等）';

-- 6️⃣ 添加版本号字段（乐观锁）
ALTER TABLE device_shadow 
ADD COLUMN IF NOT EXISTS version BIGINT NOT NULL DEFAULT 0;

COMMENT ON COLUMN device_shadow.version IS '版本号（乐观锁机制，每次更新+1）';

-- 7️⃣ 添加断开类型字段（WILL消息处理必需）
ALTER TABLE device_shadow 
ADD COLUMN IF NOT EXISTS disconnect_type VARCHAR(32) DEFAULT '';

COMMENT ON COLUMN device_shadow.disconnect_type IS '断开类型：normal-正常断开、will-WILL消息触发、keepalive_timeout-心跳超时、kicked-被踢出、unknown-未知原因';

-- 8️⃣ 添加断开时间字段（WILL消息处理必需）
ALTER TABLE device_shadow 
ADD COLUMN IF NOT EXISTS disconnect_at TIMESTAMP WITH TIME ZONE;

COMMENT ON COLUMN device_shadow.disconnect_at IS '设备最后断开时间';

-- 9️⃣ 添加更新时间字段
ALTER TABLE device_shadow 
ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW();

COMMENT ON COLUMN device_shadow.updated_at IS '记录最后更新时间';

-- ============================================================
-- 创建索引（提升查询性能）
-- ============================================================

-- 在线状态索引（用于统计在线/离线设备数）
CREATE INDEX IF NOT EXISTS idx_device_shadow_online_status 
ON device_shadow(online_status);

-- 断开时间索引（用于清理历史数据和查询最近掉线设备）
CREATE INDEX IF NOT EXISTS idx_device_shadow_disconnect_at 
ON device_shadow(disconnect_at);

-- 版本号索引（用于并发控制查询）
CREATE INDEX IF NOT EXISTS idx_device_shadow_version 
ON device_shadow(version);

-- ============================================================
-- 验证：查看补全后的表结构
-- ============================================================

-- 执行以下命令确认所有字段已添加：
-- \d device_shadow

-- 或者执行这个查询查看列信息：
SELECT 
    column_name,
    data_type,
    is_nullable,
    column_default
FROM information_schema.columns 
WHERE table_name = 'device_shadow'
ORDER BY ordinal_position;
