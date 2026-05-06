package logic

import (
	"context"
	"fmt"
	"time"

	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

// SpotifyUnbindLogic Spotify解绑逻辑
type SpotifyUnbindLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewSpotifyUnbindLogic 创建Spotify解绑逻辑实例
func NewSpotifyUnbindLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SpotifyUnbindLogic {
	return &SpotifyUnbindLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Unbind 解绑Spotify账号
func (l *SpotifyUnbindLogic) Unbind(userID int64) error {
	result := l.svcCtx.DB.Table("spotify_bindings").
		Where("user_id = ? AND is_active = true", userID).
		Update("is_active", false)

	if result.RowsAffected == 0 {
		return fmt.Errorf("未找到绑定的Spotify账号")
	}

	l.svcCtx.DB.Table("spotify_bindings").
		Where("user_id = ?", userID).
		Update("updated_at", time.Now())

	logx.Infof("解绑Spotify账号: userID=%d", userID)

	return nil
}
