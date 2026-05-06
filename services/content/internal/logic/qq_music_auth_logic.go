package logic

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/content/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// QQMusicAuthLogic QQ音乐授权逻辑
type QQMusicAuthLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewQQMusicAuthLogic 创建QQ音乐授权逻辑实例
func NewQQMusicAuthLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QQMusicAuthLogic {
	return &QQMusicAuthLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Auth 发起QQ音乐授权
func (l *QQMusicAuthLogic) Auth(userID int64, callbackURL string) (*types.QQMusicAuthResp, error) {
	appID := l.svcCtx.Config.QQMusic.AppID
	redirectURI := fmt.Sprintf("%s/api/v1/content/qq-music/callback", l.svcCtx.Config.QQMusic.CallbackBaseURL)

	if callbackURL != "" && strings.HasPrefix(callbackURL, "http") {
		redirectURI = callbackURL
	}

	scopes := []string{
		"get_user_info",
		"get_simple_userinfo",
		"get_music_library_info",
	}
	scopeStr := strings.Join(scopes, ",")

	state := fmt.Sprintf("qqmusic_%d_%d", userID, time.Now().Unix())

	authURL := fmt.Sprintf(
		"https://graph.qq.com/oauth2.0/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=%s&state=%s",
		url.QueryEscape(appID),
		url.QueryEscape(redirectURI),
		url.QueryEscape(scopeStr),
		url.QueryEscape(state),
	)

	logx.Infof("发起QQ音乐授权: userID=%d, state=%s", userID, state)

	return &types.QQMusicAuthResp{
		AuthURL: authURL,
	}, nil
}
