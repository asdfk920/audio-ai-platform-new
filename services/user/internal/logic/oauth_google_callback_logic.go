package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const authTypeGoogle = "google"

type googleUserInfo struct {
	Sub      string `json:"sub"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Picture  string `json:"picture"`
	Verified bool   `json:"email_verified"`
}

type OauthGoogleCallbackLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewOauthGoogleCallbackLogic(ctx context.Context, svcCtx *svc.ServiceContext) *OauthGoogleCallbackLogic {
	return &OauthGoogleCallbackLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *OauthGoogleCallbackLogic) OauthGoogleCallback(req *types.OAuthCallbackReq) (resp *types.OAuthLoginResp, err error) {
	if req.Code == "" {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "缺少 code")
	}
	clientID := l.svcCtx.Config.OAuth.Google.ClientID
	clientSecret := l.svcCtx.Config.OAuth.Google.ClientSecret
	if clientID == "" || clientSecret == "" {
		return nil, errorx.NewCodeError(errorx.CodeSystemError, "未配置 Google OAuth")
	}

	redirectURL := fmt.Sprintf("http://localhost:%d/api/v1/user/oauth/google/callback", l.svcCtx.Config.Port)
	conf := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}
	tok, err := conf.Exchange(l.ctx, req.Code)
	if err != nil {
		l.Logger.Errorf("google exchange: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeSystemError, "Google 授权失败")
	}

	client := conf.Client(l.ctx, tok)
	res, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		l.Logger.Errorf("google userinfo: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeSystemError, "获取 Google 用户信息失败")
	}
	defer res.Body.Close()
	var ui googleUserInfo
	if err := json.NewDecoder(res.Body).Decode(&ui); err != nil {
		return nil, errorx.NewCodeError(errorx.CodeSystemError, "解析用户信息失败")
	}
	authId := ui.Sub
	if authId == "" {
		return nil, errorx.NewCodeError(errorx.CodeSystemError, "无效的 Google 用户")
	}

	userId, err := l.svcCtx.UserRepo.FindByAuth(l.ctx, authTypeGoogle, authId)
	if err != nil {
		l.Logger.Errorf("FindByAuth google: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, err.Error())
	}
	if userId == 0 {
		userId, err = l.createGoogleOAuthUser(ui)
		if err != nil {
			return nil, err
		}
	}

	now := time.Now().Unix()
	expire := l.svcCtx.Config.Auth.AccessExpire
	claims := make(jwt.MapClaims)
	claims["exp"] = now + expire
	claims["iat"] = now
	claims["userId"] = userId
	token := jwt.New(jwt.SigningMethodHS256)
	token.Claims = claims
	accessToken, err := token.SignedString([]byte(l.svcCtx.Config.Auth.AccessSecret))
	if err != nil {
		return nil, err
	}
	return &types.OAuthLoginResp{
		UserId:       userId,
		AccessToken:  accessToken,
		RefreshToken: "oauth_refresh",
		ExpiresIn:    expire,
	}, nil
}

func (l *OauthGoogleCallbackLogic) createGoogleOAuthUser(ui googleUserInfo) (int64, error) {
	var email *string
	if ui.Email != "" {
		email = &ui.Email
	}
	nickname := ui.Name
	if nickname == "" {
		nickname = ui.Email
	}
	nick := &nickname
	var av *string
	if ui.Picture != "" {
		av = &ui.Picture
	}
	userId, err := l.svcCtx.UserRepo.Create(l.ctx, email, nil, nil, nil, nick, av, 1)
	if err != nil {
		return 0, errorx.NewCodeError(errorx.CodeDatabaseError, err.Error())
	}
	if err := l.svcCtx.UserRepo.CreateAuth(l.ctx, userId, authTypeGoogle, ui.Sub, ""); err != nil {
		l.Logger.Errorf("CreateAuth: %v", err)
		return 0, errorx.NewCodeError(errorx.CodeDatabaseError, err.Error())
	}
	return userId, nil
}
