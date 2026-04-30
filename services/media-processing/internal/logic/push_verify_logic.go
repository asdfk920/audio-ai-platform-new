package logic

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/pkg/redisx"
	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/streamutil"
	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/types"
)

type PushVerifyLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPushVerifyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PushVerifyLogic {
	return &PushVerifyLogic{ctx: ctx, svcCtx: svcCtx}
}

func (l *PushVerifyLogic) PushVerify(req *types.PushVerifyReq) (*types.PushVerifyResp, error) {
	req.StreamKey = strings.TrimSpace(req.StreamKey)
	req.Token = strings.TrimSpace(req.Token)
	if req.StreamKey == "" || req.Token == "" || req.Expire <= 0 {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "参数错误")
	}
	now := time.Now().Unix()
	if req.Expire < now {
		return nil, errorx.NewCodeError(errorx.CodeTokenExpired, "Token 已过期")
	}

	// Redis 校验
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
	sig, _ := meta["signature"].(string)
	if sig == "" || sig != req.Token {
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "Token 无效")
	}

	// 可选：来源匹配
	if st, _ := meta["source_type"].(string); req.SourceType != "" && st != "" && req.SourceType != st {
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "Token 无效")
	}
	if sid, _ := meta["source_id"].(string); req.SourceID != "" && sid != "" && req.SourceID != sid {
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "Token 无效")
	}

	// 更新通道状态为推流中
	_ = l.setPushing(req.StreamKey)

	return &types.PushVerifyResp{Allowed: true, Msg: "ok"}, nil
}

func (l *PushVerifyLogic) setPushing(streamKey string) error {
	const q = `UPDATE public.stream_channels SET push_status=1, updated_at=NOW() WHERE stream_key=$1`
	_, err := l.svcCtx.DB.ExecContext(l.ctx, q, streamKey)
	if err == nil {
		return nil
	}
	if err == sql.ErrNoRows {
		return nil
	}
	return err
}
