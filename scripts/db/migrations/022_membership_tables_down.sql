-- 回滚 022：会员体系表（谨慎：会删除数据）
SET search_path TO public;

ALTER TABLE user_member
    DROP CONSTRAINT IF EXISTS fk_user_member_level_code;

ALTER TABLE user_member
    DROP COLUMN IF EXISTS grant_by,
    DROP COLUMN IF EXISTS register_type,
    DROP COLUMN IF EXISTS is_permanent,
    DROP COLUMN IF EXISTS expire_at,
    DROP COLUMN IF EXISTS level_code;

DROP TABLE IF EXISTS member_level_benefit;
DROP TABLE IF EXISTS member_benefit;
DROP TABLE IF EXISTS member_level;

