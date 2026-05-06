package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/content/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type QQMusicBindingStatusLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewQQMusicBindingStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QQMusicBindingStatusLogic {
	return &QQMusicBindingStatusLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QQMusicBindingStatusLogic) GetStatus(userID int64) (*types.QQMusicBindingStatusResp, error) {
	var binding struct {
		ID               int64           `json:"id"`
		UserID           int64           `json:"user_id"`
		QQOpenID         string          `json:"qq_openid"`
		QQUnionID        string          `json:"qq_unionid"`
		Nickname         string          `json:"nickname"`
		AvatarURL        string          `json:"avatar_url"`
		AccessToken      string          `json:"access_token"`
		RefreshToken     string          `json:"refresh_token"`
		TokenExpiresAt   time.Time       `json:"token_expires_at"`
		MusicLibraryInfo json.RawMessage `json:"music_library_info"`
		IsActive         bool            `json:"is_active"`
		CreatedAt        time.Time       `json:"created_at"`
	}

	result := l.svcCtx.DB.Table("qq_music_bindings").
		Where("user_id = ? AND is_active = true", userID).
		First(&binding)

	if result.RowsAffected == 0 {
		return &types.QQMusicBindingStatusResp{
			IsBound: false,
			Info:    nil,
		}, nil
	}

	if time.Now().After(binding.TokenExpiresAt) {
		logx.Infof("QQ音乐Token已过期，尝试刷新: userID=%d", userID)
		err := l.refreshToken(userID, binding.RefreshToken)
		if err != nil {
			logx.Errorf("刷新QQ音乐Token失败: %v", err)
			return &types.QQMusicBindingStatusResp{
				IsBound: true,
				Info: &types.QQMusicBindingInfo{
					ID:        binding.ID,
					UserID:    binding.UserID,
					QQOpenID:  binding.QQOpenID,
					QQUnionID: binding.QQUnionID,
					Nickname:  binding.Nickname,
					AvatarURL: binding.AvatarURL,
					IsActive:  binding.IsActive,
					BoundAt:   binding.CreatedAt.Format(time.RFC3339),
				},
			}, nil
		}

		result = l.svcCtx.DB.Table("qq_music_bindings").
			Where("user_id = ? AND is_active = true", userID).
			First(&binding)
	}

	var musicLibInfo *map[string]interface{}
	if len(binding.MusicLibraryInfo) > 0 {
		var libInfo map[string]interface{}
		if err := json.Unmarshal(binding.MusicLibraryInfo, &libInfo); err == nil {
			musicLibInfo = &libInfo
		}
	}

	return &types.QQMusicBindingStatusResp{
		IsBound: true,
		Info: &types.QQMusicBindingInfo{
			ID:               binding.ID,
			UserID:           binding.UserID,
			QQOpenID:         binding.QQOpenID,
			QQUnionID:        binding.QQUnionID,
			Nickname:         binding.Nickname,
			AvatarURL:        binding.AvatarURL,
			MusicLibraryInfo: musicLibInfo,
			IsActive:         binding.IsActive,
			BoundAt:          binding.CreatedAt.Format(time.RFC3339),
		},
	}, nil
}

func (l *QQMusicBindingStatusLogic) refreshToken(userID int64, refreshToken string) error {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", l.svcCtx.Config.QQMusic.AppID)
	data.Set("client_secret", l.svcCtx.Config.QQMusic.AppSecret)

	req, err := http.NewRequest("POST", "https://graph.qq.com/oauth2.0/token", strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("刷新Token失败: %s", string(body))
	}

	var tokenResp QQMusicTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return fmt.Errorf("解析Token响应失败: %v", err)
	}

	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	l.svcCtx.DB.Table("qq_music_bindings").
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"access_token":     tokenResp.AccessToken,
			"refresh_token":    tokenResp.RefreshToken,
			"token_expires_at": expiresAt,
			"updated_at":       time.Now(),
		})

	logx.Infof("QQ音乐Token刷新成功: userID=%d", userID)
	return nil
}
