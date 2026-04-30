-- 用户订阅表
CREATE TABLE IF NOT EXISTS user_subscriptions (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL,
    subscribe_type  SMALLINT NOT NULL DEFAULT 1,
    target_id       BIGINT NOT NULL,
    target_name     VARCHAR(255) NOT NULL DEFAULT '',
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, subscribe_type, target_id)
);

CREATE INDEX IF NOT EXISTS idx_user_subscriptions_user_id ON user_subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_subscriptions_type ON user_subscriptions(subscribe_type);
CREATE INDEX IF NOT EXISTS idx_user_subscriptions_target ON user_subscriptions(target_id);

COMMENT ON TABLE user_subscriptions IS '用户订阅表';
COMMENT ON COLUMN user_subscriptions.id IS '主键ID';
COMMENT ON COLUMN user_subscriptions.user_id IS '用户ID';
COMMENT ON COLUMN user_subscriptions.subscribe_type IS '订阅类型：1-艺术家 2-系列';
COMMENT ON COLUMN user_subscriptions.target_id IS '目标ID';
COMMENT ON COLUMN user_subscriptions.target_name IS '目标名称';
COMMENT ON COLUMN user_subscriptions.created_at IS '订阅时间';
