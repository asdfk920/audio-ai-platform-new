-- 064 down
SET search_path TO public;

DROP TABLE IF EXISTS public.member_unsubscribe_log;

ALTER TABLE public.user_member DROP CONSTRAINT IF EXISTS user_member_cancel_pending_check;
ALTER TABLE public.user_member
    DROP COLUMN IF EXISTS cancel_requested_at,
    DROP COLUMN IF EXISTS cancel_pending;
