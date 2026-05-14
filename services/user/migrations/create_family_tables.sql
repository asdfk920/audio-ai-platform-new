-- Create family tables for device sharing functionality
-- Required tables: user_family, user_family_member

DO $$
BEGIN
    -- 1. Create user_family table
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'user_family' AND table_schema = 'public') THEN
        CREATE TABLE public.user_family (
            id          BIGSERIAL PRIMARY KEY,
            owner_user_id BIGINT NOT NULL,
            name        VARCHAR(200) DEFAULT '',
            status      SMALLINT NOT NULL DEFAULT 1,
            created_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            updated_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW()
        );

        CREATE INDEX idx_user_family_owner ON public.user_family(owner_user_id);
        
        COMMENT ON TABLE public.user_family IS 'Family group table';
        RAISE NOTICE 'Created user_family table';
    ELSE
        RAISE NOTICE 'user_family table already exists';
    END IF;

    -- 2. Create user_family_member table
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'user_family_member' AND table_schema = 'public') THEN
        CREATE TABLE public.user_family_member (
            id         BIGSERIAL PRIMARY KEY,
            family_id  BIGINT NOT NULL,
            user_id    BIGINT NOT NULL,
            role       VARCHAR(20) NOT NULL DEFAULT 'member',
            status     SMALLINT NOT NULL DEFAULT 1,
            joined_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            invited_by BIGINT DEFAULT 0,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
        );

        CREATE INDEX idx_user_family_member_family ON public.user_family_member(family_id);
        CREATE INDEX idx_user_family_member_user ON public.user_family_member(user_id);
        CREATE INDEX idx_user_family_member_unique ON public.user_family_member(family_id, user_id) WHERE status = 1;
        
        COMMENT ON TABLE public.user_family_member IS 'Family member table';
        RAISE NOTICE 'Created user_family_member table';
    ELSE
        RAISE NOTICE 'user_family_member table already exists';
    END IF;

END $$;
