-- 为数据库表添加注释
-- 日期：2026-04-14

-- ============================
-- 内容管理相关表
-- ============================

COMMENT ON TABLE IF EXISTS content IS '内容元数据主表：上架内容、下架内容、删除内容';
COMMENT ON TABLE IF EXISTS content_category IS '内容分类表：支持多级分类、音频分类管理';
COMMENT ON TABLE IF EXISTS content_play_records IS '播放记录表：用户播放行为记录、播放统计';
COMMENT ON TABLE IF EXISTS content_tag IS '内容标签表：标签定义、标签元数据';
COMMENT ON TABLE IF EXISTS content_tag_relation IS '内容 - 标签多对多关联表：内容与标签的关联关系';
COMMENT ON TABLE IF EXISTS contents IS '内容主表：可对外展示的内容集合';

-- ============================
-- 设备管理相关表
-- ============================

COMMENT ON TABLE IF EXISTS device IS '设备主表：存储设备信息、设备元数据';
COMMENT ON TABLE IF EXISTS device_certificate IS '设备证书表：双向 TLS 认证、设备证书管理';
COMMENT ON TABLE IF EXISTS device_commands IS '旧设备指令表（历史版本）：已废弃的设备指令记录';
COMMENT ON TABLE IF EXISTS device_event_log IS '设备事件日志（审计/追踪）：设备操作审计日志';
COMMENT ON TABLE IF EXISTS device_group IS '设备分组表（后续扩展）：设备分组管理';
COMMENT ON TABLE IF EXISTS device_instruction IS '设备指令表：App 下发指令、指令队列管理';
COMMENT ON TABLE IF EXISTS device_shadow IS '设备影子表：持久化设备状态、设备状态缓存';
COMMENT ON TABLE IF EXISTS device_state_log IS '设备状态上报日志：在线状态变化记录';
COMMENT ON TABLE IF EXISTS device_user_rel IS '旧用户 - 设备绑定关系表（历史版）：已废弃的用户设备关联';
COMMENT ON TABLE IF EXISTS devices IS '旧设备表（历史版）：已废弃的设备记录';

-- ============================
-- 权限与安全相关表
-- ============================

COMMENT ON TABLE IF EXISTS jwt_blacklist IS '失效 access/refresh token 黑名单：JWT 令牌吊销管理';

-- ============================
-- 会员管理相关表
-- ============================

COMMENT ON TABLE IF EXISTS member_benefit IS '会员权益表（供内容/订单/...使用）：会员权益定义';
COMMENT ON TABLE IF EXISTS member_level IS '会员等级表：会员等级定义、等级配置';
COMMENT ON TABLE IF EXISTS member_level_benefit IS '会员等级 - 权益关系表（多对多）：等级与权益的关联';
COMMENT ON TABLE IF EXISTS member_package IS '会员售卖套餐：会员套餐配置、套餐定价';
COMMENT ON TABLE IF EXISTS member_pay_log IS '会员付费开通/续费审计：会员支付记录、付费流水';

-- ============================
-- 订单管理相关表
-- ============================

COMMENT ON TABLE IF EXISTS order_master IS '订单主表：订单信息、订单管理';

-- ============================
-- OTA 升级相关表
-- ============================

COMMENT ON TABLE IF EXISTS ota_firmware IS 'OTA 固件表：按 product_id 区分的固件版本管理';
COMMENT ON TABLE IF EXISTS ota_upgrade_task IS 'OTA 升级任务表：设备升级任务、升级队列';

-- ============================
-- 支付管理相关表
-- ============================

COMMENT ON TABLE IF EXISTS pay_log IS '支付结果流水：支付交易记录、支付流水';

-- ============================
-- 权限定义相关表
-- ============================

COMMENT ON TABLE IF EXISTS permission_defs IS '权限点定义：细粒度授权、权限点配置';

-- ============================
-- 播放质量相关表
-- ============================

COMMENT ON TABLE IF EXISTS play_quality_logs IS '播放质量日志：播放卡顿、播放失败记录';
COMMENT ON TABLE IF EXISTS play_records IS '播放记录表：用户播放行为、播放历史';

-- ============================
-- 内容处理相关表
-- ============================

COMMENT ON TABLE IF EXISTS processed_contents IS '处理后内容表：AI 处理后的内容、内容处理结果';
COMMENT ON TABLE IF EXISTS push_records IS '推送记录表：内容推送记录、推送统计';
COMMENT ON TABLE IF EXISTS raw_contents IS '原始内容表：用户上传的原始内容、待处理内容';

-- ============================
-- 角色权限相关表
-- ============================

COMMENT ON TABLE IF EXISTS role_permissions IS '角色与权限点多对多关联表：角色权限关联';
COMMENT ON TABLE IF EXISTS roles IS '平台角色表：permission_defs 的分组、角色定义';

-- ============================
-- 流媒体相关表
-- ============================

COMMENT ON TABLE IF EXISTS stream_auth_configs IS '流媒体认证配置表：流媒体鉴权配置';
COMMENT ON TABLE IF EXISTS stream_channels IS '流媒体频道表：频道管理、频道配置';
COMMENT ON TABLE IF EXISTS stream_logs IS '流媒体日志表：流媒体访问日志、播放日志';
COMMENT ON TABLE IF EXISTS stream_stats IS '流媒体统计表：流媒体数据统计、观看统计';

-- ============================
-- 系统配置相关表
-- ============================

COMMENT ON TABLE IF EXISTS sys_config IS 'go-admin 系统配置表：系统参数配置、配置管理';
COMMENT ON TABLE IF EXISTS sys_dept IS 'go-admin 部门表：组织架构、部门管理';
COMMENT ON TABLE IF EXISTS sys_dict_data IS 'go-admin 字典数据表：字典值、数据字典';
COMMENT ON TABLE IF EXISTS sys_dict_type IS 'go-admin 字典类型表：字典分类、字典类型';
COMMENT ON TABLE IF EXISTS sys_menu IS 'go-admin 菜单/路由/权限表：后台菜单管理、路由配置';
COMMENT ON TABLE IF EXISTS sys_migration IS 'go-admin 数据库迁移记录表：迁移历史、版本管理';
COMMENT ON TABLE IF EXISTS sys_post IS 'go-admin 岗位表：岗位管理、职位配置';
COMMENT ON TABLE IF EXISTS sys_role_menu IS 'go-admin 角色菜单关联表：角色与菜单的多对多关联';

-- ============================
-- 用户认证相关表
-- ============================

COMMENT ON TABLE IF EXISTS user_auth IS '第三方登录绑定表：auth 第三方账号关联';
COMMENT ON TABLE IF EXISTS user_cancellation_log IS '注销申请流水表：1 冷静期、2 完成注销';
COMMENT ON TABLE IF EXISTS user_device_bind IS '用户 - 设备绑定表：App 用户与设备关联';
COMMENT ON TABLE IF EXISTS user_download IS '用户下载记录表：支持断点续传、下载管理';
COMMENT ON TABLE IF EXISTS user_favorite IS '用户收藏内容表：用户收藏管理、收藏夹';
COMMENT ON TABLE IF EXISTS user_login_log IS '用户登录日志（审计/安全）：用户登录审计、安全日志';
COMMENT ON TABLE IF EXISTS user_member IS '会员等级与有效期表：无感升级、会员有效期管理';
COMMENT ON TABLE IF EXISTS user_play_record IS '用户播放行为记录表（内部使用）：播放历史、播放统计';
COMMENT ON TABLE IF EXISTS user_real_name_auth IS '实名认证提交与审核流水表：实名认证管理、审核流程';
COMMENT ON TABLE IF EXISTS user_register_events IS '用户注册事件流水表（用于拉新/统计）：注册事件记录';
COMMENT ON TABLE IF EXISTS user_role_rel IS '用户 - 角色关联表：用户与角色的多对多关联';
COMMENT ON TABLE IF EXISTS user_settings IS '用户级配置项表：settings 用户个性化配置';
COMMENT ON TABLE IF EXISTS user_verify_code IS '验证码流水表：scene 如 register/login/...场景验证码';
COMMENT ON TABLE IF EXISTS users IS '用户主表：平台用户信息、用户管理';

-- ============================
-- 其他表
-- ============================

COMMENT ON TABLE IF EXISTS casbin_rule IS 'Casbin 权限规则表：RBAC 权限策略、访问控制规则';
COMMENT ON TABLE IF EXISTS cdn_nodes IS 'CDN 节点表：CDN 节点配置、节点管理';
COMMENT ON TABLE IF EXISTS casbin_rule IS 'Casbin 权限策略表：基于角色的访问控制规则';
