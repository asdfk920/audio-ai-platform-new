package logic

import (
	"context"

	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/content/internal/types"
)

// SpotifyBindingStatusLogic Spotify绑定状态查询逻辑
type SpotifyBindingStatusLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewSpotifyBindingStatusLogic 创建Spotify绑定状态查询逻辑实例
func NewSpotifyBindingStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SpotifyBindingStatusLogic {
	return &SpotifyBindingStatusLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// GetStatus 查询用户Spotify绑定状态
func (l *SpotifyBindingStatusLogic) GetStatus(userID int64) (*types.SpotifyBindingStatusResp, error) {
	var binding types.SpotifyBindingInfo
	result := l.svcCtx.DB.Table("spotify_bindings").
		Select("id, user_id, spotify_user_id, display_name, avatar_url, is_active, created_at as bound_at").
		Where("user_id = ? AND is_active = true", userID).
		First(&binding)

	if result.RowsAffected == 0 {
		return &types.SpotifyBindingStatusResp{
			IsBound: false,
			Info:    nil,
		}, nil
	}

	return &types.SpotifyBindingStatusResp{
		IsBound: true,
		Info:    &binding,
	}, nil
}
