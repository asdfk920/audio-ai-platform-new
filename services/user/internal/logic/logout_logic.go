package logic

import (
	"context"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/pkg/jwtx"
	authmw "github.com/jacklau/audio-ai-platform/services/user/internal/middleware/auth"
	"github.com/jacklau/audio-ai-platform/services/user/internal/pkg/util/reqguard"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/jacklau/audio-ai-platform/services/user/internal/userdomain/auth/refreshtoken"
	"github.com/zeromicro/go-zero/core/logx"
)

// LogoutLogic 登出：拉黑当前 access jti + 吊销 Redis 中 refresh。
type LogoutLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLogoutLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LogoutLogic {
	return &LogoutLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

// Logout 从 Authorization 解析 access token，写入 jti 黑名单并删除 refresh 索引。
func (l *LogoutLogic) Logout(authorization string) (*types.LogoutResp, error) {
	if err := reqguard.UserRepo(l.ctx, l.svcCtx); err != nil {
		return nil, err
	}
	authz := strings.TrimSpace(authorization)
	const pfx = "Bearer "
	if !strings.HasPrefix(authz, pfx) {
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "请先登录")
	}
	tok := strings.TrimSpace(strings.TrimPrefix(authz, pfx))
	if tok == "" {
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "请先登录")
	}
	secret := l.svcCtx.Config.Auth.AccessSecret
	claims, err := jwtx.ParseAccessToken(secret, tok)
	if err != nil || claims == nil || claims.ID == "" {
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "Token 无效")
	}
	var ttl time.Duration
	if claims.ExpiresAt != nil {
		ttl = time.Until(claims.ExpiresAt.Time)
	}
	if ttl <= 0 {
		ttl = time.Minute
	}
	if !l.svcCtx.Config.Login.JWTBlacklistDisabled {
		if err := authmw.Blacklist(l.ctx, claims.ID, ttl); err != nil {
			l.Logger.Errorf("logout blacklist jti: %v", err)
			return nil, errorx.NewDefaultError(errorx.CodeRedisError)
		}
	}
	if err := refreshtoken.RevokeAllForUser(l.ctx, claims.UserID); err != nil {
		l.Logger.Errorf("logout RevokeAllForUser uid=%d: %v", claims.UserID, err)
		return nil, err
	}
	return &types.LogoutResp{Message: "已退出登录"}, nil
}
