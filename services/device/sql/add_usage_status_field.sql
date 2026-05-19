-- Add usage_status field to device table
-- Usage Status: 1=Enabled, 2=Disabled
-- When device is first added: usage_status=2 (Disabled)

-- Step 1: Add new column if not exists
ALTER TABLE device ADD COLUMN IF NOT EXISTS usage_status smallint NOT NULL DEFAULT 2;

-- Step 2: Add constraint for usage_status values (1 or 2)
ALTER TABLE device DROP CONSTRAINT IF EXISTS device_usage_status_check;
ALTER TABLE device 
ADD CONSTRAINT device_usage_status_check 
CHECK (usage_status IN (1, 2));

-- Step 3: Update test devices with correct initial state
-- Initial state for newly added devices:
-- - online_status = 0 (Offline)
-- - usage_status = 2 (Disabled)
-- - status = 4 (Unregistered)

UPDATE device SET 
    online_status = 0,
    usage_status = 2,
    status = 4,
    updated_at = NOW()
WHERE deleted_at IS NULL;

-- Step 4: Verify the update
SELECT 
    id,
    sn,
    model,
    online_status AS "Online",
    CASE 
        WHEN online_status = 0 THEN 'Offline'
        WHEN online_status = 1 THEN 'Online'
        ELSE 'Unknown'
    END AS "Online_Status",
    usage_status AS "Usage",
    CASE 
        WHEN usage_status = 1 THEN 'Enabled'
        WHEN usage_status = 2 THEN 'Disabled'
        ELSE 'Unknown'
    END AS "Usage_Status",
    status AS "Device",
    CASE 
        WHEN status = 0 THEN 'Default'
        WHEN status = 1 THEN 'Normal'
        WHEN status = 2 THEN 'Disabled'
        WHEN status = 3 THEN 'Inactive'
        WHEN status = 4 THEN 'Unregistered'
        WHEN status = 5 THEN 'Unauthenticated'
        ELSE 'Unknown'
    END AS "Device_Status",
    created_at::date AS created_date
FROM device 
WHERE deleted_at IS NULL
ORDER BY id;
