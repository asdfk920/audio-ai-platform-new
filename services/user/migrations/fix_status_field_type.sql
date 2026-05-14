-- Fix: Change status field from SMALLINT to VARCHAR to match code usage
-- Code uses string values: pending, active, rejected, revoked, expired, quit

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'user_device_share' AND column_name = 'status' AND table_schema = 'public') THEN
        -- Check current type
        IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'user_device_share' AND column_name = 'status' AND data_type = 'smallint') THEN
            -- Migrate existing numeric values to string values
            ALTER TABLE public.user_device_share ALTER COLUMN status TYPE VARCHAR(20) USING CASE status 
                WHEN 0 THEN 'pending'
                WHEN 1 THEN 'active'
                WHEN 2 THEN 'rejected'
                WHEN 3 THEN 'revoked'
                ELSE 'pending'
            END;
            
            -- Set default value
            ALTER TABLE public.user_device_share ALTER COLUMN status SET DEFAULT 'pending';
            
            RAISE NOTICE 'status field migrated from SMALLINT to VARCHAR successfully';
        ELSE
            RAISE NOTICE 'status field is already VARCHAR, no migration needed';
        END IF;
    ELSE
        RAISE NOTICE 'status field does not exist';
    END IF;
END $$;
