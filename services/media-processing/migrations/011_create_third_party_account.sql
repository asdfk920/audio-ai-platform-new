-- 011_create_third_party_account.sql
-- 创建第三方账号表，存储用户通过第三方平台（如网易云音乐）登录的账号信息

-- 1. 创建第三方账号表
CREATE TABLE IF NOT EXISTS public.third_party_account (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT       NOT NULL,                    -- 系统用户 ID（关联 users 表）
    provider        VARCHAR(50)  NOT NULL DEFAULT 'netease',  -- 第三方平台：netease（网易云）、wechat、google 等
    open_id         VARCHAR(100) NOT NULL,                    -- 第三方平台的用户唯一标识
    union_id        VARCHAR(100) DEFAULT '',                  -- 第三方平台的统一用户标识（可选）
    nickname        VARCHAR(100) NOT NULL DEFAULT '',         -- 第三方平台的昵称
    avatar          VARCHAR(500) NOT NULL DEFAULT '',         -- 第三方平台的头像 URL
    gender          SMALLINT      NOT NULL DEFAULT 0,         -- 性别：0-未知，1-男，2-女
    country_code    VARCHAR(10)  NOT NULL DEFAULT '',          -- 国家/地区代码
    province        VARCHAR(50)  NOT NULL DEFAULT '',          -- 省份
    city            VARCHAR(50)  NOT NULL DEFAULT '',          -- 城市
    access_token    TEXT         NOT NULL,                    -- 访问令牌（加密存储）
    refresh_token   TEXT         DEFAULT '',                   -- 刷新令牌（加密存储）
    token_expires_at TIMESTAMP   DEFAULT NULL,                -- access_token 过期时间
    raw_data        JSONB        DEFAULT NULL,                -- 原始返回数据（完整 JSON）
    status          SMALLINT     NOT NULL DEFAULT 1,          -- 状态：1-正常，0-禁用，-1-已解绑
    last_login_at   TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,   -- 最后一次登录时间
    created_at      TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP    NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMP
);

-- 2. 添加表注释
COMMENT ON TABLE public.third_party_account IS '第三方账号绑定表，存储用户通过第三方平台登录的账号信息';
COMMENT ON COLUMN public.third_party_account.id IS '主键 ID';
COMMENT ON COLUMN public.third_party_account.user_id IS '系统用户 ID，关联 users 表';
COMMENT ON COLUMN public.third_party_account.provider IS '第三方平台标识：netease（网易云音乐）、wechat（微信）、google 等';
COMMENT ON COLUMN public.third_party_account.open_id IS '第三方平台的用户唯一标识（如网易云音乐的 userId）';
COMMENT ON COLUMN public.third_party_account.union_id IS '第三方平台的统一用户标识（微信开放平台使用）';
COMMENT ON COLUMN public.third_party_account.nickname IS '第三方平台的用户昵称';
COMMENT ON COLUMN public.third_party_account.avatar IS '第三方平台的用户头像 URL';
COMMENT ON COLUMN public.third_party_account.gender IS '性别：0-未知，1-男，2-女';
COMMENT ON COLUMN public.third_party_account.country_code IS '国家/地区代码（ISO 3166-1 alpha-2）';
COMMENT ON COLUMN public.third_party_account.province IS '省份/州';
COMMENT ON COLUMN public.third_party_account.city IS '城市';
COMMENT ON COLUMN public.third_party_account.access_token IS '访问令牌（AES 加密存储）';
COMMENT ON COLUMN public.third_party_account.refresh_token IS '刷新令牌（AES 加密存储）';
COMMENT ON COLUMN public.third_party_account.token_expires_at IS 'access_token 的过期时间';
COMMENT ON COLUMN public.third_party_account.raw_data IS '原始 API 返回的完整 JSON 数据';
COMMENT ON COLUMN public.third_party_account.status IS '账号状态：1-正常（已绑定），0-禁用，-1-已解绑';
COMMENT ON COLUMN public.third_party_account.last_login_at IS '最后一次使用此第三方账号登录的时间';

-- 3. 创建索引以提高查询性能
CREATE UNIQUE INDEX IF NOT EXISTS idx_third_party_account_user_provider_openid
    ON public.third_party_account(user_id, provider, open_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_third_party_account_provider_openid
    ON public.third_party_account(provider, open_id)
    WHERE deleted_at IS NULL AND status = 1;

CREATE INDEX IF NOT EXISTS idx_third_party_account_user_id
    ON public.third_party_account(user_id)
    WHERE deleted_at IS NULL AND status = 1;

CREATE INDEX IF NOT EXISTS idx_third_party_account_qrcode_key
    ON public.third_party_account((raw_data->>'qrcode_key'))
    WHERE raw_data ? 'qrcode_key' AND deleted_at IS NULL;

-- 4. 添加触发器自动更新 updated_at 字段
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

DROP TRIGGER IF EXISTS update_third_party_account_updated_at ON public.third_party_account;
CREATE TRIGGER update_third_party_account_updated_at
    BEFORE UPDATE ON public.third_party_account
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 5. 验证表是否创建成功
SELECT column_name, data_type, is_nullable, column_default
FROM information_schema.columns
WHERE table_name = 'third_party_account'
  AND table_schema = 'public'
ORDER BY ordinal_position;
