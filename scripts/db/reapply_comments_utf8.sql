-- 以 UTF-8 重新写入表/列/视图注释（修复 Windows 下管道编码错误导致的 ??? 乱码）
-- 执行: 见 scripts/db/fix-comments-encoding.ps1 或 docker exec + psql

SET client_encoding TO 'UTF8';

-- 来源: 005_user_realname_auth.sql
COMMENT ON COLUMN users.real_name_status IS '实名状态: 0未认证 1已通过 2审核中 3失败';
COMMENT ON COLUMN users.real_name_time IS '最近一次实名通过时间';
COMMENT ON COLUMN users.real_name_type IS '证件类型: 1个人 2企业';
COMMENT ON TABLE user_real_name_auth IS '实名认证提交与审核流水';
COMMENT ON COLUMN user_real_name_auth.auth_status IS '10待三方 11三方通过 12三方拒绝 20待人工 21人工通过 22人工拒绝 30已取消(接口异常回滚)';

-- 来源: 006_user_realname_idcard_photos.sql
COMMENT ON COLUMN user_real_name_auth.id_card_front_ref IS '身份证人像面：对象存储 URL 或服务端占位标记';
COMMENT ON COLUMN user_real_name_auth.id_card_back_ref IS '身份证国徽面：对象存储 URL 或服务端占位标记';

-- 来源: 007_user_cancellation.sql
COMMENT ON COLUMN users.cancellation_cooling_until IS '注销冷静期结束时间；非空且未到期时禁止登录';
COMMENT ON COLUMN users.account_cancelled_at IS '逻辑注销完成时间';
COMMENT ON TABLE user_cancellation_log IS '注销申请流水：1冷静中 2已执行 3已撤销';
COMMENT ON COLUMN user_cancellation_log.status IS '1冷静期 2已注销 3已撤销';

-- 来源: 008_user_center_schema.sql
COMMENT ON TABLE user_auth IS '第三方登录绑定；auth_type=wechat|google 等，auth_id=openid，unionid 微信可选';
COMMENT ON COLUMN user_auth.unionid IS '微信 unionid，可选';
COMMENT ON TABLE user_verify_code IS '验证码流水：scene 如 register/login/reset_pwd/bind；生产建议 code 存哈希或脱敏';
COMMENT ON COLUMN user_verify_code.scene IS 'register|login|reset_pwd|bind 等';
COMMENT ON TABLE user_member IS '会员等级与有效期；无行表示默认免费档';
COMMENT ON COLUMN user_member.status IS '1有效 0暂停等，由业务定义';
COMMENT ON COLUMN user_login_log.login_type IS 'password|sms|wechat|google';
COMMENT ON COLUMN user_login_log.status IS '1成功 2失败';
COMMENT ON TABLE jwt_blacklist IS '失效 access/refresh 指纹；定期按 expired_at 清理';
COMMENT ON VIEW sys_user IS '逻辑用户主表视图；物理表为 public.users（含实名/注销等扩展列未全部展开，直接查 users 可得全量）';
COMMENT ON VIEW sys_role IS '逻辑角色视图；物理表为 public.roles';
COMMENT ON VIEW sys_user_role IS '用户-角色关联视图；物理表为 public.user_role_rel';
