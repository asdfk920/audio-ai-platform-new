package logic

import (
	"context"

	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/content/internal/types"
)

type SpotifyPlaylistListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSpotifyPlaylistListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SpotifyPlaylistListLogic {
	return &SpotifyPlaylistListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SpotifyPlaylistListLogic) GetPlaylists(userID int64) (*types.SpotifyPlaylistListResp, error) {
	// 模拟数据 - 直接返回假数据
	playlists := []types.SpotifyPlaylistInfo{
		{
			ID:        "1",
			Name:      "我喜欢的音乐",
			Cover:     "https://example.com/cover1.jpg",
			SongCount: 50,
			Owner:     "用户",
			Description: "我最喜欢的歌曲合集",
		},
		{
			ID:        "2",
			Name:      "健身歌单",
			Cover:     "https://example.com/cover2.jpg",
			SongCount: 30,
			Owner:     "用户",
			Description: "健身时听的动感音乐",
		},
		{
			ID:        "3",
			Name:      "睡前音乐",
			Cover:     "https://example.com/cover3.jpg",
			SongCount: 20,
			Owner:     "用户",
			Description: "帮助入睡的舒缓音乐",
		},
		{
			ID:        "4",
			Name:      "工作专注",
			Cover:     "https://example.com/cover4.jpg",
			SongCount: 25,
			Owner:     "用户",
			Description: "提高工作效率的背景音乐",
		},
		{
			ID:        "5",
			Name:      "旅行歌单",
			Cover:     "https://example.com/cover5.jpg",
			SongCount: 40,
			Owner:     "用户",
			Description: "旅行路上听的音乐",
		},
	}

	return &types.SpotifyPlaylistListResp{
		Total: len(playlists),
		Items: playlists,
	}, nil
}