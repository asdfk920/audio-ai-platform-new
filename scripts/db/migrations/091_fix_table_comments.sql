-- 修复缺失的数据库表注释（乱码/问号）
SET search_path TO public;
SET client_encoding TO 'UTF8';

-- cdn_nodes
COMMENT ON TABLE cdn_nodes IS 'CDN节点表：存储CDN节点地址和配置信息';

-- device_admin_edit_log
COMMENT ON TABLE device_admin_edit_log IS '设备管理编辑日志：记录管理员对设备信息的修改操作';

-- device_shadow_battery
COMMENT ON TABLE device_shadow_battery IS '设备影子电池信息表：记录设备电池相关状态';

-- device_shadow_config
COMMENT ON TABLE device_shadow_config IS '设备影子配置表：存储设备desired/reported配置信息';

-- device_shadow_location
COMMENT ON TABLE device_shadow_location IS '设备影子位置信息表：记录设备地理位置数据';

-- device_shadow_profile
COMMENT ON TABLE device_shadow_profile IS '设备影子配置档案表：存储设备配置档案信息';

-- device_status_alert
COMMENT ON TABLE device_status_alert IS '设备状态告警表：记录设备异常状态告警信息';

-- device_status_logs
COMMENT ON TABLE device_status_logs IS '设备状态日志表：记录设备HTTP状态上报历史';

-- member_package
COMMENT ON TABLE member_package IS '会员套餐表：存储会员套餐配置信息';

-- member_pay_log
COMMENT ON TABLE member_pay_log IS '会员支付日志表：记录会员支付流水信息';

-- order_master
COMMENT ON TABLE order_master IS '订单主表：存储用户订单信息';

-- pay_log
COMMENT ON TABLE pay_log IS '支付日志表：记录支付交易流水';

-- play_quality_logs
COMMENT ON TABLE play_quality_logs IS '播放质量日志表：记录音频播放质量数据';

-- play_records
COMMENT ON TABLE play_records IS '播放记录表：记录用户内容播放历史';

-- push_records
COMMENT ON TABLE push_records IS '推送记录表：记录消息推送历史';

-- stream_auth_configs
COMMENT ON TABLE stream_auth_configs IS '流媒体鉴权配置表：存储流媒体服务鉴权配置';

-- stream_channels
COMMENT ON TABLE stream_channels IS '流媒体频道表：存储直播/点播频道信息';

-- stream_logs
COMMENT ON TABLE stream_logs IS '流媒体日志表：记录流媒体服务运行日志';

-- stream_stats
COMMENT ON TABLE stream_stats IS '流媒体统计表：记录流媒体服务统计数据';

-- sys_job
COMMENT ON TABLE sys_job IS '系统定时任务表：go-admin定时任务配置';

-- user_device_bind_log
COMMENT ON TABLE user_device_bind_log IS '用户设备绑定日志表：记录设备绑定/解绑操作历史';
