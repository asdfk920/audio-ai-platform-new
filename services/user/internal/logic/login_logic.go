package logic

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/pkg/passwd"
	"github.com/jacklau/audio-ai-platform/pkg/redisx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/pkg/logger"
	"github.com/jacklau/audio-ai-platform/services/user/internal/repo/dao"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type LoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 用户登录（邮箱或手机号二选一 + 密码；第三方请走 OAuth 直接登录）
func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LoginLogic) Login(req *types.LoginReq) (resp *types.LoginResp, err error) {
	account, by := l.normalizeAccount(req)
	if account == "" {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "请使用邮箱或手机号其中一种方式登录")
	}
	if req.Password == "" {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "请输入密码")
	}

	repo := l.svcCtx.UserRepo
	var u *dao.UserWithPassword
	if by == "email" {
		u, err = repo.FindByEmailForLogin(l.ctx, req.Email)
	} else {
		u, err = repo.FindByMobileForLogin(l.ctx, req.Mobile)
	}
	if err != nil {
		l.Logger.Errorf("find user: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, err.Error())
	}
	if u == nil {
		return nil, errorx.NewDefaultError(errorx.CodeUserNotFound)
	}

	if u.Password == nil || u.Salt == nil {
		return nil, errorx.NewCodeError(errorx.CodePasswordError, "该账号未设置密码，请使用第三方登录")
	}
	if !passwd.VerifyPassword(*u.Salt, req.Password, *u.Password) {
		return nil, errorx.NewDefaultError(errorx.CodePasswordError)
	}

	// 登录成功，生成 access_token 与 refresh_token
	//
	// 为什么要“两类 token”：
	// - access_token（JWT，短有效期）：每次请求携带，服务端无状态校验，性能高。
	// - refresh_token（随机串，长有效期）：只用于换取新的 access_token，降低 access_token 泄露后的风险窗口。
	//
	// refresh_token 不用 JWT：
	// - 避免把刷新逻辑也做成“可自验证”的 token（更难做吊销/单点登录）。
	// - 随机串配合 Redis 存储，可随时失效、可轮换，安全边界更清晰。
	now := time.Now().Unix()
	expire := l.svcCtx.Config.Auth.AccessExpire
	accessToken, err := l.buildJWT(l.svcCtx.Config.Auth.AccessSecret, now, expire, u.Id)
	if err != nil {
		l.Logger.Errorf("buildJWT: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeSystemError, "生成令牌失败")
	}
	// #region agent log
	logger.AgentNDJSON("H1", "login_logic.go:Login", "issued login access token", map[string]any{
		"userId":          u.Id,
		"accessExpireSec": expire,
		"iat":             now,
		"exp":             now + expire,
		"tokenLen":        len(accessToken),
		"jwtDotCount":     func() int64 { return int64(countDots(accessToken)) }(),
	})
	// #endregion
	refreshToken := uuid.New().String()

	// refresh_token 存 Redis，供后续刷新 access_token（默认 7 天）
	// 这里采用“单点登录”语义：同一用户再次登录会覆盖旧 refresh_token。
	// 好处：用户在新设备登录后可以让旧设备 refresh_token 失效，减少盗用面。
	refreshTTL := 7 * 24 * time.Hour
	// refresh:{token} -> userId
	if err := redisx.Set(l.ctx, l.refreshTokenKey(refreshToken), u.Id, refreshTTL); err != nil {
		l.Logger.Errorf("redis set refresh token: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeRedisError, "保存刷新令牌失败")
	}
	// user:{id}:refresh -> token（便于单点登录/清理旧 token）
	old, _ := redisx.Get(l.ctx, l.userRefreshTokenKey(u.Id))
	if old != "" && old != refreshToken {
		_ = redisx.Del(l.ctx, l.refreshTokenKey(old))
	}
	_ = redisx.Set(l.ctx, l.userRefreshTokenKey(u.Id), refreshToken, refreshTTL)

	return &types.LoginResp{
		UserId:       u.Id,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expire,
	}, nil
}

func (l *LoginLogic) normalizeAccount(req *types.LoginReq) (account, by string) {
	hasEmail := req.Email != ""
	hasMobile := req.Mobile != ""
	if hasEmail && !hasMobile {
		return req.Email, "email"
	}
	if hasMobile && !hasEmail {
		return req.Mobile, "mobile"
	}
	return "", ""
}

func (l *LoginLogic) refreshTokenKey(token string) string {
	return fmt.Sprintf("user:refresh:%s", token)
}

func (l *LoginLogic) userRefreshTokenKey(userId int64) string {
	return fmt.Sprintf("user:%d:refresh", userId)
}

// buildJWT 生成 JWT access_token，供后续请求携带鉴权
func (l *LoginLogic) buildJWT(secret string, iat, seconds, userId int64) (string, error) {
	claims := jwt.MapClaims{
		"exp":    iat + seconds,
		"iat":    iat,
		"userId": userId,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func countDots(token string) int {
	n := 0
	for _, ch := range token {
		if ch == '.' {
			n++
		}
	}
	return n
}
