-- 022: 会员体系表（等级/权益/用户会员关系）
SET search_path TO public;

-- 1) 会员等级表
CREATE TABLE IF NOT EXISTS member_level (
    id BIGSERIAL PRIMARY KEY,
    level_code VARCHAR(32) NOT NULL,
    level_name VARCHAR(32) NOT NULL,
    sort INT NOT NULL DEFAULT 1,
    status SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(level_code)
);

COMMENT ON TABLE member_level IS '会员等级表';
COMMENT ON COLUMN member_level.level_code IS '等级编码：ordinary/vip/year_vip/svip 等';
COMMENT ON COLUMN member_level.level_name IS '等级名称';
COMMENT ON COLUMN member_level.sort IS '排序（越小越靠前）';
COMMENT ON COLUMN member_level.status IS '状态：1启用 0禁用';

-- 2) 会员权益表
CREATE TABLE IF NOT EXISTS member_benefit (
    id BIGSERIAL PRIMARY KEY,
    benefit_code VARCHAR(32) NOT NULL,
    benefit_name VARCHAR(32) NOT NULL,
    description VARCHAR(255),
    status SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(benefit_code)
);

COMMENT ON TABLE member_benefit IS '会员权益表（供内容/播放等服务查询）';
COMMENT ON COLUMN member_benefit.benefit_code IS '权益编码：play_high_rate/download_list/ad_free 等';
COMMENT ON COLUMN member_benefit.status IS '状态：1启用 0禁用';

-- 3) 等级-权益关联表
CREATE TABLE IF NOT EXISTS member_level_benefit (
    id BIGSERIAL PRIMARY KEY,
    level_code VARCHAR(32) NOT NULL,
    benefit_code VARCHAR(32) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(level_code, benefit_code),
    CONSTRAINT fk_member_level_benefit_level_code
        FOREIGN KEY(level_code) REFERENCES member_level(level_code) ON DELETE CASCADE,
    CONSTRAINT fk_member_level_benefit_benefit_code
        FOREIGN KEY(benefit_code) REFERENCES member_benefit(benefit_code) ON DELETE CASCADE
);

COMMENT ON TABLE member_level_benefit IS '会员等级-权益关系（可选扩展表）';

-- 4) 用户会员关系表（在既有 user_member 基础上扩展，避免破坏历史）
ALTER TABLE user_member
    ADD COLUMN IF NOT EXISTS level_code VARCHAR(32),
    ADD COLUMN IF NOT EXISTS expire_at TIMESTAMP NULL,
    ADD COLUMN IF NOT EXISTS is_permanent SMALLINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS register_type VARCHAR(32),
    ADD COLUMN IF NOT EXISTS grant_by BIGINT NOT NULL DEFAULT 0;

-- 兼容：若历史列 expired_at 存在但新列 expire_at 为空，可由业务自行回填；此处不做数据迁移以免误覆盖

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE table_schema='public' AND table_name='user_member' AND constraint_name='fk_user_member_level_code'
    ) THEN
        ALTER TABLE user_member
            ADD CONSTRAINT fk_user_member_level_code FOREIGN KEY(level_code)
                REFERENCES member_level(level_code) ON DELETE SET NULL;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_user_member_level_code_status ON user_member(level_code, status);
CREATE INDEX IF NOT EXISTS idx_member_level_benefit_level_code ON member_level_benefit(level_code);

COMMENT ON COLUMN user_member.level_code IS '会员等级编码（新）；与 member_level.level_code 对齐';
COMMENT ON COLUMN user_member.expire_at IS '到期时间（新）；NULL 表示未设置';
COMMENT ON COLUMN user_member.is_permanent IS '是否永久：1永久 0非永久';
COMMENT ON COLUMN user_member.register_type IS '获取方式：register/pay/admin/gift 等';
COMMENT ON COLUMN user_member.grant_by IS '授予人（后台发放等）；0 表示系统';

-- 5) 初始化默认数据（可按需调整）
INSERT INTO member_level (level_code, level_name, sort, status)
VALUES
('ordinary', '普通会员', 1, 1),
('vip', 'VIP', 2, 1),
('year_vip', '年费VIP', 3, 1),
('svip', 'SVIP', 4, 1)
ON CONFLICT (level_code) DO NOTHING;

INSERT INTO member_benefit (benefit_code, benefit_name, description, status)
VALUES
('play_high_rate', '高码率播放', '更高音质/码率播放', 1),
('download_list', '下载清单', '允许离线/批量下载', 1),
('ad_free', '免广告', '去除广告展示', 1)
ON CONFLICT (benefit_code) DO NOTHING;

-- 等级权益绑定（示例：vip 有 2 项，svip 全部）
INSERT INTO member_level_benefit (level_code, benefit_code)
VALUES
('vip', 'play_high_rate'),
('vip', 'download_list'),
('year_vip', 'play_high_rate'),
('year_vip', 'download_list'),
('svip', 'play_high_rate'),
('svip', 'download_list'),
('svip', 'ad_free')
ON CONFLICT (level_code, benefit_code) DO NOTHING;

