package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

const authTypeWechat = "wechat"

type wechatTokenResp struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Openid       string `json:"openid"`
	Unionid      string `json:"unionid"`
	Scope        string `json:"scope"`
	Errcode      int    `json:"errcode"`
	Errmsg       string `json:"errmsg"`
}

type wechatUserInfo struct {
	Openid     string `json:"openid"`
	Nickname   string `json:"nickname"`
	HeadImgURL string `json:"headimgurl"`
	Unionid    string `json:"unionid"`
	Errcode    int    `json:"errcode"`
	Errmsg     string `json:"errmsg"`
}

type OauthWechatCallbackLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewOauthWechatCallbackLogic(ctx context.Context, svcCtx *svc.ServiceContext) *OauthWechatCallbackLogic {
	return &OauthWechatCallbackLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *OauthWechatCallbackLogic) OauthWechatCallback(req *types.OAuthCallbackReq) (resp *types.OAuthLoginResp, err error) {
	if req.Code == "" {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "缺少 code")
	}
	appId := l.svcCtx.Config.OAuth.WeChat.AppId
	appSecret := l.svcCtx.Config.OAuth.WeChat.AppSecret
	if appId == "" || appSecret == "" {
		return nil, errorx.NewCodeError(errorx.CodeSystemError, "未配置微信 OAuth")
	}

	tokenURL := fmt.Sprintf("https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code",
		url.QueryEscape(appId), url.QueryEscape(appSecret), url.QueryEscape(req.Code))
	res, err := http.Get(tokenURL)
	if err != nil {
		l.Logger.Errorf("wechat token request: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeSystemError, "微信登录失败")
	}
	defer res.Body.Close()
	var tr wechatTokenResp
	if err := json.NewDecoder(res.Body).Decode(&tr); err != nil {
		return nil, errorx.NewCodeError(errorx.CodeSystemError, "微信返回解析失败")
	}
	if tr.Errcode != 0 {
		l.Logger.Errorf("wechat token err: %d %s", tr.Errcode, tr.Errmsg)
		return nil, errorx.NewCodeError(errorx.CodeSystemError, "微信授权失败: "+tr.Errmsg)
	}
	authId := tr.Openid
	if authId == "" {
		authId = tr.Unionid
	}

	userId, err := l.svcCtx.UserRepo.FindByAuth(l.ctx, authTypeWechat, authId)
	if err != nil {
		l.Logger.Errorf("FindByAuth wechat: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, err.Error())
	}
	if userId == 0 {
		userInfoURL := fmt.Sprintf("https://api.weixin.qq.com/sns/userinfo?access_token=%s&openid=%s",
			url.QueryEscape(tr.AccessToken), url.QueryEscape(tr.Openid))
		res2, _ := http.Get(userInfoURL)
		var ui wechatUserInfo
		if res2 != nil {
			_ = json.NewDecoder(res2.Body).Decode(&ui)
			res2.Body.Close()
		}
		nickname, avatar := "", ""
		if ui.Errcode == 0 {
			nickname, avatar = ui.Nickname, ui.HeadImgURL
		}
		userId, err = l.createOAuthUser(nickname, avatar, authTypeWechat, authId, tr.RefreshToken)
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

func (l *OauthWechatCallbackLogic) createOAuthUser(nickname, avatar, authType, authId, refreshToken string) (int64, error) {
	var nick, av *string
	if nickname != "" {
		nick = &nickname
	}
	if avatar != "" {
		av = &avatar
	}
	userId, err := l.svcCtx.UserRepo.Create(l.ctx, nil, nil, nil, nil, nick, av, 1)
	if err != nil {
		return 0, errorx.NewCodeError(errorx.CodeDatabaseError, err.Error())
	}
	if err := l.svcCtx.UserRepo.CreateAuth(l.ctx, userId, authType, authId, refreshToken); err != nil {
		l.Logger.Errorf("CreateAuth: %v", err)
		return 0, errorx.NewCodeError(errorx.CodeDatabaseError, err.Error())
	}
	return userId, nil
}
