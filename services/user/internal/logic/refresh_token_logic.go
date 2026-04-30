package logic

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/pkg/redisx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
)

type RefreshTokenLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRefreshTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RefreshTokenLogic {
	return &RefreshTokenLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RefreshTokenLogic) RefreshToken(req *types.RefreshTokenReq) (resp *types.RefreshTokenResp, err error) {
	if req == nil || req.RefreshToken == "" {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "缺少 refresh_token")
	}

	// 刷新 token 的核心是：refresh_token 只在服务端可验证（Redis），从而实现“可吊销、可轮换”。
	// 这比把 refresh_token 也做成 JWT 更易控（例如单点登录、主动退出、风控拦截等）。
	val, err := redisx.Get(l.ctx, refreshTokenKey(req.RefreshToken))
	if err != nil {
		if err == redis.Nil {
			return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "refresh_token 无效或已过期，请重新登录")
		}
		l.Logger.Errorf("redis get refresh: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeRedisError, "刷新失败")
	}

	userId, err := strconv.ParseInt(val, 10, 64)
	if err != nil || userId <= 0 {
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "refresh_token 无效，请重新登录")
	}

	// 生成新 access_token（access expire 仍用配置）
	// 为什么不延长旧 access_token：
	// - access_token 为 JWT，无状态；“续期”语义用刷新接口显式完成更清晰。
	// - 便于前端在 401/过期时统一走刷新逻辑，然后重放业务请求。
	now := time.Now().Unix()
	expire := l.svcCtx.Config.Auth.AccessExpire
	accessToken, err := buildJWT(l.svcCtx.Config.Auth.AccessSecret, now, expire, userId)
	if err != nil {
		l.Logger.Errorf("buildJWT: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeSystemError, "生成令牌失败")
	}

	// refresh_token 轮换：旧的删掉，生成新的并写回
	// 轮换原因：防止 refresh_token 长期不变被窃取后可一直刷新（缩小被盗用窗口）。
	newRefresh := uuid.New().String()
	refreshTTL := 7 * 24 * time.Hour
	_ = redisx.Del(l.ctx, refreshTokenKey(req.RefreshToken))
	if err := redisx.Set(l.ctx, refreshTokenKey(newRefresh), userId, refreshTTL); err != nil {
		l.Logger.Errorf("redis set refresh: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeRedisError, "保存刷新令牌失败")
	}
	_ = redisx.Set(l.ctx, userRefreshTokenKey(userId), newRefresh, refreshTTL)

	return &types.RefreshTokenResp{
		AccessToken:  accessToken,
		RefreshToken: newRefresh,
		ExpiresIn:    expire,
	}, nil
}

func refreshTokenKey(token string) string {
	return fmt.Sprintf("user:refresh:%s", token)
}

func userRefreshTokenKey(userId int64) string {
	return fmt.Sprintf("user:%d:refresh", userId)
}

func buildJWT(secret string, iat, seconds, userId int64) (string, error) {
	claims := jwt.MapClaims{
		"exp":    iat + seconds,
		"iat":    iat,
		"userId": userId,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
