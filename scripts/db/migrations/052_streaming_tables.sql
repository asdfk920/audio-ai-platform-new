-- 流媒体相关表（来自 ER 图：stream_channels / push_records / play_records / play_quality_logs / cdn_nodes / stream_logs / stream_stats / stream_auth_configs）
-- 统一落到 public，避免 search_path 为空导致错误
SET search_path TO public;

-- 1) 推流/播放频道
CREATE TABLE IF NOT EXISTS public.stream_channels (
  id BIGSERIAL PRIMARY KEY,
  channel_id VARCHAR(64) NOT NULL,
  stream_key VARCHAR(255) NOT NULL,
  source_type VARCHAR(32),
  source_id VARCHAR(64),
  push_status SMALLINT DEFAULT 0,
  protocols VARCHAR(128),
  bitrate INTEGER,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(channel_id),
  UNIQUE(stream_key)
);

CREATE INDEX IF NOT EXISTS idx_stream_channels_source ON public.stream_channels(source_type, source_id);

-- 2) 推流记录（1:N stream_channels）
CREATE TABLE IF NOT EXISTS public.push_records (
  id BIGSERIAL PRIMARY KEY,
  push_id VARCHAR(64) NOT NULL,
  channel_id VARCHAR(64) NOT NULL REFERENCES public.stream_channels(channel_id) ON DELETE CASCADE,
  stream_key VARCHAR(255),
  user_id BIGINT,
  status SMALLINT DEFAULT 0,
  push_duration INTEGER,
  started_at TIMESTAMP NULL,
  ended_at TIMESTAMP NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(push_id)
);

CREATE INDEX IF NOT EXISTS idx_push_records_channel_id ON public.push_records(channel_id);
CREATE INDEX IF NOT EXISTS idx_push_records_user_id ON public.push_records(user_id);
CREATE INDEX IF NOT EXISTS idx_push_records_created_at ON public.push_records(created_at);

-- 3) CDN 节点（play_records N:1 cdn_nodes）
CREATE TABLE IF NOT EXISTS public.cdn_nodes (
  id BIGSERIAL PRIMARY KEY,
  node_id VARCHAR(64) NOT NULL,
  domain VARCHAR(255) NOT NULL,
  status SMALLINT DEFAULT 1,
  health_score INTEGER DEFAULT 0,
  bandwidth BIGINT DEFAULT 0,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(node_id)
);

CREATE INDEX IF NOT EXISTS idx_cdn_nodes_domain ON public.cdn_nodes(domain);

-- 4) 播放记录（1:N stream_channels；N:1 cdn_nodes）
CREATE TABLE IF NOT EXISTS public.play_records (
  id BIGSERIAL PRIMARY KEY,
  record_id VARCHAR(64) NOT NULL,
  channel_id VARCHAR(64) NOT NULL REFERENCES public.stream_channels(channel_id) ON DELETE CASCADE,
  stream_key VARCHAR(255),
  user_id BIGINT,
  device_sn VARCHAR(128),
  protocol VARCHAR(32),
  status SMALLINT DEFAULT 0,
  play_duration INTEGER,
  cdn_node_id BIGINT NULL REFERENCES public.cdn_nodes(id) ON DELETE SET NULL,
  started_at TIMESTAMP NULL,
  ended_at TIMESTAMP NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(record_id)
);

CREATE INDEX IF NOT EXISTS idx_play_records_channel_id ON public.play_records(channel_id);
CREATE INDEX IF NOT EXISTS idx_play_records_user_id ON public.play_records(user_id);
CREATE INDEX IF NOT EXISTS idx_play_records_device_sn ON public.play_records(device_sn);
CREATE INDEX IF NOT EXISTS idx_play_records_created_at ON public.play_records(created_at);
CREATE INDEX IF NOT EXISTS idx_play_records_cdn_node_id ON public.play_records(cdn_node_id);

-- 5) 播放质量日志（1:N play_records）
CREATE TABLE IF NOT EXISTS public.play_quality_logs (
  id BIGSERIAL PRIMARY KEY,
  record_id VARCHAR(64) NOT NULL REFERENCES public.play_records(record_id) ON DELETE CASCADE,
  device_sn VARCHAR(128),
  ts TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  bitrate INTEGER,
  latency_ms INTEGER,
  is_stalling BOOLEAN DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS idx_play_quality_logs_record_id ON public.play_quality_logs(record_id);
CREATE INDEX IF NOT EXISTS idx_play_quality_logs_ts ON public.play_quality_logs(ts);

-- 6) 流媒体日志
CREATE TABLE IF NOT EXISTS public.stream_logs (
  id BIGSERIAL PRIMARY KEY,
  log_id VARCHAR(64) NOT NULL,
  log_type VARCHAR(32) NOT NULL,
  user_id BIGINT,
  channel_id VARCHAR(64),
  response_code INTEGER,
  log_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  meta JSONB,
  UNIQUE(log_id)
);

CREATE INDEX IF NOT EXISTS idx_stream_logs_channel_time ON public.stream_logs(channel_id, log_time);
CREATE INDEX IF NOT EXISTS idx_stream_logs_user_time ON public.stream_logs(user_id, log_time);

-- 7) 流媒体统计（按小时/维度聚合）
CREATE TABLE IF NOT EXISTS public.stream_stats (
  id BIGSERIAL PRIMARY KEY,
  stat_date DATE NOT NULL,
  stat_hour SMALLINT NOT NULL,
  dimension_type VARCHAR(32) NOT NULL,
  dimension_id VARCHAR(64) NOT NULL,
  play_count BIGINT DEFAULT 0,
  play_duration BIGINT DEFAULT 0,
  peak_bandwidth BIGINT DEFAULT 0,
  avg_latency INTEGER DEFAULT 0,
  stall_rate NUMERIC(6,4) DEFAULT 0,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(stat_date, stat_hour, dimension_type, dimension_id)
);

CREATE INDEX IF NOT EXISTS idx_stream_stats_dim ON public.stream_stats(dimension_type, dimension_id, stat_date, stat_hour);

-- 8) 推流鉴权配置
CREATE TABLE IF NOT EXISTS public.stream_auth_configs (
  id BIGSERIAL PRIMARY KEY,
  config_id VARCHAR(64) NOT NULL,
  auth_type VARCHAR(32) NOT NULL,
  secret_key VARCHAR(255) NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(config_id)
);

