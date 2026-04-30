-- 用户点赞表
CREATE TABLE IF NOT EXISTS user_likes (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL,
    content_id      BIGINT NOT NULL,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, content_id)
);

CREATE INDEX IF NOT EXISTS idx_user_likes_user_id ON user_likes(user_id);
CREATE INDEX IF NOT EXISTS idx_user_likes_content_id ON user_likes(content_id);

COMMENT ON TABLE user_likes IS '用户点赞表';
COMMENT ON COLUMN user_likes.id IS '主键ID';
COMMENT ON COLUMN user_likes.user_id IS '用户ID';
COMMENT ON COLUMN user_likes.content_id IS '内容ID';
COMMENT ON COLUMN user_likes.created_at IS '点赞时间';
