-- 064: 会员退订 MVP — 到期前取消续订标记 + 审计日志
SET search_path TO public;

ALTER TABLE public.user_member
    ADD COLUMN IF NOT EXISTS cancel_pending SMALLINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS cancel_requested_at TIMESTAMP;

COMMENT ON COLUMN public.user_member.cancel_pending IS '1 已发起到期后终止订阅（当前周期内权益仍有效）';
COMMENT ON COLUMN public.user_member.cancel_requested_at IS '用户确认退订时间';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'user_member_cancel_pending_check'
    ) THEN
        ALTER TABLE public.user_member
        ADD CONSTRAINT user_member_cancel_pending_check CHECK (cancel_pending IN (0, 1));
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS public.member_unsubscribe_log (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    user_member_id BIGINT NOT NULL REFERENCES public.user_member(id) ON DELETE CASCADE,
    unsubscribe_type VARCHAR(32) NOT NULL,
    reason_code VARCHAR(32) NOT NULL DEFAULT '',
    feedback TEXT,
    scheduled_expire_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT member_unsubscribe_log_type_check CHECK (unsubscribe_type IN (
        'user_initiated', 'revoke', 'auto_renew_off', 'forced'
    ))
);

CREATE INDEX IF NOT EXISTS idx_member_unsubscribe_log_user_created
    ON public.member_unsubscribe_log (user_id, created_at DESC);

COMMENT ON TABLE public.member_unsubscribe_log IS '会员退订/撤销退订审计';
COMMENT ON COLUMN public.member_unsubscribe_log.unsubscribe_type IS 'user_initiated 用户发起 revoke 撤销 auto_renew_off 关自动续费 forced 强制';
