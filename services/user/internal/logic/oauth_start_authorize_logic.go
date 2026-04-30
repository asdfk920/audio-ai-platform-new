package logic

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"strings"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/userdomain/auth/oauthstart"
	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// OauthGoogleStartLogic 构造 Google 授权跳转 URL。
type OauthGoogleStartLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewOauthGoogleStartLogic 创建逻辑实例。
func NewOauthGoogleStartLogic(ctx context.Context, svcCtx *svc.ServiceContext) *OauthGoogleStartLogic {
	return &OauthGoogleStartLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

// AuthorizeURL 返回 Google OAuth2 授权页地址（配置完整时 err 为 nil）。
func (l *OauthGoogleStartLogic) AuthorizeURL() (string, error) {
	cfg := &l.svcCtx.Config
	clientID := strings.TrimSpace(cfg.OAuth.Google.ClientID)
	clientSecret := strings.TrimSpace(cfg.OAuth.Google.ClientSecret)
	if clientID == "" || clientSecret == "" {
		return "", errorx.NewCodeError(errorx.CodeSystemError, "未配置 Google OAuth")
	}
	redirectURL := oauthstart.GoogleCallbackURL(cfg)
	if redirectURL == "" {
		return "", errorx.NewCodeError(errorx.CodeSystemError, "无法解析 Google OAuth 回调地址")
	}
	o := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}
	state, err := randomOAuthState()
	if err != nil {
		return "", errorx.NewCodeError(errorx.CodeSystemError, "生成 OAuth state 失败")
	}
	return o.AuthCodeURL(state, oauth2.AccessTypeOffline), nil
}

// OauthWechatStartLogic 构造微信网页授权跳转 URL。
type OauthWechatStartLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewOauthWechatStartLogic 创建逻辑实例。
func NewOauthWechatStartLogic(ctx context.Context, svcCtx *svc.ServiceContext) *OauthWechatStartLogic {
	return &OauthWechatStartLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

// AuthorizeURL 返回微信 OAuth2 授权页地址（配置 appid 时 err 为 nil）。
func (l *OauthWechatStartLogic) AuthorizeURL() (string, error) {
	cfg := &l.svcCtx.Config
	appID := strings.TrimSpace(cfg.OAuth.WeChat.AppId)
	if appID == "" {
		return "", errorx.NewCodeError(errorx.CodeSystemError, "未配置微信 OAuth")
	}
	redirectURL := oauthstart.WeChatCallbackURL(cfg)
	if redirectURL == "" {
		return "", errorx.NewCodeError(errorx.CodeSystemError, "无法解析微信 OAuth 回调地址")
	}
	state, err := randomOAuthState()
	if err != nil {
		return "", errorx.NewCodeError(errorx.CodeSystemError, "生成 OAuth state 失败")
	}
	v := url.Values{}
	v.Set("appid", appID)
	v.Set("redirect_uri", redirectURL)
	v.Set("response_type", "code")
	v.Set("scope", "snsapi_userinfo")
	v.Set("state", state)
	return "https://open.weixin.qq.com/connect/oauth2/authorize?" + v.Encode() + "#wechat_redirect", nil
}

func randomOAuthState() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
