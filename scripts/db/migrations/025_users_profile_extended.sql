-- 用户主表扩展：资料完善、可见性、签名/简介/兴趣/所在地等（注销相关字段已在 007 等迁移中）
ALTER TABLE users ADD COLUMN IF NOT EXISTS constellation VARCHAR(10) NULL;
ALTER TABLE users ADD COLUMN IF NOT EXISTS age SMALLINT NULL;
ALTER TABLE users ADD COLUMN IF NOT EXISTS signature VARCHAR(255) NULL;
ALTER TABLE users ADD COLUMN IF NOT EXISTS bio VARCHAR(1000) NULL;
-- 生日/性别可见性：0=仅自己 1=好友可见 2=公开（NULL 表示未设置策略）
ALTER TABLE users ADD COLUMN IF NOT EXISTS birthday_visibility SMALLINT NULL;
ALTER TABLE users ADD COLUMN IF NOT EXISTS gender_visibility SMALLINT NULL;
-- 资料完整度：0=未完善 1=已完善；分数 0-100 供前端展示
ALTER TABLE users ADD COLUMN IF NOT EXISTS profile_complete SMALLINT NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS profile_complete_score SMALLINT NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS hobbies VARCHAR(255) NULL;
ALTER TABLE users ADD COLUMN IF NOT EXISTS location VARCHAR(100) NULL;

COMMENT ON COLUMN users.constellation IS '星座（可选；亦可由生日推算）';
COMMENT ON COLUMN users.age IS '年龄（可选；建议由生日实时计算，若存库需定期更新）';
COMMENT ON COLUMN users.signature IS '个性签名';
COMMENT ON COLUMN users.bio IS '个人简介';
COMMENT ON COLUMN users.birthday_visibility IS '生日可见性：0仅自己 1好友 2公开';
COMMENT ON COLUMN users.gender_visibility IS '性别可见性：0仅自己 1好友 2公开';
COMMENT ON COLUMN users.profile_complete IS '资料是否已完善：0否 1是';
COMMENT ON COLUMN users.profile_complete_score IS '资料完整度分数 0-100';
COMMENT ON COLUMN users.hobbies IS '兴趣爱好';
COMMENT ON COLUMN users.location IS '所在地';

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_age_check;
ALTER TABLE users ADD CONSTRAINT users_age_check CHECK (age IS NULL OR (age >= 0 AND age <= 150));

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_birthday_visibility_check;
ALTER TABLE users ADD CONSTRAINT users_birthday_visibility_check CHECK (birthday_visibility IS NULL OR (birthday_visibility >= 0 AND birthday_visibility <= 2));

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_gender_visibility_check;
ALTER TABLE users ADD CONSTRAINT users_gender_visibility_check CHECK (gender_visibility IS NULL OR (gender_visibility >= 0 AND gender_visibility <= 2));

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_profile_complete_check;
ALTER TABLE users ADD CONSTRAINT users_profile_complete_check CHECK (profile_complete IN (0, 1));

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_profile_complete_score_check;
ALTER TABLE users ADD CONSTRAINT users_profile_complete_score_check CHECK (profile_complete_score >= 0 AND profile_complete_score <= 100);
