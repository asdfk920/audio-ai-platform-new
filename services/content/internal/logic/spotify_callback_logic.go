package logic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

// SpotifyTokenResponse Spotify Token响应
type SpotifyTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

// SpotifyUserProfile Spotify用户信息
type SpotifyUserProfile struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Images      []struct {
		URL string `json:"url"`
	} `json:"images"`
}

// SpotifyCallbackLogic Spotify回调处理逻辑
type SpotifyCallbackLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewSpotifyCallbackLogic 创建Spotify回调处理逻辑实例
func NewSpotifyCallbackLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SpotifyCallbackLogic {
	return &SpotifyCallbackLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Callback 处理Spotify回调
func (l *SpotifyCallbackLogic) Callback(userID int64, code string, state string) error {
	if !strings.HasPrefix(state, "spotify_") {
		return fmt.Errorf("无效的state参数")
	}

	var tokenData *SpotifyTokenResponse
	var userProfile *SpotifyUserProfile
	var err error

	if l.svcCtx.Config.Spotify.MockMode {
		logx.Info("使用模拟模式处理Spotify回调")
		codePrefix := code
		if len(codePrefix) > 8 {
			codePrefix = codePrefix[:8]
		}
		codePrefix6 := code
		if len(codePrefix6) > 6 {
			codePrefix6 = codePrefix6[:6]
		}
		tokenData = &SpotifyTokenResponse{
			AccessToken:  "mock_access_token_" + codePrefix,
			TokenType:    "Bearer",
			Scope:        "user-read-private user-read-email",
			ExpiresIn:    3600,
			RefreshToken: "mock_refresh_token_" + codePrefix,
		}
		userProfile = &SpotifyUserProfile{
			ID:          "mock_spotify_user_" + codePrefix6,
			DisplayName: "Mock Spotify User",
			Email:       "mock@spotify.com",
			Images: []struct {
				URL string `json:"url"`
			}{
				{URL: "https://i.scdn.co/image/ab6761610000e5eb998ed4a848e3dfcced0dd9f9"},
			},
		}
	} else {
		tokenData, err = l.exchangeCodeForToken(code)
		if err != nil {
			logx.Errorf("获取Spotify Token失败: %v", err)
			return fmt.Errorf("获取Token失败: %v", err)
		}

		userProfile, err = l.getSpotifyUserProfile(tokenData.AccessToken)
		if err != nil {
			logx.Errorf("获取Spotify用户信息失败: %v", err)
			return fmt.Errorf("获取用户信息失败: %v", err)
		}
	}

	var avatarURL string
	if len(userProfile.Images) > 0 {
		avatarURL = userProfile.Images[0].URL
	}

	expiresAt := time.Now().Add(time.Duration(tokenData.ExpiresIn) * time.Second)

	err = l.saveOrUpdateBinding(userID, userProfile.ID, tokenData, avatarURL, expiresAt)
	if err != nil {
		logx.Errorf("保存绑定信息失败: %v", err)
		return fmt.Errorf("保存绑定信息失败: %v", err)
	}

	logx.Infof("Spotify绑定成功: userID=%d, spotifyUserID=%s", userID, userProfile.ID)
	return nil
}

// saveOrUpdateBinding 保存或更新绑定信息
func (l *SpotifyCallbackLogic) saveOrUpdateBinding(userID int64, spotifyUserID string, tokenData *SpotifyTokenResponse, avatarURL string, expiresAt time.Time) error {
	now := time.Now()

	var binding struct {
		ID int64
	}
	result := l.svcCtx.DB.Table("spotify_bindings").
		Where("spotify_user_id = ?", spotifyUserID).
		First(&binding)

	if result.RowsAffected > 0 {
		l.svcCtx.DB.Table("spotify_bindings").
			Where("id = ?", binding.ID).
			Updates(map[string]interface{}{
				"user_id":          userID,
				"access_token":     tokenData.AccessToken,
				"refresh_token":    tokenData.RefreshToken,
				"token_expires_at": expiresAt,
				"display_name":     "",
				"avatar_url":       avatarURL,
				"is_active":        true,
				"updated_at":       now,
			})

		logx.Infof("更新Spotify绑定: userID=%d, spotifyUserID=%s, bindingID=%d", userID, spotifyUserID, binding.ID)
	} else {
		newBinding := map[string]interface{}{
			"user_id":          userID,
			"spotify_user_id":  spotifyUserID,
			"display_name":     "",
			"avatar_url":       avatarURL,
			"access_token":     tokenData.AccessToken,
			"refresh_token":    tokenData.RefreshToken,
			"token_expires_at": expiresAt,
			"is_active":        true,
			"created_at":       now,
			"updated_at":       now,
		}
		l.svcCtx.DB.Table("spotify_bindings").Create(&newBinding)

		logx.Infof("创建Spotify绑定: userID=%d, spotifyUserID=%s", userID, spotifyUserID)
	}

	return nil
}

// exchangeCodeForToken 使用授权码换取AccessToken
func (l *SpotifyCallbackLogic) exchangeCodeForToken(code string) (*SpotifyTokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", fmt.Sprintf("%s/api/v1/content/spotify/callback", l.svcCtx.Config.Spotify.CallbackBaseURL))
	data.Set("client_id", l.svcCtx.Config.Spotify.ClientID)
	data.Set("client_secret", l.svcCtx.Config.Spotify.ClientSecret)

	req, err := http.NewRequest("POST", "https://accounts.spotify.com/api/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token请求失败: %s", string(body))
	}

	var tokenResp SpotifyTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

// getSpotifyUserProfile 获取Spotify用户信息
func (l *SpotifyCallbackLogic) getSpotifyUserProfile(accessToken string) (*SpotifyUserProfile, error) {
	req, err := http.NewRequest("GET", "https://api.spotify.com/v1/me", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("获取用户信息失败: %s", string(body))
	}

	var profile SpotifyUserProfile
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&profile); err != nil {
		return nil, err
	}

	return &profile, nil
}
