-- Add comments to device table columns
-- This script adds detailed Chinese comments for all fields in the device table

-- Main identification fields
COMMENT ON COLUMN device.id IS '设备主键ID，自增主键';
COMMENT ON COLUMN device.sn IS '设备序列号(SN)，全局唯一标识符';
COMMENT ON COLUMN device.model IS '设备型号/机型，如AudioSpeaker、AudioSpeaker_Pro';
COMMENT ON COLUMN device.product_key IS '产品Key（用于区分产品型号/产品线），如audio_platform_default';
COMMENT ON COLUMN device.device_secret IS '设备密钥（验证/鉴权），出厂时烧录，用于HMAC-SHA256签名计算';

-- Version information
COMMENT ON COLUMN device.firmware_version IS '当前固件版本，如1.0.0';
COMMENT ON COLUMN device.hardware_version IS '硬件版本，如HW_v1.0';

-- Network information
COMMENT ON COLUMN device.mac IS 'MAC地址，设备物理网卡地址，格式：00:11:22:33:44:55';
COMMENT ON COLUMN device.ip IS '最后一次上报IP，设备连接云端时的公网IP或内网IP';

-- Status fields (Three-state management model)
COMMENT ON COLUMN device.online_status IS '在线状态：0=离线(Offline)，1=在线(Online)';
COMMENT ON COLUMN device.usage_status IS '使用状态（管理员控制）：1=启用(Enabled)，2=禁用(Disabled)';
COMMENT ON COLUMN device.status IS '设备状态（生命周期）：0=默认(Default)，1=正常(Normal)，2=禁用(Disabled)，3=未激活(Inactive)，4=未注册(Unregistered)，5=未认证(Unauthenticated)';

-- Registration and authentication
COMMENT ON COLUMN device.register_signature IS '设备注册签名：HMAC-SHA256(device_secret, sn + timestamp)，用于WebSocket认证校验';
COMMENT ON COLUMN device.device_auth_token IS '设备认证Token，用于MQTT/WebSocket连接认证，JWT格式';

-- Timestamps
COMMENT ON COLUMN device.last_active_at IS '最后一次活跃时间，设备最后上报数据或心跳的时间戳';
COMMENT ON COLUMN device.created_at IS '创建时间，记录插入数据库的时间';
COMMENT ON COLUMN device.updated_at IS '更新时间，记录最后一次修改的时间';
COMMENT ON COLUMN device.deleted_at IS '软删除时间（GORM DeletedAt），NULL表示未删除';

-- Audit fields
COMMENT ON COLUMN device.create_by IS '创建人（后台用户ID），记录谁创建了该设备';
COMMENT ON COLUMN device.update_by IS '更新人（后台用户ID），记录谁最后修改了该设备';
COMMENT ON COLUMN device.admin_display_name IS '管理员显示名称，备用字段';
COMMENT ON COLUMN device.admin_display_name_text IS '管理员显示名称文本，备用字段';

-- Additional JSON fields
COMMENT ON COLUMN device.extra_info IS '扩展信息JSON，存储设备额外配置信息';
COMMENT ON COLUMN device.attributes IS '设备属性JSON，存储设备自定义属性，如设备类型、品牌、型号等。';
COMMENT ON COLUMN device.device_config IS '设备配置JSON，存储设备运行配置参数';

-- Verify the comments were added successfully
SELECT 
    column_name,
    data_type,
    is_nullable,
    column_default,
    pg_catalog.col_description((table_schema || '.' || table_name)::regclass::oid, ordinal_position) as comment
FROM information_schema.columns 
WHERE table_schema = 'public' AND table_name = 'device'
ORDER BY ordinal_position;
