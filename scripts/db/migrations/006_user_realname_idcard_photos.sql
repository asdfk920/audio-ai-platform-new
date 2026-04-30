-- 实名流水：身份证人像面/国徽面（或对象存储 URL）与需求文档对齐

ALTER TABLE user_real_name_auth ADD COLUMN IF NOT EXISTS id_card_front_ref TEXT;
ALTER TABLE user_real_name_auth ADD COLUMN IF NOT EXISTS id_card_back_ref TEXT;

COMMENT ON COLUMN user_real_name_auth.id_card_front_ref IS '身份证人像面：对象存储 URL 或服务端占位标记';
COMMENT ON COLUMN user_real_name_auth.id_card_back_ref IS '身份证国徽面：对象存储 URL 或服务端占位标记';
