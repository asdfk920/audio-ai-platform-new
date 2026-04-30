-- 修复会员体系相关表的注释乱码（使用 U& Unicode 转义写入，避免客户端编码影响）
SET search_path TO public;
SET client_encoding TO 'UTF8';

COMMENT ON TABLE member_level IS U&'\4f1a\5458\7b49\7ea7\8868';
COMMENT ON COLUMN member_level.level_code IS U&'\7b49\7ea7\7f16\7801\ff1aordinary/vip/year_vip/svip \7b49';
COMMENT ON COLUMN member_level.level_name IS U&'\7b49\7ea7\540d\79f0';
COMMENT ON COLUMN member_level.sort IS U&'\6392\5e8f\ff08\8d8a\5c0f\8d8a\9760\524d\ff09';
COMMENT ON COLUMN member_level.status IS U&'\72b6\6001\ff1a1\542f\7528 0\7981\7528';

COMMENT ON TABLE member_benefit IS U&'\4f1a\5458\6743\76ca\8868\ff08\4f9b\5185\5bb9/\64ad\653e\7b49\670d\52a1\67e5\8be2\ff09';
COMMENT ON COLUMN member_benefit.benefit_code IS U&'\6743\76ca\7f16\7801\ff1aplay_high_rate/download_list/ad_free \7b49';
COMMENT ON COLUMN member_benefit.benefit_name IS U&'\6743\76ca\540d\79f0';
COMMENT ON COLUMN member_benefit.description IS U&'\6743\76ca\63cf\8ff0';
COMMENT ON COLUMN member_benefit.status IS U&'\72b6\6001\ff1a1\542f\7528 0\7981\7528';

COMMENT ON TABLE member_level_benefit IS U&'\4f1a\5458\7b49\7ea7-\6743\76ca\5173\7cfb\ff08\53ef\9009\6269\5c55\8868\ff09';

