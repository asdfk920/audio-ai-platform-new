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

// SpotifyAuthLogic Spotify授权逻辑
type SpotifyAuthLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewSpotifyAuthLogic 创建Spotify授权逻辑实例
func NewSpotifyAuthLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SpotifyAuthLogic {
	return &SpotifyAuthLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Auth 发起Spotify授权
func (l *SpotifyAuthLogic) Auth(userID int64, callbackURL string) (*types.SpotifyAuthResp, error) {
	clientID := l.svcCtx.Config.Spotify.ClientID
	redirectURI := fmt.Sprintf("%s/api/v1/content/spotify/callback", l.svcCtx.Config.Spotify.CallbackBaseURL)

	if callbackURL != "" && strings.HasPrefix(callbackURL, "http") {
		redirectURI = callbackURL
	}

	scopes := []string{
		"user-read-private",
		"user-read-email",
	}
	scopeStr := strings.Join(scopes, " ")

	state := fmt.Sprintf("spotify_%d_%d", userID, time.Now().Unix())

	authURL := fmt.Sprintf(
		"https://accounts.spotify.com/authorize?client_id=%s&response_type=code&redirect_uri=%s&scope=%s&state=%s",
		url.QueryEscape(clientID),
		url.QueryEscape(redirectURI),
		url.QueryEscape(scopeStr),
		url.QueryEscape(state),
	)

	logx.Infof("发起Spotify授权: userID=%d, state=%s", userID, state)

	return &types.SpotifyAuthResp{
		AuthURL: authURL,
	}, nil
}
