-- 为 public schema 下缺失的表注释补全（使用 U& Unicode 转义，避免乱码）
SET search_path TO public;
SET client_encoding TO 'UTF8';

COMMENT ON TABLE casbin_rule IS U&'Casbin \6743\9650\89c4\5219\8868\ff08\6743\9650\7b56\7565\6301\4e45\5316\ff09';

COMMENT ON TABLE raw_contents IS U&'\539f\59cb\5185\5bb9\8868\ff1a\7528\6237\4e0a\4f20\7684\97f3\9891/\6587\4ef6\5143\6570\636e';
COMMENT ON TABLE processed_contents IS U&'\5904\7406\540e\5185\5bb9\8868\ff1aAI \5904\7406/\8f6c\7801\7b49\4ea7\51fa\7684\4e2d\95f4\7ed3\679c';
COMMENT ON TABLE contents IS U&'\5185\5bb9\4e3b\8868\ff1a\53ef\5bf9\5916\5c55\793a/\64ad\653e\7684\6700\7ec8\5185\5bb9\8bb0\5f55';
COMMENT ON TABLE content_play_records IS U&'\64ad\653e\8bb0\5f55\8868\ff1a\7528\6237\4e0e\8bbe\5907\7684\64ad\653e\884c\4e3a\65e5\5fd7';

COMMENT ON TABLE sys_config IS U&'go-admin \7cfb\7edf\914d\7f6e\8868';
COMMENT ON TABLE sys_menu IS U&'go-admin \83dc\5355/\8def\7531/\6743\9650\83dc\5355\5b9a\4e49\8868';

COMMENT ON TABLE users IS U&'\7528\6237\4e3b\8868';
COMMENT ON TABLE user_role_rel IS U&'\7528\6237-\89d2\8272\5173\8054\8868';
COMMENT ON TABLE user_register_events IS U&'\7528\6237\6ce8\518c\4e8b\4ef6\6d41\6c34\ff08\7528\4e8e\9632\91cd/\98ce\63a7/\5ba1\8ba1\ff09';
COMMENT ON TABLE user_login_log IS U&'\7528\6237\767b\5f55\65e5\5fd7\ff08\5ba1\8ba1/\6392\67e5\ff09';

