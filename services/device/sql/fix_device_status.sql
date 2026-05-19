-- Fix device status constraint to match business requirements
-- Status values: 0=Default, 1=Normal, 2=Disabled, 3=Inactive, 4=Unregistered, 5=Unauthenticated

-- Step 1: Drop existing constraint
ALTER TABLE device DROP CONSTRAINT IF EXISTS device_status_check;

-- Step 2: Update constraint to include all valid statuses (0-5)
ALTER TABLE device 
ADD CONSTRAINT device_status_check 
CHECK (status >= 0 AND status <= 5);

-- Step 3: Update test devices to use correct status values
-- Set unregistered devices to status=4 instead of 0
UPDATE device SET status = 4 WHERE sn IN ('AUSP2605000001A1', 'AUSP2605000003Z3');
-- Keep disabled as status=2
UPDATE device SET status = 2 WHERE sn = 'AUSP2605000004B4';
-- Keep inactive as status=3
UPDATE device SET status = 3 WHERE sn = 'AUSP2605000005C5';

-- Step 4: Verify the changes
SELECT 
    id,
    sn,
    model,
    status,
    CASE 
        WHEN status = 0 THEN 'Default'
        WHEN status = 1 THEN 'Normal (Active)'
        WHEN status = 2 THEN 'Disabled'
        WHEN status = 3 THEN 'Inactive/Scrapped'
        WHEN status = 4 THEN 'Unregistered'
        WHEN status = 5 THEN 'Unauthenticated'
        ELSE 'Unknown'
    END AS status_desc,
    created_at::date AS created_date
FROM device 
WHERE deleted_at IS NULL
ORDER BY id;
