-- ============================================
-- 音频路径更新脚本
-- 将 audio_url 从 HTTP URL 改为本地文件路径
-- ============================================

-- 方案 A：Docker 部署（推荐）
-- 将所有 /audio/xxx 路径映射到容器内的 /app/audio-files/

UPDATE content
SET audio_url = REPLACE(audio_url, '/audio/', '/app/audio-files/')
WHERE audio_url LIKE '/audio/%'
  OR audio_url LIKE 'http%/audio/%';

-- 验证更新结果
SELECT id, title, audio_url 
FROM content 
WHERE audio_url IS NOT NULL 
ORDER BY id 
LIMIT 10;

-- ============================================
-- 如果是 Windows 本地开发，使用这个方案：
-- ============================================

-- UPDATE content
-- SET audio_url = REPLACE(audio_url, '/audio/', './audio-files/')
-- WHERE audio_url LIKE '/audio/%'
--   OR audio_url LIKE 'http%/audio/%';
