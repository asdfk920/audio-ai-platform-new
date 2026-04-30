-- 为 content 表添加 audio_key 字段（存储 bcrypt 加密后的密钥）
ALTER TABLE public.content ADD COLUMN IF NOT EXISTS audio_key VARCHAR(255) DEFAULT NULL;
COMMENT ON COLUMN public.content.audio_key IS '音频解密密钥（bcrypt 加密，仅创建时返回明文）';
