-- Migration script: Upgrade user_device_share table structure to support new device sharing functionality
-- Issue: Code expects end_at, family_id, etc., but database uses old expire_at field
-- Solution: Add missing fields and rename mismatched fields

DO $$
BEGIN
    -- Check if table exists
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'user_device_share' AND table_schema = 'public') THEN

        -- 1. Add missing new fields (if not exists)
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'user_device_share' AND column_name = 'family_id') THEN
            ALTER TABLE public.user_device_share ADD COLUMN family_id BIGINT DEFAULT NULL;
            COMMENT ON COLUMN public.user_device_share.family_id IS 'Family group ID';
        END IF;

        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'user_device_share' AND column_name = 'device_name') THEN
            ALTER TABLE public.user_device_share ADD COLUMN device_name VARCHAR(200) DEFAULT NULL;
            COMMENT ON COLUMN public.user_device_share.device_name IS 'Device name';
        END IF;

        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'user_device_share' AND column_name = 'owner_user_id') THEN
            ALTER TABLE public.user_device_share ADD COLUMN owner_user_id BIGINT DEFAULT NULL;
            COMMENT ON COLUMN public.user_device_share.owner_user_id IS 'Device owner user ID';
        END IF;

        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'user_device_share' AND column_name = 'shared_user_id') THEN
            ALTER TABLE public.user_device_share ADD COLUMN shared_user_id BIGINT DEFAULT NULL;
            COMMENT ON COLUMN public.user_device_share.shared_user_id IS 'Shared user ID';
        END IF;

        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'user_device_share' AND column_name = 'target_account') THEN
            ALTER TABLE public.user_device_share ADD COLUMN target_account VARCHAR(200) DEFAULT NULL;
            COMMENT ON COLUMN public.user_device_share.target_account IS 'Target account (phone/email)';
        END IF;

        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'user_device_share' AND column_name = 'invite_code') THEN
            ALTER TABLE public.user_device_share ADD COLUMN invite_code VARCHAR(50) DEFAULT NULL;
            COMMENT ON COLUMN public.user_device_share.invite_code IS 'Invite code';
        END IF;

        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'user_device_share' AND column_name = 'permission_level') THEN
            ALTER TABLE public.user_device_share ADD COLUMN permission_level VARCHAR(20) DEFAULT 'view_only';
            COMMENT ON COLUMN public.user_device_share.permission_level IS 'Permission level: full_control/partial/view_only';
        END IF;

        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'user_device_share' AND column_name = 'permission_payload') THEN
            ALTER TABLE public.user_device_share ADD COLUMN permission_payload JSONB DEFAULT '{}';
            COMMENT ON COLUMN public.user_device_share.permission_payload IS 'Permission detail config';
        END IF;

        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'user_device_share' AND column_name = 'start_at') THEN
            ALTER TABLE public.user_device_share ADD COLUMN start_at TIMESTAMP WITH TIME ZONE DEFAULT NULL;
            COMMENT ON COLUMN public.user_device_share.start_at IS 'Share start time';
        END IF;

        -- Add end_at field (key fix!)
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'user_device_share' AND column_name = 'end_at') THEN
            ALTER TABLE public.user_device_share ADD COLUMN end_at TIMESTAMP WITH TIME ZONE DEFAULT NULL;
            COMMENT ON COLUMN public.user_device_share.end_at IS 'Share end/expire time';
            
            -- If expire_at exists, migrate data to end_at
            IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'user_device_share' AND column_name = 'expire_at') THEN
                UPDATE public.user_device_share SET end_at = expire_at WHERE expire_at IS NOT NULL;
            END IF;
        END IF;

        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'user_device_share' AND column_name = 'confirmed_at') THEN
            ALTER TABLE public.user_device_share ADD COLUMN confirmed_at TIMESTAMP WITH TIME ZONE DEFAULT NULL;
            COMMENT ON COLUMN public.user_device_share.confirmed_at IS 'Confirmed accept time';
            
            -- If accepted_at exists, migrate data to confirmed_at
            IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'user_device_share' AND column_name = 'accepted_at') THEN
                UPDATE public.user_device_share SET confirmed_at = accepted_at WHERE accepted_at IS NOT NULL;
            END IF;
        END IF;

        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'user_device_share' AND column_name = 'created_by') THEN
            ALTER TABLE public.user_device_share ADD COLUMN created_by BIGINT DEFAULT NULL;
            COMMENT ON COLUMN public.user_device_share.created_by IS 'Creator user ID';
        END IF;

        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'user_device_share' AND column_name = 'remark') THEN
            ALTER TABLE public.user_device_share ADD COLUMN remark VARCHAR(500) DEFAULT NULL;
            COMMENT ON COLUMN public.user_device_share.remark IS 'Remark info';
        END IF;

        -- 2. Data migration: Copy old field data to new fields
        -- Migrate from sharer_user_id to owner_user_id
        IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'user_device_share' AND column_name = 'sharer_user_id') THEN
            UPDATE public.user_device_share SET owner_user_id = sharer_user_id WHERE owner_user_id IS NULL AND sharer_user_id IS NOT NULL;
        END IF;

        -- Migrate from receiver_user_id to shared_user_id
        IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'user_device_share' AND column_name = 'receiver_user_id') THEN
            UPDATE public.user_device_share SET shared_user_id = receiver_user_id WHERE shared_user_id IS NULL AND receiver_user_id IS NOT NULL;
        END IF;

        -- Migrate from sn to device_sn (if field name differs)
        IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'user_device_share' AND column_name = 'sn') AND 
           NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'user_device_share' AND column_name = 'device_sn') THEN
            ALTER TABLE public.user_device_share RENAME COLUMN sn TO device_sn;
        END IF;

        -- 3. Modify status field type (from SMALLINT to VARCHAR)
        -- Note: This requires adding a new column, migrating data, then dropping old column (if allowed)
        
        -- Create indexes to optimize query performance
        CREATE INDEX IF NOT EXISTS idx_user_device_share_end_at ON public.user_device_share(end_at) WHERE end_at IS NOT NULL;
        CREATE INDEX IF NOT EXISTS idx_user_device_share_family_id ON public.user_device_share(family_id);
        CREATE INDEX IF NOT EXISTS idx_user_device_share_owner_user_id ON public.user_device_share(owner_user_id);
        CREATE INDEX IF NOT EXISTS idx_user_device_share_shared_user_id ON public.user_device_share(shared_user_id);
        CREATE INDEX IF NOT EXISTS idx_user_device_share_invite_code ON public.user_device_share(invite_code) WHERE invite_code IS NOT NULL;

        RAISE NOTICE 'user_device_share table structure upgrade completed! All necessary fields added';

    ELSE
        RAISE NOTICE 'user_device_share table does not exist, please create table first';
    END IF;
END $$;
