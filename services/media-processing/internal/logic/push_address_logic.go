package logic

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/pkg/jwtx"
	"github.com/jacklau/audio-ai-platform/pkg/redisx"
	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/streamutil"
	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/types"
)

type PushAddressLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPushAddressLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PushAddressLogic {
	return &PushAddressLogic{ctx: ctx, svcCtx: svcCtx}
}

func (l *PushAddressLogic) PushAddress(req *types.PushAddressReq, r *http.Request) (*types.PushAddressResp, error) {
	req.SourceType = strings.TrimSpace(req.SourceType)
	req.SourceID = strings.TrimSpace(req.SourceID)
	if req.SourceType == "" || req.SourceID == "" {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "source_type/source_id 不能为空")
	}
	switch req.SourceType {
	case "content", "ai_inference":
	default:
		return nil, errorx.NewCodeError(errorx.CodeInvalidSourceType, "无效的推流来源类型")
	}
	if len(req.Protocols) == 0 {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "protocols 不能为空")
	}

	// JWT：从 Authorization Bearer 获取 userId
	uid, err := l.userIDFromJWT(r)
	if err != nil {
		return nil, err
	}

	expiresSec := req.Expires
	if expiresSec <= 0 {
		expiresSec = l.svcCtx.Config.Stream.DefaultExpiresSec
		if expiresSec <= 0 {
			expiresSec = 180
		}
	}
	now := time.Now()
	expireAt := now.Add(time.Duration(expiresSec) * time.Second)

	// 业务校验（最小实现）：避免重复推流（同 source_type/source_id 正在推）
	if pushing, err := l.existsActiveChannel(req.SourceType, req.SourceID); err != nil {
		return nil, errorx.NewCodeError(errorx.CodeSystemError, "服务器内部错误")
	} else if pushing {
		return nil, errorx.NewCodeError(errorx.CodeAlreadyPushing, "该资源已在推流中")
	}

	streamKey := streamutil.BuildStreamKey(req.SourceType, req.SourceID, now)
	channelID := "ch_" + streamKey

	// secret_key 优先取 DB stream_auth_configs(default)，否则用配置
	secretKey, err := l.getSecretKey()
	if err != nil {
		return nil, errorx.NewCodeError(errorx.CodeSystemError, "服务器内部错误")
	}
	ts := now.Unix()
	signMsg := streamKey + ":" + itoa64(ts)
	token := streamutil.HMACSHA256Hex(secretKey, signMsg)

	// 组装推流地址（query: token/expire/type/timestamp）
	pushURL := strings.TrimRight(l.svcCtx.Config.Stream.RTMPBaseURL, "/") + "/" + streamKey +
		"?token=" + token + "&expire=" + itoa64(expireAt.Unix()) + "&type=" + req.SourceType + "&timestamp=" + itoa64(ts)

	// DB：创建通道记录
	if err := l.insertStreamChannel(channelID, streamKey, req.SourceType, req.SourceID, pushURL, req.Protocols); err != nil {
		return nil, errorx.NewCodeError(errorx.CodeSystemError, "服务器内部错误")
	}

	// Redis：缓存 token（hash + TTL）
	if !l.svcCtx.RedisAvailable {
		return nil, errorx.NewCodeError(errorx.CodeSystemError, "Redis 不可用")
	}
	redisKey := streamutil.RedisPushTokenKey(streamKey)
	meta := map[string]any{
		"user_id":     uid,
		"source_type": req.SourceType,
		"source_id":   req.SourceID,
		"signature":   token,
		"expire_time": expireAt.Unix(),
		"timestamp":   ts,
		"status":      "active",
		"created_at":  now.Format(time.RFC3339),
	}
	b, _ := json.Marshal(meta)
	if err := redisx.Set(l.ctx, redisKey, string(b), time.Duration(expiresSec)*time.Second); err != nil {
		// DB 已写入时，redis 失败仍返回错误（可按需补偿/删除通道）
		return nil, errorx.NewCodeError(errorx.CodeSystemError, "服务器内部错误")
	}

	rtmpURL := strings.TrimRight(l.svcCtx.Config.Stream.RTMPBaseURL, "/") + "/" + streamKey
	flvURL := ""
	if strings.TrimSpace(l.svcCtx.Config.Stream.FLVBaseURL) != "" {
		flvURL = strings.TrimRight(l.svcCtx.Config.Stream.FLVBaseURL, "/") + "/" + streamKey + ".flv"
	}

	return &types.PushAddressResp{
		ChannelID:  channelID,
		StreamKey:  streamKey,
		PushURL:    pushURL,
		ExpiresIn:  expiresSec,
		ExpireTime: expireAt.Format(time.RFC3339),
		Protocols:  req.Protocols,
		RTMPURL:    rtmpURL,
		FLVURL:     flvURL,
	}, nil
}

func (l *PushAddressLogic) userIDFromJWT(r *http.Request) (int64, error) {
	secret := l.svcCtx.Config.Auth.AccessSecret
	if secret == "" {
		return 0, errorx.NewCodeError(errorx.CodeTokenInvalid, "Token 无效")
	}
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	const pfx = "Bearer "
	if len(auth) < len(pfx) || !strings.EqualFold(auth[:len(pfx)], pfx) {
		return 0, errorx.NewCodeError(errorx.CodeTokenInvalid, "Token 无效")
	}
	tok := strings.TrimSpace(auth[len(pfx):])
	claims, err := jwtx.ParseAccessToken(secret, tok)
	if err != nil {
		// 过期/非法统一按文案映射
		if strings.Contains(strings.ToLower(err.Error()), "expired") {
			return 0, errorx.NewCodeError(errorx.CodeTokenExpired, "Token 已过期")
		}
		return 0, errorx.NewCodeError(errorx.CodeTokenInvalid, "Token 无效")
	}
	if claims.UserID <= 0 {
		return 0, errorx.NewCodeError(errorx.CodeTokenInvalid, "Token 无效")
	}
	return claims.UserID, nil
}

func (l *PushAddressLogic) existsActiveChannel(sourceType, sourceID string) (bool, error) {
	// push_status=1 视为推流中
	const q = `SELECT 1 FROM public.stream_channels WHERE source_type=$1 AND source_id=$2 AND push_status=1 LIMIT 1`
	var one int
	err := l.svcCtx.DB.QueryRowContext(l.ctx, q, sourceType, sourceID).Scan(&one)
	if err == nil {
		return true, nil
	}
	if err == sql.ErrNoRows {
		return false, nil
	}
	return false, err
}

func (l *PushAddressLogic) insertStreamChannel(channelID, streamKey, sourceType, sourceID, pushURL string, protocols []string) error {
	protoStr := strings.Join(protocols, ",")
	const q = `INSERT INTO public.stream_channels(channel_id, stream_key, stream_type, source_type, source_id, push_url, push_status, protocols, bitrate, auth_type, status, created_at, updated_at)
VALUES ($1,$2,'live',$3,$4,$5,0,$6,128,'token',1,NOW(),NOW())`
	_, err := l.svcCtx.DB.ExecContext(l.ctx, q, channelID, streamKey, sourceType, sourceID, pushURL, protoStr)
	return err
}

func (l *PushAddressLogic) getSecretKey() (string, error) {
	cfgID := strings.TrimSpace(l.svcCtx.Config.Stream.DefaultConfigID)
	if cfgID == "" {
		cfgID = "default"
	}
	const q = `SELECT secret_key FROM public.stream_auth_configs WHERE config_id=$1 LIMIT 1`
	var s string
	err := l.svcCtx.DB.QueryRowContext(l.ctx, q, cfgID).Scan(&s)
	if err == nil && strings.TrimSpace(s) != "" {
		return s, nil
	}
	if err != nil && err != sql.ErrNoRows {
		return "", err
	}
	return l.svcCtx.Config.Stream.DefaultSecretKey, nil
}

func itoa64(v int64) string {
	return strconv.FormatInt(v, 10)
}
