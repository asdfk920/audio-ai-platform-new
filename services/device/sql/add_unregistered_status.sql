-- Add "Unregistered" status (0) to device table
-- Status values: 0=Unregistered, 1=Normal(Active), 2=Disabled, 3=Inactive/Scrapped

-- Step 1: Drop existing constraint
ALTER TABLE device DROP CONSTRAINT IF EXISTS device_status_check;

-- Step 2: Update constraint to include status 0 (Unregistered)
ALTER TABLE device 
ADD CONSTRAINT device_status_check 
CHECK (status = ANY (ARRAY[0, 1, 2, 3]));

-- Step 3: Verify the constraint change
SELECT conname, pg_get_constraintdef(oid) as definition 
FROM pg_constraint 
WHERE conrelid = 'device'::regclass AND contype = 'c';

-- Step 4: Show current status distribution
SELECT 
    COUNT(*) AS total_devices,
    COUNT(CASE WHEN status = 0 THEN 1 END) AS unregistered_count,
    COUNT(CASE WHEN status = 1 THEN 1 END) AS normal_count,
    COUNT(CASE WHEN status = 2 THEN 1 END) AS disabled_count,
    COUNT(CASE WHEN status = 3 THEN 1 END) AS inactive_count
FROM device 
WHERE deleted_at IS NULL;
