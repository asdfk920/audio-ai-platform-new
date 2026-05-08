-- ============================================
-- 更新音频路径为容器内路径
-- 执行时间: 2026-05-08
-- 用途: 将 audio_url 改为 Docker 容器内路径
-- ============================================

-- ⚡ 核心更新语句（立即执行）
UPDATE content 
SET audio_url = '/audio/test1.mp3' 
WHERE id = 2;

-- ✅ 验证更新结果
SELECT id, title, audio_url, updated_at 
FROM content 
WHERE id = 2;

-- ============================================
-- 备选：如果需要批量更新其他记录
-- ============================================

-- UPDATE content
-- SET audio_url = '/audio/' || title || '.mp3'
-- WHERE audio_url LIKE '%Downloads%'
--   OR audio_url LIKE 'C:\%';
