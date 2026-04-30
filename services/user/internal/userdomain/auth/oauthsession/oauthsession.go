// Package oauthsession 提供 OAuth 登录后与密码登录一致的 JWT + Redis refresh 签发，以及用户状态校验。
package oauthsession

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/pkg/jwtx"
	"github.com/jacklau/audio-ai-platform/pkg/redisx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/pkg/util/userconst"
	"github.com/jacklau/audio-ai-platform/services/user/internal/repo/dao"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/jacklau/audio-ai-platform/services/user/internal/userdomain/profile/accountcancel"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
)

func refreshTokenRedisKey(token string) string {
	return fmt.Sprintf("user:refresh:%s", token)
}

func userRefreshRedisKey(userID int64) string {
	return fmt.Sprintf("user:%d:refresh", userID)
}

// IssueAccessAndRefresh 签发 access JWT，并将 refresh 写入 Redis（与密码登录一致的轮换策略）。
func IssueAccessAndRefresh(ctx context.Context, svcCtx *svc.ServiceContext, logger logx.Logger, userID int64) (*types.OAuthLoginResp, error) {
	if svcCtx == nil {
		return nil, errorx.NewDefaultError(errorx.CodeSystemError)
	}
	now := time.Now().Unix()
	expire := svcCtx.Config.Auth.AccessExpire
	accessToken, err := jwtx.SignAccessTokenHS256(svcCtx.Config.Auth.AccessSecret, now, expire, userID)
	if err != nil {
		logger.Errorf("jwtx.SignAccessTokenHS256: %v", err)
		return nil, errorx.NewDefaultError(errorx.CodeSystemError)
	}

	refreshToken := uuid.New().String()
	refreshTTL := svcCtx.Config.Login.EffectiveRefreshTTL()
	if err := redisx.Set(ctx, refreshTokenRedisKey(refreshToken), userID, refreshTTL); err != nil {
		logger.Errorf("redis set refresh token: %v", err)
		return nil, errorx.NewDefaultError(errorx.CodeRedisError)
	}
	old, gerr := redisx.Get(ctx, userRefreshRedisKey(userID))
	if gerr != nil && gerr != redis.Nil {
		logger.Errorf("redis get user refresh index: %v", gerr)
		return nil, errorx.NewDefaultError(errorx.CodeRedisError)
	}
	if gerr == redis.Nil {
		old = ""
	}
	if old != "" && old != refreshToken {
		if derr := redisx.Del(ctx, refreshTokenRedisKey(old)); derr != nil {
			logger.Errorf("redis del old refresh token: %v", derr)
			return nil, errorx.NewDefaultError(errorx.CodeRedisError)
		}
	}
	if err := redisx.Set(ctx, userRefreshRedisKey(userID), refreshToken, refreshTTL); err != nil {
		logger.Errorf("redis set user refresh index: %v", err)
		return nil, errorx.NewDefaultError(errorx.CodeRedisError)
	}

	return &types.OAuthLoginResp{
		UserId:       userID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expire,
	}, nil
}

// EnsureUserActive 登录前校验用户存在且 status 为正常。
func EnsureUserActive(ctx context.Context, repo *dao.UserRepo, logger logx.Logger, userID int64) error {
	if repo == nil {
		return errorx.NewDefaultError(errorx.CodeSystemError)
	}
	u, err := repo.FindByID(ctx, userID)
	if err != nil {
		logger.Errorf("FindByID after oauth: %v", err)
		return errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	if u == nil {
		return errorx.NewDefaultError(errorx.CodeUserNotFound)
	}
	if err := accountcancel.ErrIfClosedOrCooling(u); err != nil {
		return err
	}
	if int32(u.Status) != int32(userconst.UserStatusActive) {
		return errorx.NewCodeError(errorx.CodeInvalidParam, "账号状态异常，无法登录")
	}
	return nil
}
