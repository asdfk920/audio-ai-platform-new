-- OTA 服务数据库迁移脚本（PostgreSQL）
-- 创建固件版本表和升级任务表

-- ============================================
-- 删除旧表（如果存在）
-- ============================================
DROP TABLE IF EXISTS ota_upgrade_history;
DROP TABLE IF EXISTS ota_upgrade_task;
DROP TABLE IF EXISTS ota_firmware;

-- ============================================
-- 固件版本表
-- ============================================
CREATE TABLE ota_firmware (
  id BIGSERIAL PRIMARY KEY,
  model VARCHAR(50) NOT NULL,
  version VARCHAR(50) NOT NULL,
  device_type VARCHAR(50) NOT NULL DEFAULT '',
  upgrade_type VARCHAR(20) NOT NULL DEFAULT 'optional',
  changelog TEXT,
  download_url VARCHAR(500) NOT NULL,
  file_size BIGINT NOT NULL DEFAULT 0,
  file_md5 VARCHAR(64) NOT NULL,
  published SMALLINT NOT NULL DEFAULT 0,
  gray_scale SMALLINT NOT NULL DEFAULT 0,
  gray_percent INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ
);

COMMENT ON TABLE ota_firmware IS 'OTA固件版本表，存储所有设备型号的固件版本信息';
COMMENT ON COLUMN ota_firmware.id IS '主键ID';
COMMENT ON COLUMN ota_firmware.model IS '设备型号，如 X1、X2、Pro、Mini 等';
COMMENT ON COLUMN ota_firmware.version IS '固件版本号，如 FW_1.0.0、FW_2.0.0';
COMMENT ON COLUMN ota_firmware.device_type IS '设备类型，如 speaker、soundbar、headphone';
COMMENT ON COLUMN ota_firmware.upgrade_type IS '升级类型：force-强制升级，optional-可选升级';
COMMENT ON COLUMN ota_firmware.changelog IS '更新日志，记录版本变更内容';
COMMENT ON COLUMN ota_firmware.download_url IS '固件下载地址，支持 HTTPS 协议';
COMMENT ON COLUMN ota_firmware.file_size IS '文件大小（字节），用于显示和校验';
COMMENT ON COLUMN ota_firmware.file_md5 IS '文件 MD5 校验值，确保下载完整性';
COMMENT ON COLUMN ota_firmware.published IS '是否发布：0-未发布（测试中），1-已发布（可用）';
COMMENT ON COLUMN ota_firmware.gray_scale IS '是否灰度发布：0-全量发布，1-灰度发布';
COMMENT ON COLUMN ota_firmware.gray_percent IS '灰度比例（0-100），控制升级设备百分比';
COMMENT ON COLUMN ota_firmware.created_at IS '创建时间';
COMMENT ON COLUMN ota_firmware.updated_at IS '更新时间';
COMMENT ON COLUMN ota_firmware.deleted_at IS '删除时间（软删除标记）';

CREATE INDEX idx_ota_firmware_model_version ON ota_firmware(model, version);
CREATE INDEX idx_ota_firmware_model_published ON ota_firmware(model, published, deleted_at);
CREATE INDEX idx_ota_firmware_created_at ON ota_firmware(created_at);

-- ============================================
-- 升级任务表
-- ============================================
CREATE TABLE ota_upgrade_task (
  id BIGSERIAL PRIMARY KEY,
  task_id VARCHAR(64) NOT NULL,
  device_sn VARCHAR(50) NOT NULL,
  device_model VARCHAR(50) NOT NULL,
  device_type VARCHAR(50) NOT NULL DEFAULT '',
  user_id BIGINT NOT NULL DEFAULT 0,
  from_version VARCHAR(50) NOT NULL,
  to_version VARCHAR(50) NOT NULL,
  upgrade_type VARCHAR(20) NOT NULL DEFAULT 'optional',
  status SMALLINT NOT NULL DEFAULT 0,
  progress INT NOT NULL DEFAULT 0,
  download_url VARCHAR(500) NOT NULL,
  file_size BIGINT NOT NULL DEFAULT 0,
  file_md5 VARCHAR(64) NOT NULL,
  error_message TEXT,
  started_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE ota_upgrade_task IS 'OTA升级任务表，记录设备固件升级任务及进度';
COMMENT ON COLUMN ota_upgrade_task.id IS '主键ID';
COMMENT ON COLUMN ota_upgrade_task.task_id IS '任务唯一ID，用于追踪升级进度';
COMMENT ON COLUMN ota_upgrade_task.device_sn IS '设备序列号，标识升级设备';
COMMENT ON COLUMN ota_upgrade_task.device_model IS '设备型号';
COMMENT ON COLUMN ota_upgrade_task.device_type IS '设备类型';
COMMENT ON COLUMN ota_upgrade_task.user_id IS '用户ID，设备所属用户';
COMMENT ON COLUMN ota_upgrade_task.from_version IS '升级前版本号';
COMMENT ON COLUMN ota_upgrade_task.to_version IS '升级目标版本号';
COMMENT ON COLUMN ota_upgrade_task.upgrade_type IS '升级类型：force-强制，optional-可选';
COMMENT ON COLUMN ota_upgrade_task.status IS '任务状态：0-待升级，1-下载中，2-升级中，3-成功，4-失败，5-已取消';
COMMENT ON COLUMN ota_upgrade_task.progress IS '升级进度（0-100），百分比表示';
COMMENT ON COLUMN ota_upgrade_task.download_url IS '固件下载地址';
COMMENT ON COLUMN ota_upgrade_task.file_size IS '固件文件大小';
COMMENT ON COLUMN ota_upgrade_task.file_md5 IS '固件 MD5 校验值';
COMMENT ON COLUMN ota_upgrade_task.error_message IS '错误信息，升级失败时记录原因';
COMMENT ON COLUMN ota_upgrade_task.started_at IS '开始升级时间';
COMMENT ON COLUMN ota_upgrade_task.completed_at IS '完成升级时间';
COMMENT ON COLUMN ota_upgrade_task.created_at IS '任务创建时间';
COMMENT ON COLUMN ota_upgrade_task.updated_at IS '任务更新时间';

CREATE UNIQUE INDEX uk_ota_upgrade_task_task_id ON ota_upgrade_task(task_id);
CREATE INDEX idx_ota_upgrade_task_device_sn ON ota_upgrade_task(device_sn);
CREATE INDEX idx_ota_upgrade_task_user_id ON ota_upgrade_task(user_id);
CREATE INDEX idx_ota_upgrade_task_status ON ota_upgrade_task(status);
CREATE INDEX idx_ota_upgrade_task_created_at ON ota_upgrade_task(created_at);

-- ============================================
-- 升级记录表
-- ============================================
CREATE TABLE ota_upgrade_history (
  id BIGSERIAL PRIMARY KEY,
  device_sn VARCHAR(50) NOT NULL,
  device_model VARCHAR(50) NOT NULL,
  user_id BIGINT NOT NULL DEFAULT 0,
  from_version VARCHAR(50) NOT NULL,
  to_version VARCHAR(50) NOT NULL,
  upgrade_type VARCHAR(20) NOT NULL DEFAULT 'optional',
  status SMALLINT NOT NULL DEFAULT 3,
  duration INT NOT NULL DEFAULT 0,
  error_message TEXT,
  upgrade_time TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE ota_upgrade_history IS 'OTA升级历史记录表，用于统计和分析升级情况';
COMMENT ON COLUMN ota_upgrade_history.id IS '主键ID';
COMMENT ON COLUMN ota_upgrade_history.device_sn IS '设备序列号';
COMMENT ON COLUMN ota_upgrade_history.device_model IS '设备型号';
COMMENT ON COLUMN ota_upgrade_history.user_id IS '用户ID';
COMMENT ON COLUMN ota_upgrade_history.from_version IS '升级前版本';
COMMENT ON COLUMN ota_upgrade_history.to_version IS '升级后版本';
COMMENT ON COLUMN ota_upgrade_history.upgrade_type IS '升级类型';
COMMENT ON COLUMN ota_upgrade_history.status IS '升级结果：3-成功，4-失败';
COMMENT ON COLUMN ota_upgrade_history.duration IS '升级耗时（秒）';
COMMENT ON COLUMN ota_upgrade_history.error_message IS '错误信息（失败时）';
COMMENT ON COLUMN ota_upgrade_history.upgrade_time IS '升级完成时间';
COMMENT ON COLUMN ota_upgrade_history.created_at IS '记录创建时间';

CREATE INDEX idx_ota_upgrade_history_device_sn ON ota_upgrade_history(device_sn);
CREATE INDEX idx_ota_upgrade_history_user_id ON ota_upgrade_history(user_id);
CREATE INDEX idx_ota_upgrade_history_upgrade_time ON ota_upgrade_history(upgrade_time);

-- ============================================
-- 插入固件版本测试数据
-- ============================================
INSERT INTO ota_firmware (model, version, device_type, upgrade_type, changelog, download_url, file_size, file_md5, published, gray_scale, gray_percent) VALUES
('X1', 'FW_1.0.0', 'speaker', 'optional', '初始版本', 'https://firmware.example.com/x1/FW_1.0.0.bin', 1024000, 'a1b2c3d4e5f6789012345678901234567', 1, 0, 0),
('X1', 'FW_1.1.0', 'speaker', 'optional', '1. 优化音频播放效果
2. 修复已知问题', 'https://firmware.example.com/x1/FW_1.1.0.bin', 1048576, 'b2c3d4e5f67890123456789012345678', 1, 0, 0),
('X1', 'FW_2.0.0', 'speaker', 'force', '1. 全新UI界面
2. 支持蓝牙5.0
3. 提升系统稳定性', 'https://firmware.example.com/x1/FW_2.0.0.bin', 2097152, 'c3d4e5f6789012345678901234567890', 1, 1, 30),
('X2', 'FW_1.0.0', 'speaker', 'optional', '初始版本', 'https://firmware.example.com/x2/FW_1.0.0.bin', 1024000, 'd4e5f67890123456789012345678901a', 1, 0, 0),
('X2', 'FW_1.2.0', 'speaker', 'optional', '1. 新增语音助手功能
2. 优化网络连接稳定性
3. 修复充电指示灯异常', 'https://firmware.example.com/x2/FW_1.2.0.bin', 1153433, 'e5f67890123456789012345678901ab2', 1, 0, 0),
('X2', 'FW_2.0.0', 'speaker', 'optional', '1. 支持多房间音频同步
2. 新增EQ调节功能
3. 优化电池续航', 'https://firmware.example.com/x2/FW_2.0.0.bin', 1572864, 'f67890123456789012345678901ab2c3', 1, 1, 50),
('Pro', 'FW_1.0.0', 'speaker', 'optional', '初始版本', 'https://firmware.example.com/pro/FW_1.0.0.bin', 2048000, 'a2b3c4d5e6f789012345678901234567', 1, 0, 0),
('Pro', 'FW_1.5.0', 'speaker', 'optional', '1. 支持无损音频播放
2. 新增空间音频功能
3. 优化DAC驱动', 'https://firmware.example.com/pro/FW_1.5.0.bin', 2359296, 'b3c4d5e6f7890123456789012345678a', 1, 0, 0),
('Pro', 'FW_2.0.0', 'speaker', 'force', '1. 安全漏洞修复
2. 加密传输升级
3. 系统内核更新', 'https://firmware.example.com/pro/FW_2.0.0.bin', 3145728, 'c4d5e6f78901234567890123456789ab', 1, 1, 20),
('Mini', 'FW_1.0.0', 'speaker', 'optional', '初始版本', 'https://firmware.example.com/mini/FW_1.0.0.bin', 512000, 'd5e6f789012345678901234567890abc', 1, 0, 0),
('Mini', 'FW_1.1.0', 'speaker', 'optional', '1. 优化低功耗模式
2. 修复蓝牙配对问题', 'https://firmware.example.com/mini/FW_1.1.0.bin', 524288, 'e6f789012345678901234567890abcd5', 1, 0, 0),
('Mini', 'FW_1.2.0', 'speaker', 'optional', '1. 新增定时关机功能
2. 优化触摸按键灵敏度', 'https://firmware.example.com/mini/FW_1.2.0.bin', 536576, 'f789012345678901234567890abcde6f', 1, 0, 0),
('SoundBar', 'FW_1.0.0', 'soundbar', 'optional', '初始版本', 'https://firmware.example.com/soundbar/FW_1.0.0.bin', 3072000, 'a3b4c5d6e7f890123456789012345678', 1, 0, 0),
('SoundBar', 'FW_1.3.0', 'soundbar', 'optional', '1. 支持HDMI ARC
2. 新增电影模式
3. 优化低音效果', 'https://firmware.example.com/soundbar/FW_1.3.0.bin', 3407872, 'b4c5d6e7f8901234567890123456789a', 1, 0, 0),
('SoundBar', 'FW_2.0.0', 'soundbar', 'optional', '1. 支持Dolby Atmos
2. 新增WiFi 6支持
3. 多设备联动功能', 'https://firmware.example.com/soundbar/FW_2.0.0.bin', 4194304, 'c5d6e7f89012345678901234567890ab', 1, 1, 40),
('Headphone', 'FW_1.0.0', 'headphone', 'optional', '初始版本', 'https://firmware.example.com/headphone/FW_1.0.0.bin', 768000, 'd6e7f8901234567890123456789012ab', 1, 0, 0),
('Headphone', 'FW_1.2.0', 'headphone', 'optional', '1. 优化主动降噪算法
2. 新增通透模式
3. 修复触控延迟', 'https://firmware.example.com/headphone/FW_1.2.0.bin', 819200, 'e7f8901234567890123456789012abcd', 1, 0, 0),
('Headphone', 'FW_2.0.0', 'headphone', 'force', '1. 电池安全补丁
2. 充电保护升级
3. 修复过热问题', 'https://firmware.example.com/headphone/FW_2.0.0.bin', 1048576, 'f8901234567890123456789012abcdef', 1, 1, 25),
('X1', 'FW_2.1.0', 'speaker', 'optional', '1. 修复FW_2.0.0已知问题
2. 优化启动速度
3. 新增自定义EQ', 'https://firmware.example.com/x1/FW_2.1.0.bin', 2150400, 'a4b5c6d7e8f901234567890123456789', 0, 0, 0),
('Pro', 'FW_2.1.0', 'speaker', 'optional', '1. 新增LDAC编解码支持
2. 优化USB DAC模式
3. 修复偶发断连问题', 'https://firmware.example.com/pro/FW_2.1.0.bin', 3250176, 'b5c6d7e8f9012345678901234567890a', 0, 0, 0);
