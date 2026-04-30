-- 回滚：移除流媒体表注释（PostgreSQL）
SET search_path TO public;

COMMENT ON TABLE public.stream_channels IS NULL;
COMMENT ON COLUMN public.stream_channels.id IS NULL;
COMMENT ON COLUMN public.stream_channels.channel_id IS NULL;
COMMENT ON COLUMN public.stream_channels.stream_key IS NULL;
COMMENT ON COLUMN public.stream_channels.source_type IS NULL;
COMMENT ON COLUMN public.stream_channels.source_id IS NULL;
COMMENT ON COLUMN public.stream_channels.push_status IS NULL;
COMMENT ON COLUMN public.stream_channels.protocols IS NULL;
COMMENT ON COLUMN public.stream_channels.bitrate IS NULL;
COMMENT ON COLUMN public.stream_channels.created_at IS NULL;
COMMENT ON COLUMN public.stream_channels.updated_at IS NULL;

COMMENT ON TABLE public.push_records IS NULL;
COMMENT ON COLUMN public.push_records.id IS NULL;
COMMENT ON COLUMN public.push_records.push_id IS NULL;
COMMENT ON COLUMN public.push_records.channel_id IS NULL;
COMMENT ON COLUMN public.push_records.stream_key IS NULL;
COMMENT ON COLUMN public.push_records.user_id IS NULL;
COMMENT ON COLUMN public.push_records.status IS NULL;
COMMENT ON COLUMN public.push_records.push_duration IS NULL;
COMMENT ON COLUMN public.push_records.started_at IS NULL;
COMMENT ON COLUMN public.push_records.ended_at IS NULL;
COMMENT ON COLUMN public.push_records.created_at IS NULL;

COMMENT ON TABLE public.cdn_nodes IS NULL;
COMMENT ON COLUMN public.cdn_nodes.id IS NULL;
COMMENT ON COLUMN public.cdn_nodes.node_id IS NULL;
COMMENT ON COLUMN public.cdn_nodes.domain IS NULL;
COMMENT ON COLUMN public.cdn_nodes.status IS NULL;
COMMENT ON COLUMN public.cdn_nodes.health_score IS NULL;
COMMENT ON COLUMN public.cdn_nodes.bandwidth IS NULL;
COMMENT ON COLUMN public.cdn_nodes.created_at IS NULL;
COMMENT ON COLUMN public.cdn_nodes.updated_at IS NULL;

COMMENT ON TABLE public.play_records IS NULL;
COMMENT ON COLUMN public.play_records.id IS NULL;
COMMENT ON COLUMN public.play_records.record_id IS NULL;
COMMENT ON COLUMN public.play_records.channel_id IS NULL;
COMMENT ON COLUMN public.play_records.stream_key IS NULL;
COMMENT ON COLUMN public.play_records.user_id IS NULL;
COMMENT ON COLUMN public.play_records.device_sn IS NULL;
COMMENT ON COLUMN public.play_records.protocol IS NULL;
COMMENT ON COLUMN public.play_records.status IS NULL;
COMMENT ON COLUMN public.play_records.play_duration IS NULL;
COMMENT ON COLUMN public.play_records.cdn_node_id IS NULL;
COMMENT ON COLUMN public.play_records.started_at IS NULL;
COMMENT ON COLUMN public.play_records.ended_at IS NULL;
COMMENT ON COLUMN public.play_records.created_at IS NULL;

COMMENT ON TABLE public.play_quality_logs IS NULL;
COMMENT ON COLUMN public.play_quality_logs.id IS NULL;
COMMENT ON COLUMN public.play_quality_logs.record_id IS NULL;
COMMENT ON COLUMN public.play_quality_logs.device_sn IS NULL;
COMMENT ON COLUMN public.play_quality_logs.ts IS NULL;
COMMENT ON COLUMN public.play_quality_logs.bitrate IS NULL;
COMMENT ON COLUMN public.play_quality_logs.latency_ms IS NULL;
COMMENT ON COLUMN public.play_quality_logs.is_stalling IS NULL;

COMMENT ON TABLE public.stream_logs IS NULL;
COMMENT ON COLUMN public.stream_logs.id IS NULL;
COMMENT ON COLUMN public.stream_logs.log_id IS NULL;
COMMENT ON COLUMN public.stream_logs.log_type IS NULL;
COMMENT ON COLUMN public.stream_logs.user_id IS NULL;
COMMENT ON COLUMN public.stream_logs.channel_id IS NULL;
COMMENT ON COLUMN public.stream_logs.response_code IS NULL;
COMMENT ON COLUMN public.stream_logs.log_time IS NULL;
COMMENT ON COLUMN public.stream_logs.meta IS NULL;

COMMENT ON TABLE public.stream_stats IS NULL;
COMMENT ON COLUMN public.stream_stats.id IS NULL;
COMMENT ON COLUMN public.stream_stats.stat_date IS NULL;
COMMENT ON COLUMN public.stream_stats.stat_hour IS NULL;
COMMENT ON COLUMN public.stream_stats.dimension_type IS NULL;
COMMENT ON COLUMN public.stream_stats.dimension_id IS NULL;
COMMENT ON COLUMN public.stream_stats.play_count IS NULL;
COMMENT ON COLUMN public.stream_stats.play_duration IS NULL;
COMMENT ON COLUMN public.stream_stats.peak_bandwidth IS NULL;
COMMENT ON COLUMN public.stream_stats.avg_latency IS NULL;
COMMENT ON COLUMN public.stream_stats.stall_rate IS NULL;
COMMENT ON COLUMN public.stream_stats.created_at IS NULL;

COMMENT ON TABLE public.stream_auth_configs IS NULL;
COMMENT ON COLUMN public.stream_auth_configs.id IS NULL;
COMMENT ON COLUMN public.stream_auth_configs.config_id IS NULL;
COMMENT ON COLUMN public.stream_auth_configs.auth_type IS NULL;
COMMENT ON COLUMN public.stream_auth_configs.secret_key IS NULL;
COMMENT ON COLUMN public.stream_auth_configs.created_at IS NULL;
COMMENT ON COLUMN public.stream_auth_configs.updated_at IS NULL;

