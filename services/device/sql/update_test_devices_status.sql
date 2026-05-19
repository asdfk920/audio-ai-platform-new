-- Update test devices to default status=1 (Normal)
-- Business rule: New devices are created with status=1 by default
-- After registration call, status changes to 5 (Unauthenticated)
-- After WebSocket authentication, status returns to 1 (Normal)

UPDATE device SET 
    status = 1,  -- Default to Normal for all new devices
    updated_at = NOW()
WHERE deleted_at IS NULL;

-- Verify the update
SELECT 
    id,
    sn,
    model,
    status,
    CASE 
        WHEN status = 0 THEN 'Default'
        WHEN status = 1 THEN 'Normal (Default)'
        WHEN status = 2 THEN 'Disabled'
        WHEN status = 3 THEN 'Inactive'
        WHEN status = 4 THEN 'Unregistered'
        WHEN status = 5 THEN 'Unauthenticated'
        ELSE 'Unknown'
    END AS status_desc,
    created_at::date AS created_date
FROM device 
WHERE deleted_at IS NULL
ORDER BY id;
