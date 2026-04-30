-- 为流媒体表补充注释（PostgreSQL）
SET search_path TO public;

COMMENT ON TABLE public.stream_channels IS '流媒体频道表（推流/播放入口）';
COMMENT ON COLUMN public.stream_channels.id IS '主键';
COMMENT ON COLUMN public.stream_channels.channel_id IS '频道ID（业务唯一）';
COMMENT ON COLUMN public.stream_channels.stream_key IS '推流/播放密钥（唯一）';
COMMENT ON COLUMN public.stream_channels.source_type IS '来源类型（如 device/content/user）';
COMMENT ON COLUMN public.stream_channels.source_id IS '来源ID（对应 source_type）';
COMMENT ON COLUMN public.stream_channels.push_status IS '推流状态（0未知/1在线/2离线等）';
COMMENT ON COLUMN public.stream_channels.protocols IS '支持的协议集合（逗号分隔，如 rtmp,hls,webrtc）';
COMMENT ON COLUMN public.stream_channels.bitrate IS '目标码率/当前码率（kbps）';
COMMENT ON COLUMN public.stream_channels.created_at IS '创建时间';
COMMENT ON COLUMN public.stream_channels.updated_at IS '更新时间';

COMMENT ON TABLE public.push_records IS '推流记录表（每次推流会话）';
COMMENT ON COLUMN public.push_records.id IS '主键';
COMMENT ON COLUMN public.push_records.push_id IS '推流会话ID（唯一）';
COMMENT ON COLUMN public.push_records.channel_id IS '频道ID（外键 -> stream_channels.channel_id）';
COMMENT ON COLUMN public.push_records.stream_key IS '推流使用的 stream_key（冗余）';
COMMENT ON COLUMN public.push_records.user_id IS '推流用户ID（可选）';
COMMENT ON COLUMN public.push_records.status IS '推流状态（0开始/1进行中/2结束/3失败等）';
COMMENT ON COLUMN public.push_records.push_duration IS '推流时长（秒）';
COMMENT ON COLUMN public.push_records.started_at IS '开始时间';
COMMENT ON COLUMN public.push_records.ended_at IS '结束时间';
COMMENT ON COLUMN public.push_records.created_at IS '创建时间';

COMMENT ON TABLE public.cdn_nodes IS 'CDN 节点表';
COMMENT ON COLUMN public.cdn_nodes.id IS '主键';
COMMENT ON COLUMN public.cdn_nodes.node_id IS '节点ID（唯一）';
COMMENT ON COLUMN public.cdn_nodes.domain IS '节点域名';
COMMENT ON COLUMN public.cdn_nodes.status IS '节点状态（1可用/2不可用等）';
COMMENT ON COLUMN public.cdn_nodes.health_score IS '健康评分（0-100）';
COMMENT ON COLUMN public.cdn_nodes.bandwidth IS '带宽容量/当前带宽（bps）';
COMMENT ON COLUMN public.cdn_nodes.created_at IS '创建时间';
COMMENT ON COLUMN public.cdn_nodes.updated_at IS '更新时间';

COMMENT ON TABLE public.play_records IS '播放记录表（每次播放会话）';
COMMENT ON COLUMN public.play_records.id IS '主键';
COMMENT ON COLUMN public.play_records.record_id IS '播放会话ID（唯一）';
COMMENT ON COLUMN public.play_records.channel_id IS '频道ID（外键 -> stream_channels.channel_id）';
COMMENT ON COLUMN public.play_records.stream_key IS '播放使用的 stream_key（冗余）';
COMMENT ON COLUMN public.play_records.user_id IS '播放用户ID（可选）';
COMMENT ON COLUMN public.play_records.device_sn IS '设备序列号（可选）';
COMMENT ON COLUMN public.play_records.protocol IS '播放协议（hls/dash/webrtc/rtmp等）';
COMMENT ON COLUMN public.play_records.status IS '播放状态（0开始/1播放中/2结束/3失败等）';
COMMENT ON COLUMN public.play_records.play_duration IS '播放时长（秒）';
COMMENT ON COLUMN public.play_records.cdn_node_id IS 'CDN 节点ID（外键 -> cdn_nodes.id，可选）';
COMMENT ON COLUMN public.play_records.started_at IS '开始时间';
COMMENT ON COLUMN public.play_records.ended_at IS '结束时间';
COMMENT ON COLUMN public.play_records.created_at IS '创建时间';

COMMENT ON TABLE public.play_quality_logs IS '播放质量日志（播放过程采样）';
COMMENT ON COLUMN public.play_quality_logs.id IS '主键';
COMMENT ON COLUMN public.play_quality_logs.record_id IS '播放会话ID（外键 -> play_records.record_id）';
COMMENT ON COLUMN public.play_quality_logs.device_sn IS '设备序列号（可选）';
COMMENT ON COLUMN public.play_quality_logs.ts IS '采样时间';
COMMENT ON COLUMN public.play_quality_logs.bitrate IS '采样码率（kbps）';
COMMENT ON COLUMN public.play_quality_logs.latency_ms IS '采样延迟（ms）';
COMMENT ON COLUMN public.play_quality_logs.is_stalling IS '是否发生卡顿';

COMMENT ON TABLE public.stream_logs IS '流媒体日志（事件/错误/请求记录）';
COMMENT ON COLUMN public.stream_logs.id IS '主键';
COMMENT ON COLUMN public.stream_logs.log_id IS '日志ID（唯一）';
COMMENT ON COLUMN public.stream_logs.log_type IS '日志类型（push/play/auth/cdn等）';
COMMENT ON COLUMN public.stream_logs.user_id IS '关联用户ID（可选）';
COMMENT ON COLUMN public.stream_logs.channel_id IS '关联频道ID（可选）';
COMMENT ON COLUMN public.stream_logs.response_code IS '响应/错误码（可选）';
COMMENT ON COLUMN public.stream_logs.log_time IS '日志时间';
COMMENT ON COLUMN public.stream_logs.meta IS '扩展元数据（JSON）';

COMMENT ON TABLE public.stream_stats IS '流媒体统计（按小时/维度聚合）';
COMMENT ON COLUMN public.stream_stats.id IS '主键';
COMMENT ON COLUMN public.stream_stats.stat_date IS '统计日期';
COMMENT ON COLUMN public.stream_stats.stat_hour IS '统计小时（0-23）';
COMMENT ON COLUMN public.stream_stats.dimension_type IS '维度类型（channel/user/device/cdn等）';
COMMENT ON COLUMN public.stream_stats.dimension_id IS '维度ID';
COMMENT ON COLUMN public.stream_stats.play_count IS '播放次数';
COMMENT ON COLUMN public.stream_stats.play_duration IS '播放总时长（秒）';
COMMENT ON COLUMN public.stream_stats.peak_bandwidth IS '峰值带宽（bps）';
COMMENT ON COLUMN public.stream_stats.avg_latency IS '平均延迟（ms）';
COMMENT ON COLUMN public.stream_stats.stall_rate IS '卡顿率（0-1）';
COMMENT ON COLUMN public.stream_stats.created_at IS '创建时间';

COMMENT ON TABLE public.stream_auth_configs IS '推流鉴权配置';
COMMENT ON COLUMN public.stream_auth_configs.id IS '主键';
COMMENT ON COLUMN public.stream_auth_configs.config_id IS '配置ID（唯一）';
COMMENT ON COLUMN public.stream_auth_configs.auth_type IS '鉴权类型（token/hmac/whitelist等）';
COMMENT ON COLUMN public.stream_auth_configs.secret_key IS '鉴权密钥';
COMMENT ON COLUMN public.stream_auth_configs.created_at IS '创建时间';
COMMENT ON COLUMN public.stream_auth_configs.updated_at IS '更新时间';

