-- Insert test device data with "Unregistered" status (0)
-- Status values: 0=Unregistered, 1=Normal(Active), 2=Disabled, 3=Inactive/Scrapped
-- Online status: 0=Offline, 1=Online

-- Clear existing test data (optional, use with caution)
-- TRUNCATE TABLE device RESTART IDENTITY CASCADE;

-- Insert test device data
INSERT INTO device (
    sn,
    model,
    product_key,
    device_secret,
    firmware_version,
    hardware_version,
    mac,
    ip,
    online_status,
    status,
    create_by,
    last_active_at,
    created_at,
    updated_at,
    deleted_at
) VALUES
-- Test Device 1: UNREGISTERED (status=0), for first-time registration test
(
    'AUSP2605000001A1',
    'AudioSpeaker',
    'audio_platform_default',
    'Kp8mNx2vLq9wRt4yBh3cD6eF7gH8iJ9k',  -- Device Secret 1
    '1.0.0',
    'HW_v1.0',
    '00:11:22:33:44:55',
    '',    -- IP address (offline)
    0,     -- online_status: offline
    0,     -- status: **UNREGISTERED** (new!)
    0,     -- create_by
    NOW(), -- last_active_at
    NOW(), -- created_at
    NOW(), -- updated_at
    NULL   -- deleted_at
),

-- Test Device 2: NORMAL (status=1), complete info, the one you are using
(
    'AUSP2605000002Y2',
    'AudioSpeaker_Pro',
    'audio_platform_default',
    'G2WCNIrxLdYGBVbBze9KCH2pkQCUiKkq',  -- Device Secret 2
    '2.0.1',
    'HW_v2.0',
    'AA:BB:CC:DD:EE:FF',
    '192.168.1.100',
    0,     -- online_status: offline
    1,     -- status: **NORMAL** (active)
    0,     -- create_by
    NOW() - INTERVAL '1 hour',
    NOW() - INTERVAL '24 hours',
    NOW(),
    NULL
),

-- Test Device 3: UNREGISTERED (status=0), new device for fresh registration test
(
    'AUSP2605000003Z3',
    'AudioSpeaker_Mini',
    'audio_platform_default',
    'Xm5nQr8sTu2vWx3yZ4a5bC6dE7fG8hI9j',  -- Device Secret 3
    '1.5.0',
    'HW_v1.5',
    '11:22:33:44:55:66',
    '',    -- IP address (offline)
    0,     -- online_status: offline
    0,     -- status: **UNREGISTERED** (new!)
    0,     -- create_by
    NULL,  -- last_active_at
    NOW(), -- created_at
    NOW(), -- updated_at
    NULL   -- deleted_at
),

-- Test Device 4: DISABLED (status=2), simulating admin-disabled
(
    'AUSP2605000004B4',
    'AudioSpeaker_Lite',
    'audio_platform_default',
    'JkLmNoPqRsTuVwXyZ1a2bC3dE4fG5hI6j',  -- Device Secret 4
    '1.0.0',
    'HW_v1.0',
    '22:33:44:55:66:77',
    '10.0.0.50',
    0,     -- online_status: offline
    2,     -- status: **DISABLED**
    1,     -- create_by: admin created
    NOW() - INTERVAL '2 days',
    NOW() - INTERVAL '30 days',
    NOW(),
    NULL
),

-- Test Device 5: INACTIVE (status=3), simulating scrapped/inactive device
(
    'AUSP2605000005C5',
    'AudioSpeaker_Max',
    'audio_platform_premium',
    'MnOpQrStUvWxYzA1b2cD3eF4gH5iJ6kL7m',  -- Device Secret 5
    '3.0.0-beta',
    'HW_v3.0',
    '33:44:55:66:77:88',
    '172.16.0.100',
    1,     -- online_status: online
    3,     -- status: **INACTIVE/SCRAPPED**
    0,     -- create_by
    NOW(), -- last_active_at: just active
    NOW() - INTERVAL '7 days',
    NOW(),
    NULL
);

-- Verify insertion results
SELECT 
    id,
    sn,
    model,
    LEFT(device_secret, 20) AS device_secret_prefix,
    firmware_version,
    online_status,
    status,
    CASE 
        WHEN status = 0 THEN 'Unregistered'
        WHEN status = 1 THEN 'Normal (Active)'
        WHEN status = 2 THEN 'Disabled'
        WHEN status = 3 THEN 'Inactive/Scrapped'
        ELSE 'Unknown'
    END AS status_desc,
    CASE 
        WHEN online_status = 0 THEN 'Offline'
        WHEN online_status = 1 THEN 'Online'
        ELSE 'Unknown'
    END AS online_desc,
    created_at::date AS register_date
FROM device 
WHERE deleted_at IS NULL
ORDER BY id;

-- Output statistics
SELECT 
    COUNT(*) AS total_devices,
    COUNT(CASE WHEN status = 0 THEN 1 END) AS unregistered_count,
    COUNT(CASE WHEN status = 1 THEN 1 END) AS normal_count,
    COUNT(CASE WHEN status = 2 THEN 1 END) AS disabled_count,
    COUNT(CASE WHEN status = 3 THEN 1 END) AS inactive_count,
    COUNT(CASE WHEN online_status = 1 THEN 1 END) AS online_count,
    COUNT(CASE WHEN online_status = 0 THEN 1 END) AS offline_count
FROM device 
WHERE deleted_at IS NULL;
