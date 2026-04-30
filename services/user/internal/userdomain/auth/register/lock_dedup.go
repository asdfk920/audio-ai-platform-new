package register

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/pkg/redisx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/config"
	"github.com/zeromicro/go-zero/core/logx"
)

// WithRegisterTargetLock 同一邮箱/手机互斥注册，避免并发双插；调用方须在流程结束时执行 release（含失败路径）。
func WithRegisterTargetLock(ctx context.Context, target string, cfg config.Register) (release func(), err error) {
	t := strings.TrimSpace(target)
	if t == "" {
		return func() {}, nil
	}
	key := fmt.Sprintf("user:register:lock:%s", t)
	ttl := time.Duration(cfg.EffectiveRegisterLockSeconds()) * time.Second
	ok, rerr := redisx.SetNX(ctx, key, "1", ttl)
	if rerr != nil {
		logx.WithContext(ctx).Errorf("[register] redis setnx lock key=%s: %v", key, rerr)
		return nil, errorx.NewDefaultError(errorx.CodeRedisError)
	}
	if !ok {
		return nil, errorx.NewDefaultError(errorx.CodeRegisterBusy)
	}
	return func() { _ = redisx.Del(ctx, key) }, nil
}

// EnsureSubmitDedup 同 IP + 账号短时间只能提交一次，减轻重放与双击重复注册。
func EnsureSubmitDedup(ctx context.Context, cfg config.Register, clientIP, target string) error {
	key := fmt.Sprintf("user:register:dedup:%s:%s", sanitizeIPKey(clientIP), strings.TrimSpace(target))
	window := time.Duration(cfg.EffectiveSubmitDedupSeconds()) * time.Second
	ok, err := redisx.SetNX(ctx, key, "1", window)
	if err != nil {
		logx.WithContext(ctx).Errorf("[register] redis setnx dedup key=%s: %v", key, err)
		return errorx.NewDefaultError(errorx.CodeRedisError)
	}
	if !ok {
		return errorx.NewDefaultError(errorx.CodeRegisterDuplicateSubmit)
	}
	return nil
}
