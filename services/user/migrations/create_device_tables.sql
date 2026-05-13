-- Migration: 设备绑定相关表结构
-- 用于支持 /api/v1/user/device/bind 接口
-- 适用于 PostgreSQL

DO $$
BEGIN
    RAISE NOTICE '开始执行设备绑定相关表的迁移...';

    -- ============================================================
    -- 1. 创建 device 表（如果不存在）
    -- ============================================================
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'device') THEN
        CREATE TABLE public.device (
            id                  BIGSERIAL PRIMARY KEY,
            sn                  VARCHAR(64) NOT NULL UNIQUE,                    -- 设备序列号（唯一索引）
            product_key         VARCHAR(100) NOT NULL DEFAULT '',              -- 产品Key（用于标识设备型号）
            mac                 VARCHAR(17) NOT NULL DEFAULT '',               -- MAC地址
            firmware_version    VARCHAR(50) NOT NULL DEFAULT '',                -- 固件版本
            hardware_version    VARCHAR(50) NOT NULL DEFAULT '',                -- 硬件版本
            ip                  VARCHAR(45) NOT NULL DEFAULT '',                -- IP地址（支持IPv6）
            status              SMALLINT NOT NULL DEFAULT 0,                   -- 设备状态：0=正常，1=禁用
            online_status       SMALLINT NOT NULL DEFAULT 0,                   -- 在线状态：0=离线，1=在线
            device_secret       VARCHAR(255) NOT NULL DEFAULT '',              -- 设备密钥（用于认证）
            last_active_at      TIMESTAMP WITH TIME ZONE,                      -- 最后活跃时间
            
            -- 绑定相关字段（可选，兼容不同迁移版本）
            bound_user_id       BIGINT DEFAULT NULL,                           -- 绑定的用户ID（NULL=未绑定）
            bound_at            TIMESTAMP WITH TIME ZONE DEFAULT NULL,         -- 绑定时间
            bind_status         SMALLINT DEFAULT 0,                            -- 绑定状态：0=未绑定，1=已绑定
            updated_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            
            created_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW()
        );

        CREATE INDEX idx_device_sn ON public.device(sn);
        CREATE INDEX idx_device_bound_user_id ON public.device(bound_user_id) WHERE bound_user_id IS NOT NULL;
        CREATE INDEX idx_device_product_key ON public.device(product_key);

        COMMENT ON TABLE public.device IS '设备信息表';
        COMMENT ON COLUMN public.device.sn IS '设备序列号（唯一）';
        COMMENT ON COLUMN public.device.bound_user_id IS '绑定的用户ID（0或NULL=未绑定）';
        COMMENT ON COLUMN public.device.bind_status IS '绑定状态：0=未绑定，1=已绑定';
        
        RAISE NOTICE '✅ 已创建 device 表';
    ELSE
        RAISE NOTICE 'ℹ️  device 表已存在，跳过创建';

        -- 检查并添加缺失的列（兼容性处理）
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name = 'device' AND column_name = 'bound_user_id'
        ) THEN
            ALTER TABLE public.device ADD COLUMN bound_user_id BIGINT DEFAULT NULL;
            RAISE NOTICE '✅ device 表已添加 bound_user_id 列';
        END IF;

        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name = 'device' AND column_name = 'bound_at'
        ) THEN
            ALTER TABLE public.device ADD COLUMN bound_at TIMESTAMP WITH TIME ZONE DEFAULT NULL;
            RAISE NOTICE '✅ device 表已添加 bound_at 列';
        END IF;

        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name = 'device' AND column_name = 'bind_status'
        ) THEN
            ALTER TABLE public.device ADD COLUMN bind_status SMALLINT DEFAULT 0;
            RAISE NOTICE '✅ device 表已添加 bind_status 列';
        END IF;

        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name = 'device' AND column_name = 'last_active_at'
        ) THEN
            ALTER TABLE public.device ADD COLUMN last_active_at TIMESTAMP WITH TIME ZONE;
            RAISE NOTICE '✅ device 表已添加 last_active_at 列';
        END IF;
    END IF;

    -- ============================================================
    -- 2. 创建 user_device_bind 表（用户-设备绑定关系表）
    -- ============================================================
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'user_device_bind') THEN
        CREATE TABLE public.user_device_bind (
            id              BIGSERIAL PRIMARY KEY,
            user_id         BIGINT NOT NULL,                                  -- 用户ID
            device_id       BIGINT NOT NULL,                                  -- 设备ID（关联device.id）
            sn              VARCHAR(64) NOT NULL,                             -- 设备SN（冗余存储，便于查询）
            alias           VARCHAR(50) NOT NULL DEFAULT '我的设备',           -- 用户自定义的设备名称
            is_default      SMALLINT NOT NULL DEFAULT 0,                     -- 是否默认设备：0=否，1=是
            bind_type       SMALLINT NOT NULL DEFAULT 1,                     -- 绑定类型：1=主账号，2=共享
            status          SMALLINT NOT NULL DEFAULT 1,                     -- 状态：0=已解绑，1=绑定中
            bound_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),  -- 绑定时间
            unbound_at      TIMESTAMP WITH TIME ZONE,                        -- 解绑时间
            updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            
            CONSTRAINT uk_user_device UNIQUE (user_id, device_id)
        );

        CREATE INDEX idx_user_device_bind_user_id ON public.user_device_bind(user_id) WHERE status = 1;
        CREATE INDEX idx_user_device_bind_device_id ON public.user_device_bind(device_id) WHERE status = 1;
        CREATE INDEX idx_user_device_bind_sn ON public.user_device_bind(sn);

        COMMENT ON TABLE public.user_device_bind IS '用户-设备绑定关系表';
        COMMENT ON COLUMN public.user_device_bind.alias IS '用户自定义的设备名称（最多50字符）';
        COMMENT ON COLUMN public.user_device_bind.bind_type IS '绑定类型：1=主账号绑定，2=通过共享接受';
        COMMENT ON COLUMN public.user_device_bind.status IS '状态：0=已解绑，1=绑定中';
        
        RAISE NOTICE '✅ 已创建 user_device_bind 表';
    ELSE
        RAISE NOTICE 'ℹ️  user_device_bind 表已存在，跳过创建';
    END IF;

    -- ============================================================
    -- 3. 创建 user_device_bind_log 表（绑定操作日志表）
    -- ============================================================
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'user_device_bind_log') THEN
        CREATE TABLE public.user_device_bind_log (
            id           BIGSERIAL PRIMARY KEY,
            user_id      BIGINT NOT NULL,
            device_id    BIGINT NOT NULL,
            sn           VARCHAR(64) NOT NULL,
            operator     VARCHAR(100) NOT NULL DEFAULT 'system',             -- 操作人标识
            action       VARCHAR(20) NOT NULL,                              -- 操作：BIND/UNBIND/SHARE/REVOKE
            action_time  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
            
            created_at   TIMESTAMP WITH TIME ZONE DEFAULT NOW()
        );

        CREATE INDEX idx_user_device_bind_log_user_id ON public.user_device_bind_log(user_id);
        CREATE INDEX idx_user_device_bind_log_device_id ON public.user_device_bind_log(device_id);
        CREATE INDEX idx_user_device_bind_log_action_time ON public.user_device_bind_log(action_time);

        COMMENT ON TABLE public.user_device_bind_log IS '设备绑定操作日志表';
        COMMENT ON COLUMN public.user_device_bind_log.action IS '操作类型：BIND=绑定，UNBIND=解绑，SHARE=共享，REVOKE=撤销共享';
        
        RAISE NOTICE '✅ 已创建 user_device_bind_log 表';
    ELSE
        RAISE NOTICE 'ℹ️  user_device_bind_log 表已存在，跳过创建';
    END IF;

    -- ============================================================
    -- 4. 检查并更新 user_profile 表（可选：记录用户设备数等统计信息）
    -- ============================================================
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'user_profile') THEN
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name = 'user_profile' AND column_name = 'device_count'
        ) THEN
            ALTER TABLE public.user_profile ADD COLUMN device_count INTEGER NOT NULL DEFAULT 0;
            COMMENT ON COLUMN public.user_profile.device_count IS '用户已绑定设备数量';
            RAISE NOTICE '✅ user_profile 表已添加 device_count 列';
        END IF;

        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name = 'user_profile' AND column_name = 'last_bind_time'
        ) THEN
            ALTER TABLE public.user_profile ADD COLUMN last_bind_time TIMESTAMP WITH TIME ZONE;
            COMMENT ON COLUMN public.user_profile.last_bind_time IS '最后绑定设备的时间';
            RAISE NOTICE '✅ user_profile 表已添加 last_bind_time 列';
        END IF;
    ELSE
        RAISE NOTICE 'ℹ️  user_profile 表不存在，跳过更新';
    END IF;

    RAISE NOTICE '';
    RAISE NOTICE '🎉 设备绑定相关表迁移完成！';
    RAISE NOTICE '';
    RAISE NOTICE '已准备好的功能：';
    RAISE NOTICE '  ✅ device 表 - 存储设备基本信息和绑定状态';
    RAISE NOTICE '  ✅ user_device_bind 表 - 记录用户与设备的绑定关系';
    RAISE NOTICE '  ✅ user_device_bind_log 表 - 记录所有绑定操作日志';
    RAISE NOTICE '  ✅ 支持接口: POST /api/v1/user/device/bind';

EXCEPTION WHEN OTHERS THEN
    RAISE EXCEPTION '❌ 迁移失败: %', SQLERRM;
END $$;

-- 验证迁移结果
SELECT 
    table_name,
    CASE 
        WHEN table_name = 'device' THEN '设备信息表'
        WHEN table_name = 'user_device_bind' THEN '用户-设备绑定关系表'
        WHEN table_name = 'user_device_bind_log' => '设备绑定操作日志表'
        ELSE '未知表'
    END as table_description,
    (SELECT COUNT(*) FROM information_schema.columns WHERE table_name = t.table_name) as column_count
FROM information_schema.tables t
WHERE table_schema = 'public' 
AND table_name IN ('device', 'user_device_bind', 'user_device_bind_log')
ORDER BY 
    CASE table_name
        WHEN 'device' THEN 1
        WHEN 'user_device_bind' THEN 2
        WHEN 'user_device_bind_log' THEN 3
        ELSE 4
    END;
