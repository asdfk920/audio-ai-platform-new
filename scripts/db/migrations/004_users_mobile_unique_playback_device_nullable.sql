-- 1) 非空手机号全局唯一（多 NULL 仍允许，与 email UNIQUE 语义对齐）
--    若执行失败，请先清理重复 mobile：SELECT mobile, count(*) FROM users WHERE mobile IS NOT NULL AND trim(mobile) <> '' GROUP BY mobile HAVING count(*) > 1;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_mobile_unique ON users (mobile)
WHERE mobile IS NOT NULL AND trim(mobile) <> '';

-- 2) 播放记录允许无设备（Web 等场景）
ALTER TABLE content_play_records ALTER COLUMN device_id DROP NOT NULL;
