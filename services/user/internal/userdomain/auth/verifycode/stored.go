package verifycode

import (
	"context"
	"errors"
	"strings"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/pkg/redisx"
	"github.com/redis/go-redis/v9"
)

// CheckStoredEquals 校验 Redis 中验证码是否与提交一致（不删除；成功后由业务调用 DeleteStored）。
func CheckStoredEquals(ctx context.Context, target, submitted string) error {
	submitted = strings.TrimSpace(submitted)
	if submitted == "" {
		return errorx.NewDefaultError(errorx.CodeVerifyCodeInvalid)
	}
	stored, err := redisx.Get(ctx, CodeKey(target))
	if err != nil {
		if err == redis.Nil {
			return errorx.NewDefaultError(errorx.CodeVerifyCodeInvalid)
		}
		return errorx.NewDefaultError(errorx.CodeRedisError)
	}
	if stored != submitted {
		return errorx.NewDefaultError(errorx.CodeVerifyCodeInvalid)
	}
	return nil
}

// CheckStoredEqualsOrBurn 校验验证码；仅在「验证码错误或已过期」时删除 Redis 键，防止对同一码反复试错；Redis 故障等不删键。
func CheckStoredEqualsOrBurn(ctx context.Context, target, submitted string) error {
	err := CheckStoredEquals(ctx, target, submitted)
	if err == nil {
		return nil
	}
	var ce *errorx.CodeError
	if errors.As(err, &ce) && ce.GetCode() == errorx.CodeVerifyCodeInvalid {
		_ = DeleteStored(ctx, target)
	}
	return err
}

// CheckStoredEqualsAlwaysBurn 读取 Redis 验证码并比对；只要成功读到存储值，无论对错都会在返回前删除键，防止重复利用。
func CheckStoredEqualsAlwaysBurn(ctx context.Context, target, submitted string) error {
	submitted = strings.TrimSpace(submitted)
	if submitted == "" {
		return errorx.NewDefaultError(errorx.CodeVerifyCodeInvalid)
	}
	stored, err := redisx.Get(ctx, CodeKey(target))
	if err != nil {
		if err == redis.Nil {
			return errorx.NewDefaultError(errorx.CodeVerifyCodeInvalid)
		}
		return errorx.NewDefaultError(errorx.CodeRedisError)
	}
	defer func() { _ = DeleteStored(ctx, target) }()
	if stored != submitted {
		return errorx.NewDefaultError(errorx.CodeVerifyCodeInvalid)
	}
	return nil
}
