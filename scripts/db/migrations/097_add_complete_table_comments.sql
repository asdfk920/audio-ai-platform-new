-- Migration: 为所有数据库表添加/更新中文注释
-- Version: 097
-- Created: 2026-05-09
-- Description: 统一为所有表添加清晰的中文注释，便于团队理解和维护

SET search_path TO public;
SET client_encoding TO 'UTF8';

-- ============================================================
-- AI Worker 相关表（音频分离服务）
-- ============================================================

COMMENT ON TABLE audio_separation_tasks IS '音频分离任务表：存储用户的AI音轨分离任务记录';
COMMENT ON COLUMN audio_separation_tasks.id IS '主键ID';
COMMENT ON COLUMN audio_separation_tasks.task_id IS '任务唯一标识符（UUID格式）';
COMMENT ON COLUMN audio_separation_tasks.user_id IS '所属用户ID（关联users表）';
COMMENT ON COLUMN audio_separation_tasks.content_id IS '内容ID（关联content表）';
COMMENT ON COLUMN audio_separation_tasks.audio_url IS '原始音频文件URL（支持HTTP/HTTPS/OSS）';
COMMENT ON COLUMN audio_separation_tasks.status IS '任务状态：pending(待处理)/processing(处理中)/completed(已完成)/failed(失败)';
COMMENT ON COLUMN audio_separation_tasks.progress IS '处理进度（0-100百分比）';
COMMENT ON COLUMN audio_separation_tasks.message IS '状态消息或错误信息';
COMMENT ON COLUMN audio_separation_tasks.result_url IS '分离结果ZIP文件下载URL';
COMMENT ON COLUMN audio_separation_tasks.created_at IS '创建时间';
COMMENT ON COLUMN audio_separation_tasks.updated_at IS '最后更新时间';

COMMENT ON TABLE audio_tracks IS '音轨分离结果表：存储AI分离后的各个独立音轨信息';
COMMENT ON COLUMN audio_tracks.id IS '主键ID';
COMMENT ON COLUMN audio_tracks.task_id IS '所属任务ID（关联audio_separation_tasks表）';
COMMENT ON COLUMN audio_tracks.track_name IS '音轨名称：vocals(人声)/drums(鼓点)/bass(贝斯)/other(其他乐器)';
COMMENT ON COLUMN audio_tracks.track_url IS '音轨文件下载URL（OSS或本地路径）';
COMMENT ON COLUMN audio_tracks.file_size IS '文件大小（字节）';
COMMENT ON COLUMN audio_tracks.duration IS '音轨时长（秒）';
COMMENT ON COLUMN audio_tracks.sample_rate IS '采样率（Hz）';
COMMENT ON COLUMN audio_tracks.channels IS '声道数（1=单声道，2=立体声）';
COMMENT ON COLUMN audio_tracks.format IS '音频格式（wav/mp3/flac等）';
COMMENT ON COLUMN audio_tracks.order_index IS '排序索引（用于固定音轨显示顺序）';
COMMENT ON COLUMN audio_tracks.created_at IS '创建时间';

-- ============================================================
-- 内容管理相关表
-- ============================================================

COMMENT ON TABLE artists IS '艺术家/歌手表：存储音乐艺术家、歌手、乐队等信息';
COMMENT ON TABLE content IS '内容数据主表：存储音频/视频等多媒体内容的元数据信息';
COMMENT ON TABLE content_category IS '内容分类表：支持多级分类层级结构';
COMMENT ON TABLE content_download_stats IS '内容下载统计表：记录内容的下载次数和下载用户统计';
COMMENT ON TABLE content_files IS '内容文件表：存储内容的多个版本文件（原始文件、转码文件等）';
COMMENT ON TABLE content_tag IS '内容标签表：定义可用的标签（标签字典）';
COMMENT ON TABLE content_tag_relation IS '内容-标签多对多关联表：实现内容与标签的多对多关系';
COMMENT ON TABLE content_tags IS '内容标签表（旧版）：兼容旧系统，建议使用content_tag';

-- ============================================================
-- 用户相关表
-- ============================================================

COMMENT ON TABLE users IS '用户主表：存储平台注册用户的基本信息和账户数据';
COMMENT ON TABLE user_member IS '用户会员表：记录用户的会员等级、有效期等会员信息';
COMMENT ON TABLE user_play_record IS '用户播放记录表：记录用户的历史播放行为';
COMMENT ON TABLE user_favorite IS '用户收藏表：记录用户收藏的内容列表';
COMMENT ON TABLE user_likes IS '用户点赞表：记录用户对内容的点赞操作';
COMMENT ON TABLE user_downloads IS '用户下载记录表：记录用户的内容下载历史';
COMMENT ON TABLE user_subscriptions IS '用户订阅表：记录用户订阅的艺术家或频道';
COMMENT ON TABLE user_auth IS '用户第三方登录绑定表：存储微信、QQ、Apple等OAuth绑定信息';
COMMENT ON TABLE user_cancellation_log IS '用户注销日志表：记录用户注销账户的操作历史';
COMMENT ON TABLE user_device_bind IS '用户设备绑定表：记录用户与设备的绑定关系';
COMMENT ON TABLE user_device_bind_log IS '用户设备绑定日志表：记录设备绑定的变更历史';
COMMENT ON TABLE user_login_log IS '用户登录日志表：用于安全审计和登录分析';
COMMENT ON TABLE user_real_name_auth IS '用户实名认证表：存储实名认证信息和审核状态';
COMMENT ON TABLE user_register_events IS '用户注册事件表：用于风控和防刷机制';
COMMENT ON TABLE user_settings IS '用户配置表：存储用户的个性化设置偏好';
COMMENT ON TABLE user_verify_code IS '验证码表：存储短信/邮箱验证码及过期时间';
COMMENT ON TABLE user_role_rel IS '用户角色关联表：实现用户与角色的多对多关系';

-- ============================================================
-- 设备相关表（IoT硬件设备）
-- ============================================================

COMMENT ON TABLE device IS '设备主表：存储IoT智能音箱等硬件设备的基本信息';
COMMENT ON TABLE device_activate_nonce IS '设备激活随机数表：用于设备首次激活的安全验证';
COMMENT ON TABLE device_admin_edit_log IS '设备管理员编辑日志表：记录管理员对设备信息的修改操作';
COMMENT ON TABLE device_certificate IS '设备证书表：存储设备的TLS/SSL证书信息';
COMMENT ON TABLE device_command_schedule IS '设备指令调度表：存储待发送给设备的指令队列';
COMMENT ON TABLE device_command_schedule_log IS '设备指令调度日志表：记录指令的执行结果和状态';
COMMENT ON TABLE device_diagnosis IS '设备诊断表：存储设备自检和远程诊断数据';
COMMENT ON TABLE device_event_log IS '设备事件日志表：记录设备上报的各种事件（按键、语音等）';
COMMENT ON TABLE device_group IS '设备分组表：支持对设备进行分组管理';
COMMENT ON TABLE device_instruction IS '设备指令表：定义可发送给设备的指令类型和参数';
COMMENT ON TABLE device_instruction_state_log IS '设备指令状态日志表：追踪指令的生命周期状态变化';
COMMENT ON TABLE device_log IS '设备日志表：设备运行时产生的通用日志';
COMMENT ON TABLE device_log_batch IS '设备批量日志表：批量上传的设备日志数据';
COMMENT ON TABLE device_shadow IS '设备影子表：同步设备 desired/reported 状态的双向同步';
COMMENT ON TABLE device_shadow_battery IS '设备影子电池表：记录设备电池电量、健康度等状态';
COMMENT ON TABLE device_shadow_config IS '设备影子配置表：存储设备的配置参数（音量、WiFi等）';
COMMENT ON TABLE device_shadow_location IS '设备影子位置表：记录设备的GPS地理位置信息';
COMMENT ON TABLE device_shadow_profile IS '设备影子配置档案表：存储设备固件版本、硬件型号等档案信息';
COMMENT ON TABLE device_state_log IS '设备状态日志表：记录设备在线/离线等状态变化';
COMMENT ON TABLE device_status IS '设备状态表：存储设备当前实时状态快照';
COMMENT ON TABLE device_status_alert IS '设备状态告警表：记录设备异常状态的告警规则和通知';
COMMENT ON TABLE device_status_logs IS '设备状态日志表：记录设备HTTP API上报的状态历史数据';

-- ============================================================
-- 系统管理相关表（go-admin后台）
-- ============================================================

COMMENT ON TABLE sys_admin IS '系统管理员表：后台管理系统的管理员账户';
COMMENT ON TABLE sys_config IS '系统配置表：存储平台的全局配置参数';
COMMENT ON TABLE sys_dept IS '系统部门表：组织架构的部门层级结构';
COMMENT ON TABLE sys_dict_data IS '系统字典数据表：字典的具体数据项（如性别：男/女）';
COMMENT ON TABLE sys_dict_type IS '系统字典类型表：字典类型定义（如性别、状态等）';
COMMENT ON TABLE sys_job IS '系统定时任务表：go-admin框架的定时任务调度配置';
COMMENT ON TABLE sys_menu IS '系统菜单表：后台管理的菜单、路由、权限定义';
COMMENT ON TABLE sys_migration IS '系统迁移记录表：数据库版本迁移的执行历史';
COMMENT ON TABLE sys_post IS '系统岗位表：定义工作岗位（如开发工程师、产品经理）';
COMMENT ON TABLE sys_role_menu IS '系统角色菜单表：角色与菜单权限的关联关系';

-- ============================================================
-- 权限与认证相关表
-- ============================================================

COMMENT ON TABLE casbin_rule IS 'Casbin权限规则表：基于Casbin模型的RBAC权限策略存储';
COMMENT ON TABLE permission_defs IS '权限定义表：自定义权限点的定义和描述';
COMMENT ON TABLE role_permissions IS '角色权限表：角色与权限点的关联关系';
COMMENT ON TABLE roles IS '角色表：定义系统角色（如管理员、普通用户、VIP等）';
COMMENT ON TABLE jwt_blacklist IS 'JWT黑名单表：存储已注销的JWT Token防止重放攻击';

-- ============================================================
-- 会员与订单相关表
-- ============================================================

COMMENT ON TABLE member_level IS '会员等级表：定义VIP等级（青铜、白银、黄金、钻石等）';
COMMENT ON TABLE member_level_benefit IS '会员等级权益表：每个等级对应的权益说明';
COMMENT ON TABLE member_package IS '会员套餐表：会员购买套餐的价格和时长配置';
COMMENT ON TABLE order_music IS '音乐订单表：用户购买音乐内容的订单记录';
COMMENT ON TABLE pay_log IS '支付日志表：第三方支付（微信/支付宝）的交易流水记录';

-- ============================================================
-- IoT产品相关表
-- ============================================================

COMMENT ON TABLE iot_product IS 'IoT产品表：定义物联网设备的产品型号和规格';

-- ============================================================
-- OTA升级相关表
-- ============================================================

COMMENT ON TABLE ota_firmware IS 'OTA固件表：存储设备固件版本文件和更新信息';
COMMENT ON TABLE ota_upgrade_task IS 'OTA升级任务表：记录设备固件升级任务的状态和进度';

-- ============================================================
-- 流媒体服务相关表
-- ============================================================

COMMENT ON TABLE stream_auth_configs IS '流媒体鉴权配置表：直播/点播流的访问控制配置';
COMMENT ON TABLE stream_channels IS '流媒体频道表：直播频道或点播频道的配置信息';
COMMENT ON TABLE stream_logs IS '流媒体日志表：流媒体服务的运行和访问日志';
COMMENT ON TABLE stream_stats IS '流媒体统计表：流媒体的带宽、并发、播放量等统计数据';

-- ============================================================
-- CDN与网络相关表
-- ============================================================

COMMENT ON TABLE cdn_nodes IS 'CDN节点配置表：CDN分发节点的地址和配置参数';

-- ============================================================
-- 播放与质量监控相关表
-- ============================================================

COMMENT ON TABLE play_quality_logs IS '播放质量日志表：记录音频播放的卡顿、缓冲等质量指标';
COMMENT ON TABLE play_records IS '播放记录表：用户播放内容的详细历史记录';
COMMENT ON TABLE push_records IS '推送记录表：APP消息推送的发送记录和到达率统计';

-- ============================================================
-- 内容处理相关表（AI工作流）
-- ============================================================

COMMENT ON TABLE contents IS '内容表（旧版）：兼容旧系统的内容数据，建议使用content表';
COMMENT ON TABLE processed_contents IS '处理后内容表：AI处理（转码、分离、增强）后的中间结果';
COMMENT ON TABLE raw_contents IS '原始内容表：用户上传的未处理的原始音频/视频文件元数据';
COMMENT ON TABLE tags IS '标签表（旧版）：兼容旧系统的标签数据，建议使用content_tag表';
COMMENT ON TABLE content_play_records IS '内容播放记录表：基于内容的播放统计分析数据';

-- ============================================================
-- 其他辅助表
-- ============================================================

COMMENT ON TABLE device_import_job IS '设备导入任务表：批量导入设备的异步任务记录';
COMMENT ON TABLE member_unsubscribe_log IS '会员取消订阅日志表：记录用户退订会员的操作历史';

-- ============================================================
-- 完成提示
-- ============================================================

SELECT '✅ 数据库表注释更新完成！' AS status,
       (SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public') AS total_tables,
       (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = 'public') AS total_columns;
