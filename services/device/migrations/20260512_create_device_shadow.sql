-- 设备影子表 - 用于存储设备实时状态和期望状态
-- 支持设备影子同步、WILL消息处理、在线状态管理

CREATE TABLE IF NOT EXISTS device_shadow (
    id              BIGSERIAL PRIMARY KEY,
    sn              VARCHAR(32) NOT NULL UNIQUE,                    -- 设备序列号
    online_status   SMALLINT NOT NULL DEFAULT 0,                   -- 在线状态：0-离线、1-在线
    firmware_version VARCHAR(64) DEFAULT '',                        -- 固件版本号
    battery_level   INTEGER,                                       -- 电量百分比 (0-100)
    last_report_time TIMESTAMP WITH TIME ZONE,                     -- 最后上报时间
    last_command_time TIMESTAMP WITH TIME ZONE,                    -- 最后指令时间
    reported        JSONB DEFAULT '{}',                            -- 设备上报的属性（JSON格式）
    desired         JSONB DEFAULT '{}',                            -- 云端期望的属性（JSON格式）
    metadata        JSONB DEFAULT '{}',                            -- 影子元数据（版本信息等）
    version         BIGINT NOT NULL DEFAULT 0,                     -- 版本号（乐观锁，每次更新+1）
    disconnect_type VARCHAR(32) DEFAULT '',                         -- 断开类型：normal/will/keepalive_timeout/kicked/unknown
    disconnect_at   TIMESTAMP WITH TIME ZONE,                      -- 断开时间
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 创建索引：按SN查询设备影子
CREATE INDEX IF NOT EXISTS idx_device_shadow_sn ON device_shadow(sn);

-- 创建索引：按在线状态查询（用于统计在线设备数）
CREATE INDEX IF NOT EXISTS idx_device_shadow_online_status ON device_shadow(online_status);

-- 创建索引：按断开时间查询（用于清理历史数据）
CREATE INDEX IF NOT EXISTS idx_device_shadow_disconnect_at ON device_shadow(disconnect_at);

-- 添加注释
COMMENT ON TABLE device_shadow IS '设备影子表 - 存储设备实时状态和期望状态';
COMMENT ON COLUMN device_shadow.sn IS '设备序列号（唯一标识）';
COMMENT ON COLUMN device_shadow.online_status IS '在线状态：0-离线、1-在线';
COMMENT ON COLUMN device_shadow.reported IS '设备上报的属性（JSON格式，如电量、音量等）';
COMMENT ON COLUMN device_shadow.desired IS '云端期望的属性（JSON格式，待下发的指令等）';
COMMENT ON COLUMN device_shadow.version IS '版本号（乐观锁机制，防止并发冲突）';
COMMENT ON COLUMN device_shadow.disconnect_type IS '断开类型：normal-正常断开、will-WILL消息触发、keepalive_timeout-心跳超时、kicked-被踢出、unknown-未知原因';

-- 示例数据（可选）
-- INSERT INTO device_shadow (sn, online_status, firmware_version, battery_level, reported, desired)
-- VALUES ('AUSP2605000001G2', 1, 'v1.2.3', 85, '{"battery":85,"volume":50,"state":"playing"}', '{}');