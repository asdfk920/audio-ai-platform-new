// Package realnamerebind 实名换绑辅助：防重复提交、身份核验失败计数与临时锁定（Redis）。
package realnamerebind

import (
	"context"
	"fmt"
	"time"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/pkg/redisx"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	dedupTTLSeconds  = 5
	failCountWindow  = 10 * time.Minute
	maxIdentityFails = 5
	identityLockTTL  = 30 * time.Minute
)

func dedupKey(userID int64) string {
	return fmt.Sprintf("user:%d:realname_rebind:dedup", userID)
}

func identityFailKey(userID int64) string {
	return fmt.Sprintf("user:%d:realname_rebind:identity_fail", userID)
}

func identityLockKey(userID int64) string {
	return fmt.Sprintf("user:%d:realname_rebind:identity_lock", userID)
}

// EnsureDedup 同一用户 5 秒内仅处理一次提交，减轻网络重放与双击。
func EnsureDedup(ctx context.Context, userID int64) error {
	if ctx == nil || userID <= 0 {
		return nil
	}
	ok, err := redisx.SetNX(ctx, dedupKey(userID), "1", time.Duration(dedupTTLSeconds)*time.Second)
	if err != nil {
		logx.WithContext(ctx).Errorf("realname_rebind dedup SetNX: %v", err)
		return errorx.NewDefaultError(errorx.CodeRedisError)
	}
	if !ok {
		return errorx.NewCodeError(errorx.CodeRegisterDuplicateSubmit, "操作过于频繁，请稍后再试")
	}
	return nil
}

// EnsureIdentityNotLocked 身份核验失败达上限后的冷静期（与登录锁定区分，专用本接口）。
func EnsureIdentityNotLocked(ctx context.Context, userID int64) error {
	if ctx == nil || userID <= 0 {
		return nil
	}
	n, err := redisx.Exists(ctx, identityLockKey(userID))
	if err != nil {
		logx.WithContext(ctx).Errorf("realname_rebind lock Exists: %v", err)
		return errorx.NewDefaultError(errorx.CodeRedisError)
	}
	if n > 0 {
		return errorx.NewDefaultError(errorx.CodeRealNameRebindLocked)
	}
	return nil
}

// RecordIdentityFailure 姓名/证号/密码/解密失败等累计；窗口内超过阈值则写入锁定键。
func RecordIdentityFailure(ctx context.Context, userID int64) {
	if ctx == nil || userID <= 0 {
		return
	}
	key := identityFailKey(userID)
	n, err := redisx.Incr(ctx, key)
	if err != nil {
		logx.WithContext(ctx).Errorf("realname_rebind fail Incr: %v", err)
		return
	}
	if n == 1 {
		if e := redisx.Expire(ctx, key, failCountWindow); e != nil {
			logx.WithContext(ctx).Errorf("realname_rebind fail Expire: %v", e)
		}
	}
	if n >= maxIdentityFails {
		if e := redisx.Set(ctx, identityLockKey(userID), "1", identityLockTTL); e != nil {
			logx.WithContext(ctx).Errorf("realname_rebind lock Set: %v", e)
		}
		_ = redisx.Del(ctx, key)
		logx.WithContext(ctx).Infof("[SECURITY_AUDIT] severity=high event=realname_rebind_identity_lock user_id=%d fails=%d", userID, n)
	}
}

// ResetIdentityGuard 换绑成功或流程正常结束后清理失败计数与锁定（若有）。
func ResetIdentityGuard(ctx context.Context, userID int64) {
	if ctx == nil || userID <= 0 {
		return
	}
	_ = redisx.Del(ctx, identityFailKey(userID), identityLockKey(userID))
}
