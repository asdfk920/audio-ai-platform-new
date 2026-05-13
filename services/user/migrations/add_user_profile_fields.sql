-- Migration: 添加用户基础画像字段（birthday, gender, signature）
-- 如果这些列不存在，则添加它们
-- 适用于 PostgreSQL

DO $$
BEGIN
    -- 添加 birthday 字段（生日）
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'users' AND column_name = 'birthday'
    ) THEN
        ALTER TABLE users ADD COLUMN birthday DATE;
        COMMENT ON COLUMN users.birthday IS '用户生日';
        RAISE NOTICE '✅ 已添加 birthday 列';
    ELSE
        RAISE NOTICE 'ℹ️  birthday 列已存在，跳过';
    END IF;

    -- 添加 gender 字段（性别：0=未知，1=男，2=女）
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'users' AND column_name = 'gender'
    ) THEN
        ALTER TABLE users ADD COLUMN gender SMALLINT DEFAULT 0;
        ALTER TABLE users ALTER COLUMN gender SET NOT NULL;
        COMMENT ON COLUMN users.gender IS '性别：0=未知，1=男，2=女';
        RAISE NOTICE '✅ 已添加 gender 列（默认值：0）';
    ELSE
        RAISE NOTICE 'ℹ️  gender 列已存在，跳过';
    END IF;

    -- 添加 signature 字段（个性签名）
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'users' AND column_name = 'signature'
    ) THEN
        ALTER TABLE users ADD COLUMN signature VARCHAR(255) DEFAULT '';
        COMMENT ON COLUMN users.signature IS '个性签名/个人简介';
        RAISE NOTICE '✅ 已添加 signature 列（默认值：空字符串）';
    ELSE
        RAISE NOTICE 'ℹ️  signature 列已存在，跳过';
    END IF;

    -- 可选：添加其他扩展字段（如果不存在）
    
    -- constellation 星座
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'users' AND column_name = 'constellation'
    ) THEN
        ALTER TABLE users ADD COLUMN constellation VARCHAR(20);
        COMMENT ON COLUMN users.constellation IS '星座';
        RAISE NOTICE '✅ 已添加 constellation 列';
    END IF;

    -- age 年龄
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'users' AND column_name = 'age'
    ) THEN
        ALTER TABLE users ADD COLUMN age SMALLINT;
        COMMENT ON COLUMN users.age IS '年龄';
        RAISE NOTICE '✅ 已添加 age 列';
    END IF;

    -- bio 个人简介
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'users' AND column_name = 'bio'
    ) THEN
        ALTER TABLE users ADD COLUMN bio TEXT;
        COMMENT ON COLUMN users.bio IS '详细个人简介';
        RAISE NOTICE '✅ 已添加 bio 列';
    END IF;

    -- hobbies 爱好
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'users' AND column_name = 'hobbies'
    ) THEN
        ALTER TABLE users ADD COLUMN hobbies TEXT;
        COMMENT ON COLUMN users.hobbies IS '兴趣爱好';
        RAISE NOTICE '✅ 已添加 hobbies 列';
    END IF;

    -- location 所在地
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'users' AND column_name = 'location'
    ) THEN
        ALTER TABLE users ADD COLUMN location VARCHAR(100);
        COMMENT ON COLUMN users.location IS '所在地';
        RAISE NOTICE '✅ 已添加 location 列';
    END IF;

    -- birthday_visibility 生日可见性
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'users' AND column_name = 'birthday_visibility'
    ) THEN
        ALTER TABLE users ADD COLUMN birthday_visibility SMALLINT DEFAULT 0;
        COMMENT ON COLUMN users.birthday_visibility IS '生日可见性：0=仅自己，1=好友，2=公开';
        RAISE NOTICE '✅ 已添加 birthday_visibility 列（默认值：0-仅自己可见）';
    END IF;

    -- gender_visibility 性别可见性
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'users' AND column_name = 'gender_visibility'
    ) THEN
        ALTER TABLE users ADD COLUMN gender_visibility SMALLINT DEFAULT 2;
        COMMENT ON COLUMN users.gender_visibility IS '性别可见性：0=仅自己，1=好友，2=公开';
        RAISE NOTICE '✅ 已添加 gender_visibility 列（默认值：2-公开）';
    END IF;

    RAISE NOTICE '';
    RAISE NOTICE '🎉 迁移完成！所有必要的用户资料字段已就绪。';

EXCEPTION WHEN OTHERS THEN
    RAISE EXCEPTION '❌ 迁移失败: %', SQLERRM;
END $$;

-- 验证迁移结果
SELECT 
    column_name,
    data_type,
    is_nullable,
    column_default
FROM information_schema.columns 
WHERE table_name = 'users' 
AND column_name IN ('birthday', 'gender', 'signature', 'constellation', 'age', 'bio', 'hobbies', 'location', 'birthday_visibility', 'gender_visibility')
ORDER BY ordinal_position;
