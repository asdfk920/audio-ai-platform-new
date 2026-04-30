package logic

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/pkg/redisx"
	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/streamutil"
	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/types"
)

type PushNotifyLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPushNotifyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PushNotifyLogic {
	return &PushNotifyLogic{ctx: ctx, svcCtx: svcCtx}
}

func (l *PushNotifyLogic) PushNotify(req *types.PushNotifyReq) (*types.PushNotifyResp, error) {
	// 参数校验
	req.StreamKey = streamutil.TrimSpace(req.StreamKey)
	req.Token = streamutil.TrimSpace(req.Token)
	req.ClientIP = streamutil.TrimSpace(req.ClientIP)
	req.ServerIP = streamutil.TrimSpace(req.ServerIP)

	if req.StreamKey == "" || req.Token == "" || req.Expire <= 0 {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "参数错误")
	}

	// 检查过期时间
	now := time.Now().Unix()
	if req.Expire < now {
		return nil, errorx.NewCodeError(errorx.CodeTokenExpired, "Token 已过期")
	}

	// Redis 校验 Token
	if !l.svcCtx.RedisAvailable {
		return nil, errorx.NewCodeError(errorx.CodeSystemError, "Redis 不可用")
	}

	key := streamutil.RedisPushTokenKey(req.StreamKey)
	raw, err := redisx.Get(l.ctx, key)
	if err != nil {
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "Token 无效")
	}

	var meta map[string]any
	if err := json.Unmarshal([]byte(raw), &meta); err != nil {
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "Token 无效")
	}

	// 验证签名
	sig, _ := meta["signature"].(string)
	if sig == "" || sig != req.Token {
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "Token 无效")
	}

	// 获取通道信息
	channel, err := l.getChannelByStreamKey(req.StreamKey)
	if err != nil {
		return nil, errorx.NewCodeError(errorx.CodeResourceNotFound, "通道不存在")
	}

	// 检查通道状态
	if channel.Status == 0 { // 0 表示禁用
		return nil, errorx.NewCodeError(errorx.CodeNoPermission, "通道已禁用")
	}

	// 获取用户 ID
	userID, _ := meta["user_id"].(float64)
	if userID <= 0 {
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "Token 无效")
	}

	// 检查用户账户状态和权限（简化版本）
	if err := l.checkUserPermission(int64(userID), channel.SourceType); err != nil {
		return nil, err
	}

	// 检查推流并发数
	if err := l.checkConcurrentLimit(int64(userID)); err != nil {
		return nil, err
	}

	// 更新通道状态为推流中
	if err := l.updateChannelStatus(channel.ChannelID, req.ClientIP, req.ServerIP); err != nil {
		return nil, errorx.NewCodeError(errorx.CodeSystemError, "更新通道状态失败")
	}

	// 创建推流记录
	pushID, err := l.createPushRecord(channel, int64(userID), req)
	if err != nil {
		return nil, errorx.NewCodeError(errorx.CodeSystemError, "创建推流记录失败")
	}

	// 更新 Redis 缓存状态
	if err := l.updateRedisStatus(req.StreamKey, pushID); err != nil {
		// Redis 更新失败不影响推流，仅记录日志
		fmt.Printf("更新 Redis 状态失败：%v\n", err)
	}

	// 初始化统计信息
	if err := l.initStreamStats(req.StreamKey); err != nil {
		fmt.Printf("初始化统计信息失败：%v\n", err)
	}

	// 记录操作日志
	if err := l.logStreamEvent(int64(userID), channel.ChannelID, req.StreamKey, req.ClientIP, "publish_start"); err != nil {
		fmt.Printf("记录日志失败：%v\n", err)
	}

	// TODO: 通知下游服务（内容服务、AI 推理服务等）
	// 可以使用 Kafka 异步发送通知消息
	// if err := l.notifyDownstreamServices(channel, int64(userID), req); err != nil {
	// 	fmt.Printf("通知下游服务失败：%v\n", err)
	// }

	return &types.PushNotifyResp{
		Code:          0,
		Message:       "允许推流",
		Allowed:       true,
		ChannelID:     channel.ChannelID,
		MaxBitrate:    256,  // 默认最大码率 256kbps
		MaxViewers:    1000, // 默认最大观众数 1000
		RecordEnabled: true, // 默认启用录制
	}, nil
}

// streamChannel 通道信息结构
type streamChannel struct {
	ChannelID  string
	StreamKey  string
	SourceType string
	SourceID   string
	UserID     int64
	Status     int // 0 禁用，1 启用
	PushStatus int // 0 未推流，1 推流中
}

// getChannelByStreamKey 根据 stream_key 获取通道信息
func (l *PushNotifyLogic) getChannelByStreamKey(streamKey string) (*streamChannel, error) {
	const q = `SELECT channel_id, stream_key, source_type, source_id, user_id, status, push_status 
	FROM public.stream_channels WHERE stream_key=$1 LIMIT 1`

	var ch streamChannel
	err := l.svcCtx.DB.QueryRowContext(l.ctx, q, streamKey).Scan(
		&ch.ChannelID, &ch.StreamKey, &ch.SourceType, &ch.SourceID,
		&ch.UserID, &ch.Status, &ch.PushStatus,
	)
	if err == sql.ErrNoRows {
		return nil, errorx.NewCodeError(errorx.CodeResourceNotFound, "通道不存在")
	}
	if err != nil {
		return nil, err
	}
	return &ch, nil
}

// checkUserPermission 检查用户权限
func (l *PushNotifyLogic) checkUserPermission(userID int64, sourceType string) error {
	// TODO: 检查用户账户状态是否正常
	// TODO: 检查用户是否有当前类型的推流权限
	// TODO: 检查会员等级是否满足推流要求
	// 简化版本：暂时不检查
	return nil
}

// checkConcurrentLimit 检查推流并发数
func (l *PushNotifyLogic) checkConcurrentLimit(userID int64) error {
	// TODO: 检查用户当前的推流数量是否达到限制
	// 简化版本：暂时不检查
	return nil
}

// updateChannelStatus 更新通道状态为推流中
func (l *PushNotifyLogic) updateChannelStatus(channelID, clientIP, serverIP string) error {
	const q = `UPDATE public.stream_channels 
	SET push_status=1, push_start_time=NOW(), push_client_ip=$2, push_server_ip=$3, updated_at=NOW() 
	WHERE channel_id=$1`

	_, err := l.svcCtx.DB.ExecContext(l.ctx, q, channelID, clientIP, serverIP)
	return err
}

// createPushRecord 创建推流记录
func (l *PushNotifyLogic) createPushRecord(channel *streamChannel, userID int64, req *types.PushNotifyReq) (string, error) {
	pushID := uuid.New().String()
	const q = `INSERT INTO public.stream_push_records 
	(push_id, channel_id, stream_key, source_type, source_id, user_id, client_ip, server_ip, 
	status, push_start, last_heartbeat, created_at) 
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 0, NOW(), NOW(), NOW())`

	_, err := l.svcCtx.DB.ExecContext(l.ctx, q, pushID, channel.ChannelID, req.StreamKey,
		channel.SourceType, channel.SourceID, userID, req.ClientIP, req.ServerIP)
	if err != nil {
		return "", err
	}
	return pushID, nil
}

// updateRedisStatus 更新 Redis 缓存状态
func (l *PushNotifyLogic) updateRedisStatus(streamKey, pushID string) error {
	key := streamutil.RedisPushTokenKey(streamKey)
	// 先获取现有的 TTL
	ttl, err := redisx.TTL(l.ctx, key)
	if err != nil {
		return err
	}
	// 更新状态为 publishing
	meta := map[string]any{
		"status":     "publishing",
		"push_id":    pushID,
		"push_start": time.Now().Unix(),
		"updated_at": time.Now().Format(time.RFC3339),
	}
	b, _ := json.Marshal(meta)
	// 保持原有 TTL
	if ttl > 0 {
		return redisx.Set(l.ctx, key, string(b), ttl)
	}
	return redisx.Set(l.ctx, key, string(b), 24*time.Hour)
}

// initStreamStats 初始化推流统计信息
func (l *PushNotifyLogic) initStreamStats(streamKey string) error {
	statsKey := "push:stats:" + streamKey
	stats := map[string]int{
		"bytes_sent":   0,
		"bitrate":      0,
		"viewer_count": 0,
	}
	b, _ := json.Marshal(stats)
	return redisx.Set(l.ctx, statsKey, string(b), 24*time.Hour)
}

// logStreamEvent 记录流媒体事件日志
func (l *PushNotifyLogic) logStreamEvent(userID int64, channelID, streamKey, clientIP, eventType string) error {
	const q = `INSERT INTO public.stream_logs 
	(log_type, user_id, channel_id, stream_key, client_ip, event_type, event_time, created_at) 
	VALUES ('push', $1, $2, $3, $4, $5, NOW(), NOW())`

	_, err := l.svcCtx.DB.ExecContext(l.ctx, q, userID, channelID, streamKey, clientIP, eventType)
	return err
}

// notifyDownstreamServices 通知下游服务（预留接口）
func (l *PushNotifyLogic) notifyDownstreamServices(channel *streamChannel, userID int64, req *types.PushNotifyReq) error {
	// TODO: 使用 Kafka 发送通知消息
	// Topic: stream-push-events
	// Message: JSON 格式包含 event_type, channel_id, stream_key, user_id 等
	return nil
}
