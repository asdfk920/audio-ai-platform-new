-- 修复数据库表注释显示为问号的问题
-- 请在数据库管理工具中直接执行此脚本

-- 内容相关表
COMMENT ON TABLE content IS '内容主表';
COMMENT ON TABLE content_category IS '内容分类表';
COMMENT ON TABLE content_files IS '内容文件表';
COMMENT ON TABLE content_play_records IS '内容播放记录表';
COMMENT ON TABLE content_tag IS '内容标签表';
COMMENT ON TABLE content_tag_relation IS '内容标签关联表';
COMMENT ON TABLE content_tags IS '内容标签表';
COMMENT ON TABLE contents IS '内容表(旧)';
COMMENT ON TABLE processed_contents IS '处理后内容表';
COMMENT ON TABLE raw_contents IS '原始内容表';
COMMENT ON TABLE tags IS '标签表';

-- 用户相关表
COMMENT ON TABLE users IS '用户主表';
COMMENT ON TABLE user_member IS '用户会员表';
COMMENT ON TABLE user_play_record IS '用户播放记录表';
COMMENT ON TABLE user_favorite IS '用户收藏表';
COMMENT ON TABLE user_likes IS '用户点赞表';
COMMENT ON TABLE user_download IS '用户下载记录表(旧)';
COMMENT ON TABLE user_downloads IS '用户下载记录表';
COMMENT ON TABLE user_subscriptions IS '用户订阅表';
COMMENT ON TABLE user_auth IS '用户第三方登录绑定表';
COMMENT ON TABLE user_cancellation_log IS '用户注销日志表';
COMMENT ON TABLE user_device_bind IS '用户设备绑定表';
COMMENT ON TABLE user_device_bind_log IS '用户设备绑定日志表';
COMMENT ON TABLE user_login_log IS '用户登录日志表';
COMMENT ON TABLE user_real_name_auth IS '用户实名认证表';
COMMENT ON TABLE user_register_events IS '用户注册事件表';
COMMENT ON TABLE user_settings IS '用户配置表';
COMMENT ON TABLE user_verify_code IS '验证码表';
COMMENT ON TABLE user_role_rel IS '用户角色关联表';

-- 设备相关表
COMMENT ON TABLE device IS '设备主表';
COMMENT ON TABLE device_activate_nonce IS '设备激活随机数表';
COMMENT ON TABLE device_admin_edit_log IS '设备管理员编辑日志表';
COMMENT ON TABLE device_certificate IS '设备证书表';
COMMENT ON TABLE device_command_schedule IS '设备指令调度表';
COMMENT ON TABLE device_command_schedule_log IS '设备指令调度日志表';
COMMENT ON TABLE device_diagnosis IS '设备诊断表';
COMMENT ON TABLE device_event_log IS '设备事件日志表';
COMMENT ON TABLE device_group IS '设备分组表';
COMMENT ON TABLE device_instruction IS '设备指令表';
COMMENT ON TABLE device_instruction_state_log IS '设备指令状态日志表';
COMMENT ON TABLE device_log IS '设备日志表';
COMMENT ON TABLE device_log_batch IS '设备批量日志表';
COMMENT ON TABLE device_shadow IS '设备影子表';
COMMENT ON TABLE device_shadow_battery IS '设备影子电池表';
COMMENT ON TABLE device_shadow_config IS '设备影子配置表';
COMMENT ON TABLE device_shadow_location IS '设备影子位置表';
COMMENT ON TABLE device_shadow_profile IS '设备影子配置表';
COMMENT ON TABLE device_state_log IS '设备状态日志表';
COMMENT ON TABLE device_status IS '设备状态表';
COMMENT ON TABLE device_status_alert IS '设备状态告警表';
COMMENT ON TABLE device_status_logs IS '设备状态日志表';

-- 系统管理相关表
COMMENT ON TABLE sys_admin IS '系统管理员表';
COMMENT ON TABLE sys_config IS '系统配置表';
COMMENT ON TABLE sys_dept IS '系统部门表';
COMMENT ON TABLE sys_dict_data IS '系统字典数据表';
COMMENT ON TABLE sys_dict_type IS '系统字典类型表';
COMMENT ON TABLE sys_job IS '系统定时任务表';
COMMENT ON TABLE sys_menu IS '系统菜单表';
COMMENT ON TABLE sys_migration IS '系统迁移记录表';
COMMENT ON TABLE sys_post IS '系统岗位表';
COMMENT ON TABLE sys_role_menu IS '系统角色菜单表';

-- 权限相关表
COMMENT ON TABLE casbin_rule IS 'Casbin权限规则表';
COMMENT ON TABLE permission_defs IS '权限定义表';
COMMENT ON TABLE role_permissions IS '角色权限表';
COMMENT ON TABLE roles IS '角色表';
COMMENT ON TABLE jwt_blacklist IS 'JWT黑名单表';

-- 会员订单相关表
COMMENT ON TABLE member_level IS '会员等级表';
COMMENT ON TABLE member_level_benefit IS '会员等级权益表';
COMMENT ON TABLE member_package IS '会员套餐表';
COMMENT ON TABLE order_music IS '音乐订单表';
COMMENT ON TABLE pay_log IS '支付日志表';

-- IoT相关表
COMMENT ON TABLE iot_product IS 'IoT产品表';

-- OTA相关表
COMMENT ON TABLE ota_firmware IS 'OTA固件表';
COMMENT ON TABLE ota_upgrade_task IS 'OTA升级任务表';

-- 流媒体相关表
COMMENT ON TABLE stream_auth_configs IS '流媒体认证配置表';
COMMENT ON TABLE stream_channels IS '流媒体频道表';
COMMENT ON TABLE stream_logs IS '流媒体日志表';
COMMENT ON TABLE stream_stats IS '流媒体统计表';

-- 其他表
COMMENT ON TABLE cdn_nodes IS 'CDN节点表';
COMMENT ON TABLE play_quality_logs IS '播放质量日志表';
COMMENT ON TABLE play_records IS '播放记录表';
COMMENT ON TABLE push_records IS '推送记录表';
