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

// QQMusicTokenResponse QQ音乐Token响应
type QQMusicTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

// QQMusicUserProfile QQ音乐用户信息
type QQMusicUserProfile struct {
	Ret       int    `json:"ret"`
	Msg       string `json:"msg"`
	Nickname  string `json:"nickname"`
	Gender    string `json:"gender"`
	AvatarURL string `json:"figureurl_qq_2"`
	OpenID    string `json:"openid"`
	UnionID   string `json:"unionid"`
}

// QQMusicLibraryInfo 音乐库信息
type QQMusicLibraryInfo struct {
	SongCount     int `json:"song_count"`
	PlaylistCount int `json:"playlist_count"`
	FavoriteCount int `json:"favorite_count"`
}

// QQMusicCallbackLogic QQ音乐回调处理逻辑
type QQMusicCallbackLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewQQMusicCallbackLogic 创建QQ音乐回调处理逻辑实例
func NewQQMusicCallbackLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QQMusicCallbackLogic {
	return &QQMusicCallbackLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func safeSubstring(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length]
}

// Callback 处理QQ音乐回调
func (l *QQMusicCallbackLogic) Callback(userID int64, code string, state string) error {
	if !strings.HasPrefix(state, "qqmusic_") {
		return fmt.Errorf("无效的state参数")
	}

	var tokenData *QQMusicTokenResponse
	var userProfile *QQMusicUserProfile
	var musicLibInfo *QQMusicLibraryInfo
	var err error

	if l.svcCtx.Config.QQMusic.MockMode {
		logx.Info("使用模拟模式处理QQ音乐回调")
		codePrefix := safeSubstring(code, 8)
		codePrefix6 := safeSubstring(code, 6)
		tokenData = &QQMusicTokenResponse{
			AccessToken:  "mock_qq_access_token_" + codePrefix,
			TokenType:    "Bearer",
			ExpiresIn:    7776000,
			RefreshToken: "mock_qq_refresh_token_" + codePrefix,
		}
		userProfile = &QQMusicUserProfile{
			Ret:       0,
			Msg:       "success",
			Nickname:  "QQ音乐Mock用户",
			Gender:    "男",
			AvatarURL: "https://thirdqq.qlogo.cn/g?b=oid&k=mock&s=100",
			OpenID:    "mock_qq_openid_" + codePrefix6,
			UnionID:   "mock_qq_unionid_" + codePrefix6,
		}
		musicLibInfo = &QQMusicLibraryInfo{
			SongCount:     1234,
			PlaylistCount: 56,
			FavoriteCount: 789,
		}
	} else {
		tokenData, err = l.exchangeCodeForToken(code)
		if err != nil {
			logx.Errorf("获取QQ音乐Token失败: %v", err)
			return fmt.Errorf("获取Token失败: %v", err)
		}

		userProfile, err = l.getQQUserInfo(tokenData.AccessToken)
		if err != nil {
			logx.Errorf("获取QQ音乐用户信息失败: %v", err)
			return fmt.Errorf("获取用户信息失败: %v", err)
		}

		musicLibInfo, err = l.getMusicLibraryInfo(tokenData.AccessToken)
		if err != nil {
			logx.Infof("获取音乐库信息失败（非致命）: %v", err)
			musicLibInfo = &QQMusicLibraryInfo{}
		}
	}

	expiresAt := time.Now().Add(time.Duration(tokenData.ExpiresIn) * time.Second)

	err = l.saveOrUpdateBinding(userID, userProfile.OpenID, userProfile.UnionID, userProfile.Nickname, userProfile.AvatarURL, tokenData, musicLibInfo, expiresAt)
	if err != nil {
		logx.Errorf("保存绑定信息失败: %v", err)
		return fmt.Errorf("保存绑定信息失败: %v", err)
	}

	logx.Infof("QQ音乐绑定成功: userID=%d, openID=%s", userID, userProfile.OpenID)
	return nil
}

// saveOrUpdateBinding 保存或更新绑定信息
func (l *QQMusicCallbackLogic) saveOrUpdateBinding(userID int64, openID string, unionID string, nickname string, avatarURL string, tokenData *QQMusicTokenResponse, musicLibInfo *QQMusicLibraryInfo, expiresAt time.Time) error {
	now := time.Now()

	musicLibJSON, _ := json.Marshal(musicLibInfo)

	var binding struct {
		ID int64
	}
	result := l.svcCtx.DB.Table("qq_music_bindings").
		Where("qq_openid = ?", openID).
		First(&binding)

	if result.RowsAffected > 0 {
		l.svcCtx.DB.Table("qq_music_bindings").
			Where("id = ?", binding.ID).
			Updates(map[string]interface{}{
				"user_id":            userID,
				"qq_unionid":         unionID,
				"nickname":           nickname,
				"avatar_url":         avatarURL,
				"access_token":       tokenData.AccessToken,
				"refresh_token":      tokenData.RefreshToken,
				"token_expires_at":   expiresAt,
				"music_library_info": json.RawMessage(musicLibJSON),
				"is_active":          true,
				"updated_at":         now,
			})

		logx.Infof("更新QQ音乐绑定: userID=%d, openID=%s, bindingID=%d", userID, openID, binding.ID)
	} else {
		newBinding := map[string]interface{}{
			"user_id":            userID,
			"qq_openid":          openID,
			"qq_unionid":         unionID,
			"nickname":           nickname,
			"avatar_url":         avatarURL,
			"access_token":       tokenData.AccessToken,
			"refresh_token":      tokenData.RefreshToken,
			"token_expires_at":   expiresAt,
			"music_library_info": json.RawMessage(musicLibJSON),
			"is_active":          true,
			"created_at":         now,
			"updated_at":         now,
		}
		l.svcCtx.DB.Table("qq_music_bindings").Create(&newBinding)

		logx.Infof("创建QQ音乐绑定: userID=%d, openID=%s", userID, openID)
	}

	return nil
}

// exchangeCodeForToken 使用授权码换取AccessToken
func (l *QQMusicCallbackLogic) exchangeCodeForToken(code string) (*QQMusicTokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("client_id", l.svcCtx.Config.QQMusic.AppID)
	data.Set("client_secret", l.svcCtx.Config.QQMusic.AppSecret)
	data.Set("redirect_uri", fmt.Sprintf("%s/api/v1/content/qq-music/callback", l.svcCtx.Config.QQMusic.CallbackBaseURL))

	req, err := http.NewRequest("POST", "https://graph.qq.com/oauth2.0/token", strings.NewReader(data.Encode()))
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

	var tokenResp QQMusicTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

// getQQUserInfo 获取QQ音乐用户信息
func (l *QQMusicCallbackLogic) getQQUserInfo(accessToken string) (*QQMusicUserProfile, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://graph.qq.com/user/get_user_info?access_token=%s&oauth_consumer_key=%s&format=json",
		url.QueryEscape(accessToken),
		url.QueryEscape(l.svcCtx.Config.QQMusic.AppID),
	), nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var profile QQMusicUserProfile
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&profile); err != nil {
		return nil, err
	}

	if profile.Ret != 0 {
		return nil, fmt.Errorf("获取用户信息失败: %s", profile.Msg)
	}

	return &profile, nil
}

// getMusicLibraryInfo 获取音乐库信息
func (l *QQMusicCallbackLogic) getMusicLibraryInfo(accessToken string) (*QQMusicLibraryInfo, error) {
	req, err := http.NewRequest("GET", "https://c.y.qq.com/music/fcgi-bin/fcg_myinfo_toplist.fcg?format=json&g_tk=5381", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Cookie", fmt.Sprintf("qqmusic_key=%s; qqmusic_uin=0", accessToken))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var libInfo QQMusicLibraryInfo
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&libInfo); err != nil {
		return nil, err
	}

	return &libInfo, nil
}
