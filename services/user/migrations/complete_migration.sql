-- 完整的数据库迁移脚本（一次性执行所有必要的表结构）
-- 用于解决 "关系不存在" 的SQLSTATE 42P01错误
-- 执行方式: psql -U admin -d audio_platform -f complete_migration.sql

-- ============================================================
-- 1. device 表（如果不存在）
-- ============================================================
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'device') THEN
        CREATE TABLE public.device (
            id                  BIGSERIAL PRIMARY KEY,
            sn                  VARCHAR(64) NOT NULL UNIQUE,
            product_key         VARCHAR(100) NOT NULL DEFAULT '',
            mac                 VARCHAR(17) NOT NULL DEFAULT '',
            firmware_version    VARCHAR(50) NOT NULL DEFAULT '',
            hardware_version    VARCHAR(50) NOT NULL DEFAULT '',
            ip                  VARCHAR(45) NOT NULL DEFAULT '',
            status              SMALLINT NOT NULL DEFAULT 0,
            online_status       SMALLINT NOT NULL DEFAULT 0,
            device_secret       VARCHAR(255) NOT NULL DEFAULT '',
            last_active_at      TIMESTAMP WITH TIME ZONE,
            bound_user_id       BIGINT DEFAULT NULL,
            bound_at            TIMESTAMP WITH TIME ZONE DEFAULT NULL,
            bind_status         SMALLINT DEFAULT 0,
            updated_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            created_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW()
        );

        CREATE INDEX idx_device_sn ON public.device(sn);
        CREATE INDEX idx_device_bound_user_id ON public.device(bound_user_id) WHERE bound_user_id IS NOT NULL;

        RAISE NOTICE '✅ 已创建 device 表';
    ELSE
        RAISE NOTICE 'ℹ️  device 表已存在';

        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'device' AND column_name = 'bound_user_id') THEN
            ALTER TABLE public.device ADD COLUMN bound_user_id BIGINT DEFAULT NULL;
        END IF;
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'device' AND column_name = 'bound_at') THEN
            ALTER TABLE public.device ADD COLUMN bound_at TIMESTAMP WITH TIME ZONE DEFAULT NULL;
        END IF;
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'device' AND column_name = 'bind_status') THEN
            ALTER TABLE public.device ADD COLUMN bind_status SMALLINT DEFAULT 0;
        END IF;
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'device' AND column_name = 'last_active_at') THEN
            ALTER TABLE public.device ADD COLUMN last_active_at TIMESTAMP WITH TIME ZONE;
        END IF;

        RAISE NOTICE '✅ device 表字段检查完成';
    END IF;
END $$;

-- ============================================================
-- 2. user_device_bind 表（用户-设备绑定关系）
-- ============================================================
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'user_device_bind') THEN
        CREATE TABLE public.user_device_bind (
            id              BIGSERIAL PRIMARY KEY,
            user_id         BIGINT NOT NULL,
            device_id       BIGINT NOT NULL,
            sn              VARCHAR(64) NOT NULL,
            alias           VARCHAR(50) NOT NULL DEFAULT '我的设备',
            is_default      SMALLINT NOT NULL DEFAULT 0,
            bind_type       SMALLINT NOT NULL DEFAULT 1,
            status          SMALLINT NOT NULL DEFAULT 1,
            bound_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
            unbound_at      TIMESTAMP WITH TIME ZONE,
            updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            CONSTRAINT uk_user_device UNIQUE (user_id, device_id)
        );

        CREATE INDEX idx_user_device_bind_user_id ON public.user_device_bind(user_id) WHERE status = 1;
        CREATE INDEX idx_user_device_bind_device_id ON public.user_device_bind(device_id) WHERE status = 1;
        CREATE INDEX idx_user_device_bind_sn ON public.user_device_bind(sn);

        RAISE NOTICE '✅ 已创建 user_device_bind 表';
    ELSE
        RAISE NOTICE 'ℹ️  user_device_bind 表已存在';
    END IF;
END $$;

-- ============================================================
-- 3. user_device_bind_log 表（绑定操作日志）
-- ============================================================
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'user_device_bind_log') THEN
        CREATE TABLE public.user_device_bind_log (
            id           BIGSERIAL PRIMARY KEY,
            user_id      BIGINT NOT NULL,
            device_id    BIGINT NOT NULL,
            sn           VARCHAR(64) NOT NULL,
            operator     VARCHAR(100) NOT NULL DEFAULT 'system',
            action       VARCHAR(20) NOT NULL,
            action_time  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
            created_at   TIMESTAMP WITH TIME ZONE DEFAULT NOW()
        );

        CREATE INDEX idx_user_device_bind_log_user_id ON public.user_device_bind_log(user_id);
        CREATE INDEX idx_user_device_bind_log_device_id ON public.user_device_bind_log(device_id);

        RAISE NOTICE '✅ 已创建 user_device_bind_log 表';
    ELSE
        RAISE NOTICE 'ℹ️  user_device_bind_log 表已存在';
    END IF;
END $$;

-- ============================================================
-- 4. user_device_share 表（设备共享）⭐ 这个表缺失导致报错
-- ============================================================
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'user_device_share') THEN
        CREATE TABLE public.user_device_share (
            id              BIGSERIAL PRIMARY KEY,
            sharer_user_id  BIGINT NOT NULL,                    -- 分享者用户ID
            receiver_user_id BIGINT NOT NULL,                   -- 接收者用户ID
            device_id       BIGINT NOT NULL,                    -- 设备ID
            sn              VARCHAR(64) NOT NULL,               -- 设备SN
            status          SMALLINT NOT NULL DEFAULT 0,         -- 状态：0=待接受，1=已接受，2=已拒绝，3=已撤销
            share_type      SMALLINT NOT NULL DEFAULT 1,         -- 共享类型：1=只读，2=可控制
            expire_at       TIMESTAMP WITH TIME ZONE,           -- 过期时间（NULL表示永久）
            created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            accepted_at     TIMESTAMP WITH TIME ZONE,            -- 接受时间
            revoked_at      TIMESTAMP WITH TIME ZONE             -- 撤销时间
        );

        CREATE INDEX idx_user_device_share_sharer ON public.user_device_share(sharer_user_id);
        CREATE INDEX idx_user_device_share_receiver ON public.user_device_share(receiver_user_id);
        CREATE INDEX idx_user_device_share_device ON public.user_device_share(device_id);
        CREATE INDEX idx_user_device_share_status ON public.user_device_share(status) WHERE status IN (0, 1);
        CREATE INDEX idx_user_device_share_expire ON public.user_device_share(expire_at) WHERE expire_at IS NOT NULL;

        COMMENT ON TABLE public.user_device_share IS '设备共享表';
        COMMENT ON COLUMN public.user_device_share.status IS '状态：0=待接受，1=已接受，2=已拒绝，3=已撤销';

        RAISE NOTICE '✅ 已创建 user_device_share 表（解决SQLSTATE 42P01错误）';
    ELSE
        RAISE NOTICE 'ℹ️  user_device_share 表已存在';
    END IF;
END $$;

-- ============================================================
-- 5. 插入测试数据（可选）
-- ============================================================
DO $$
DECLARE
    device_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO device_count FROM public.device WHERE sn = 'AUSP2605000001G2';
    
    IF device_count = 0 THEN
        INSERT INTO public.device (sn, product_key, mac, firmware_version, hardware_version, ip, status, online_status)
        VALUES ('AUSP2605000001G2', 'SmartSpeaker_V1', 'AA:BB:CC:DD:EE:FF', 'V1.2.3', 'HW_V1.0', '192.168.1.100', 0, 1);
        
        RAISE NOTICE '✅ 已插入测试设备: AUSP2605000001G2';
    ELSE
        RAISE NOTICE 'ℹ️  测试设备已存在: AUSP2605000001G2';
    END IF;
END $$;

-- ============================================================
-- 完成
-- ============================================================
DO $$
BEGIN
    RAISE NOTICE '';
    RAISE NOTICE '🎉 数据库迁移完成！';
    RAISE NOTICE '';
    RAISE NOTICE '已创建/验证的表：';
    RAISE NOTICE '  ✅ device (设备信息表)';
    RAISE NOTICE '  ✅ user_device_bind (用户-设备绑定)';
    RAISE NOTICE '  ✅ user_device_bind_log (绑定日志)';
    RAISE NOTICE '  ✅ user_device_share (设备共享) ⭐';
    RAISE NOTICE '';
    RAISE NOTICE '现在可以：';
    RAISE NOTICE '  1. 重启服务';
    RAISE NOTICE '  2. 使用 GET /api/v1/debug/token 诊断Token';
    RAISE NOTICE '  3. 使用 POST /api/v1/user/login 重新登录';
EXCEPTION WHEN OTHERS THEN
    RAISE EXCEPTION '❌ 迁移失败: %', SQLERRM;
END $$;

-- 验证所有表是否都已存在
SELECT 
    table_name,
    CASE 
        WHEN table_name = 'device' THEN '设备信息表'
        WHEN table_name = 'user_device_bind' THEN '用户-设备绑定'
        WHEN table_name = 'user_device_bind_log' THEN '绑定操作日志'
        WHEN table_name = 'user_device_share' THEN '设备共享 ⭐'
        ELSE '其他表'
    END as description,
    (SELECT COUNT(*) FROM information_schema.columns WHERE table_name = t.table_name) as columns
FROM information_schema.tables t
WHERE table_schema = 'public' 
AND table_name IN ('device', 'user_device_bind', 'user_device_bind_log', 'user_device_share')
ORDER BY 
    CASE table_name
        WHEN 'device' THEN 1
        WHEN 'user_device_bind' THEN 2
        WHEN 'user_device_bind_log' THEN 3
        WHEN 'user_device_share' THEN 4
        ELSE 5
    END;
