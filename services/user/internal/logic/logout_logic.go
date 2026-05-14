package logic

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/pkg/jwtx"
	"github.com/jacklau/audio-ai-platform/pkg/redisx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/middleware/auth"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type LogoutLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLogoutLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LogoutLogic {
	return &LogoutLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LogoutLogic) Logout(r *http.Request) (resp *types.LogoutResp, err error) {
	token, err := l.extractToken(r)
	if err != nil {
		l.Logger.Errorf("Logout: 提取token失败: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "请先登录")
	}

	claims, parseErr := jwtx.ParseAccessToken(l.svcCtx.Config.Auth.AccessSecret, token)
	if parseErr != nil || claims == nil || claims.ID == "" {
		l.Logger.Errorf("Logout: 解析token失败: %v", parseErr)
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "Token 无效，请重新登录")
	}

	userId := claims.UserID
	jti := claims.ID

	l.Logger.Infof("Logout: 开始退出登录, userId=%d, jti=%s", userId, jti)

	var remainingTTL time.Duration
	if claims.ExpiresAt != nil {
		remainingTTL = time.Until(claims.ExpiresAt.Time)
		if remainingTTL < time.Second {
			remainingTTL = time.Second
		}
	} else {
		remainingTTL = time.Duration(l.svcCtx.Config.Auth.AccessExpire) * time.Second
	}

	if blErr := auth.Blacklist(l.ctx, jti, remainingTTL); blErr != nil {
		l.Logger.Errorf("Logout: 加入黑名单失败, jti=%s, err=%v", jti, blErr)
		return nil, errorx.NewCodeError(errorx.CodeRedisError, "服务暂不可用，请稍后重试")
	}
	l.Logger.Infof("Logout: token已加入黑名单, jti=%s, ttl=%v", jti, remainingTTL)

	refreshKey := fmt.Sprintf("user:%d:refresh", userId)
	oldRefresh, _ := redisx.Get(l.ctx, refreshKey)
	if oldRefresh != "" {
		_ = redisx.Del(l.ctx, fmt.Sprintf("user:refresh:%s", oldRefresh))
		_ = redisx.Del(l.ctx, refreshKey)
		l.Logger.Infof("Logout: 已清理refresh_token, userId=%d", userId)
	}

	l.Logger.Infof("Logout: 退出登录完成, userId=%d", userId)

	return &types.LogoutResp{
		Message: "退出登录成功",
	}, nil
}

func (l *LogoutLogic) extractToken(r *http.Request) (string, error) {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	const pfx = "Bearer "

	if !strings.HasPrefix(authHeader, pfx) {
		return "", fmt.Errorf("missing or invalid Authorization header format")
	}

	tok := strings.TrimSpace(strings.TrimPrefix(authHeader, pfx))
	if tok == "" {
		return "", fmt.Errorf("empty token after Bearer prefix")
	}

	return tok, nil
}
