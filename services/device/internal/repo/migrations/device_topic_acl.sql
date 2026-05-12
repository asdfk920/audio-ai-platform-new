-- 设备 Topic 权限控制表（ACL - Access Control List）
-- 用于实现 MQTT 发布/订阅的细粒度权限控制

CREATE TABLE IF NOT EXISTS device_topic_acl (
    id BIGSERIAL PRIMARY KEY,
    sn VARCHAR(64) NOT NULL,                    -- 设备序列号，关联设备主表
    topic_pattern VARCHAR(255) NOT NULL,         -- Topic 规则，支持通配符 + / #
    action SMALLINT NOT NULL DEFAULT 2,          -- 操作类型：0=订阅(subscribe), 1=发布(publish), 2=两者都允许(both)
    permission SMALLINT NOT NULL DEFAULT 1,      -- 权限类型：0=拒绝(黑名单/deny), 1=允许(白名单/allow)
    description VARCHAR(500) DEFAULT '',         -- 规则描述说明
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT uk_sn_topic_action UNIQUE (sn, topic_pattern, action)
);

COMMENT ON TABLE device_topic_acl IS '设备 Topic 权限控制表';
COMMENT ON COLUMN device_topic_acl.id IS '主键ID';
COMMENT ON COLUMN device_topic_acl.sn IS '设备序列号，关联设备主表';
COMMENT ON COLUMN device_topic_acl.topic_pattern IS 'Topic 规则，支持通配符 + 和 #';
COMMENT ON COLUMN device_topic_acl.action IS '操作类型：0=订阅, 1=发布, 2=两者都允许';
COMMENT ON COLUMN device_topic_acl.permission IS '权限类型：0=拒绝(黑名单), 1=允许(白名单)';
COMMENT ON COLUMN device_topic_acl.description IS '规则描述说明';

-- 创建索引以提升查询性能
CREATE INDEX IF NOT EXISTS idx_device_topic_acl_sn ON device_topic_acl(sn);
CREATE INDEX IF NOT EXISTS idx_device_topic_acl_sn_permission ON device_topic_acl(sn, permission);

-- 初始化默认的设备 Topic 白名单规则（适用于所有设备的通用模板）
-- 注意：实际使用时需要将 ${sn} 替换为具体的设备SN，或使用通配符规则

-- 示例：为特定设备插入白名单规则
-- INSERT INTO device_topic_acl (sn, topic_pattern, action, permission, description) VALUES
-- ('AUSP2605000001G2', 'device/${sn}/status', 1, 1, '设备状态上报-发布'),
-- ('AUSP2605000001G2', 'device/${sn}/log', 1, 1, '设备日志上报-发布'),
-- ('AUSP2605000001G2', 'audio/${sn}/upload', 1, 1, '音频数据上传-发布'),
-- ('AUSP2605000001G2', 'cmd/${sn}/down', 0, 1, '平台指令下发-订阅'),
-- ('AUSP2605000001G2', 'ota/${sn}/info', 0, 1, 'OTA升级通知-订阅'),
-- ('AUSP2605000001G2', 'diagnose/${sn}/cmd', 0, 1, '远程诊断指令-订阅');

-- 系统级黑名单规则示例（禁止访问系统级 Topic）
-- INSERT INTO device_topic_acl (sn, topic_pattern, action, permission, description) VALUES
-- ('*', '$SYS/#', 2, 0, '禁止访问服务端系统级 Topic'),
-- ('*', 'device/+/status', 2, 0, '禁止访问其他设备的状态 Topic');