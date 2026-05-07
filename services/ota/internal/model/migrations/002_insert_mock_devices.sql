-- OTA 服务模拟设备数据（PostgreSQL）
-- 根据 ota_firmware 表中的 model 和 version 创建对应设备，并绑定到 user_id=1

SET search_path TO public;

-- ============================================
-- 0. 确保测试用户存在（user_id=1）
-- ============================================
-- user_device_bind 有外键约束引用 users(id)，必须先创建用户

INSERT INTO public.users (id, email, nickname, status, user_type, language, timezone)
VALUES (1, 'test@example.com', '测试用户', 1, 1, 'zh-CN', 'Asia/Shanghai')
ON CONFLICT (id) DO NOTHING;

-- 确保序列从正确位置开始（避免后续插入用户时 ID 冲突）
SELECT setval('public.users_id_seq', COALESCE((SELECT MAX(id) FROM public.users), 1), true);

-- ============================================
-- 1. 插入模拟设备数据（device 表）
-- ============================================
-- 设备固件版本与 ota_firmware 表对应，部分设备使用旧版本以便测试升级检测

INSERT INTO public.device (sn, product_key, device_secret, firmware_version, hardware_version, model, mac, ip, online_status, status)
VALUES
  -- X1 系列设备
  ('SN-X1-001', 'audio-x1-001', '$2a$10$mockhash001xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx', 'FW_1.0.0', 'HW_1.0', 'X1', 'AA:BB:CC:DD:EE:01', '192.168.1.101', 1, 1),
  ('SN-X1-002', 'audio-x1-001', '$2a$10$mockhash002xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx', 'FW_1.1.0', 'HW_1.0', 'X1', 'AA:BB:CC:DD:EE:02', '192.168.1.102', 1, 1),
  ('SN-X1-003', 'audio-x1-001', '$2a$10$mockhash003xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx', 'FW_2.0.0', 'HW_1.0', 'X1', 'AA:BB:CC:DD:EE:03', '192.168.1.103', 0, 1),

  -- X2 系列设备
  ('SN-X2-001', 'audio-x2-001', '$2a$10$mockhash004xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx', 'FW_1.0.0', 'HW_2.0', 'X2', 'AA:BB:CC:DD:EE:04', '192.168.1.104', 1, 1),
  ('SN-X2-002', 'audio-x2-001', '$2a$10$mockhash005xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx', 'FW_1.2.0', 'HW_2.0', 'X2', 'AA:BB:CC:DD:EE:05', '192.168.1.105', 1, 1),

  -- Pro 系列设备
  ('SN-PRO-001', 'audio-pro-001', '$2a$10$mockhash006xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx', 'FW_1.0.0', 'HW_3.0', 'Pro', 'AA:BB:CC:DD:EE:06', '192.168.1.106', 1, 1),
  ('SN-PRO-002', 'audio-pro-001', '$2a$10$mockhash007xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx', 'FW_1.5.0', 'HW_3.0', 'Pro', 'AA:BB:CC:DD:EE:07', '192.168.1.107', 1, 1),

  -- Mini 系列设备
  ('SN-MINI-001', 'audio-mini-001', '$2a$10$mockhash008xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx', 'FW_1.0.0', 'HW_4.0', 'Mini', 'AA:BB:CC:DD:EE:08', '192.168.1.108', 1, 1),
  ('SN-MINI-002', 'audio-mini-001', '$2a$10$mockhash009xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx', 'FW_1.1.0', 'HW_4.0', 'Mini', 'AA:BB:CC:DD:EE:09', '192.168.1.109', 0, 1),

  -- SoundBar 系列设备
  ('SN-SBAR-001', 'audio-sbar-001', '$2a$10$mockhash010xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx', 'FW_1.0.0', 'HW_5.0', 'SoundBar', 'AA:BB:CC:DD:EE:10', '192.168.1.110', 1, 1),
  ('SN-SBAR-002', 'audio-sbar-001', '$2a$10$mockhash011xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx', 'FW_1.3.0', 'HW_5.0', 'SoundBar', 'AA:BB:CC:DD:EE:11', '192.168.1.111', 1, 1),

  -- Headphone 系列设备
  ('SN-HP-001', 'audio-hp-001', '$2a$10$mockhash012xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx', 'FW_1.0.0', 'HW_6.0', 'Headphone', 'AA:BB:CC:DD:EE:12', '192.168.1.112', 1, 1),
  ('SN-HP-002', 'audio-hp-001', '$2a$10$mockhash013xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx', 'FW_1.2.0', 'HW_6.0', 'Headphone', 'AA:BB:CC:DD:EE:13', '192.168.1.113', 1, 1)
ON CONFLICT (sn) DO NOTHING;

-- ============================================
-- 2. 绑定设备到用户 user_id=1（user_device_bind 表）
-- ============================================
-- 将上述设备绑定到用户 1，status=1 表示绑定中

INSERT INTO public.user_device_bind (user_id, device_id, sn, alias, is_default, bind_type, status)
SELECT
  1 AS user_id,
  d.id AS device_id,
  d.sn,
  CASE
    WHEN d.model = 'X1' THEN '客厅音箱'
    WHEN d.model = 'X2' THEN '卧室音箱'
    WHEN d.model = 'Pro' THEN '专业音箱'
    WHEN d.model = 'Mini' THEN '便携音箱'
    WHEN d.model = 'SoundBar' THEN '回音壁'
    WHEN d.model = 'Headphone' THEN '头戴耳机'
    ELSE d.sn
  END AS alias,
  CASE WHEN d.sn LIKE '%-001' THEN 1 ELSE 0 END AS is_default,
  1 AS bind_type,
  1 AS status
FROM public.device d
WHERE d.sn IN (
  'SN-X1-001', 'SN-X1-002', 'SN-X1-003',
  'SN-X2-001', 'SN-X2-002',
  'SN-PRO-001', 'SN-PRO-002',
  'SN-MINI-001', 'SN-MINI-002',
  'SN-SBAR-001', 'SN-SBAR-002',
  'SN-HP-001', 'SN-HP-002'
)
ON CONFLICT (user_id, device_id) DO NOTHING;

-- ============================================
-- 3. 验证数据
-- ============================================
-- 查看已插入的设备
SELECT d.sn, d.model, d.firmware_version, d.online_status, udb.alias, udb.status AS bind_status
FROM public.device d
LEFT JOIN public.user_device_bind udb ON udb.device_id = d.id AND udb.user_id = 1
WHERE d.sn LIKE 'SN-%'
ORDER BY d.model, d.sn;
