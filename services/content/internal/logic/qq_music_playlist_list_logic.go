package logic

import (
	"context"

	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/content/internal/types"
)

type QQMusicPlaylistListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewQQMusicPlaylistListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QQMusicPlaylistListLogic {
	return &QQMusicPlaylistListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QQMusicPlaylistListLogic) GetPlaylists(userID int64) (*types.QQMusicPlaylistListResp, error) {
	// 模拟数据 - 直接返回假数据
	playlists := []types.QQMusicPlaylistInfo{
		{
			ID:        "1001",
			Name:      "我喜欢的音乐",
			Cover:     "https://y.gtimg.cn/music/photo_new/T002R300x300M000003RMaRI1iFoYd.jpg",
			SongCount: 45,
			Owner:     "用户",
			Description: "我最喜欢的歌曲合集",
		},
		{
			ID:        "1002",
			Name:      "华语流行",
			Cover:     "https://y.gtimg.cn/music/photo_new/T002R300x300M000004AlfUb0cVkN1.jpg",
			SongCount: 60,
			Owner:     "用户",
			Description: "经典华语流行歌曲",
		},
		{
			ID:        "1003",
			Name:      "欧美热歌",
			Cover:     "https://y.gtimg.cn/music/photo_new/T002R300x300M0000025NhlN2yWrP4.jpg",
			SongCount: 35,
			Owner:     "用户",
			Description: "最新欧美热门歌曲",
		},
		{
			ID:        "1004",
			Name:      "纯音乐",
			Cover:     "https://y.gtimg.cn/music/photo_new/T002R300x300M000003DFRzD192KKD.jpg",
			SongCount: 25,
			Owner:     "用户",
			Description: "放松心情的纯音乐",
		},
		{
			ID:        "1005",
			Name:      "KTV必点",
			Cover:     "https://y.gtimg.cn/music/photo_new/T002R300x300M000000f01724b3pWc.jpg",
			SongCount: 40,
			Owner:     "用户",
			Description: "KTV热门点唱歌曲",
		},
		{
			ID:        "1006",
			Name:      "车载音乐",
			Cover:     "https://y.gtimg.cn/music/photo_new/T002R300x300M000003cQCUf2pYq0V.jpg",
			SongCount: 50,
			Owner:     "用户",
			Description: "适合开车听的音乐",
		},
	}

	return &types.QQMusicPlaylistListResp{
		Total: len(playlists),
		Items: playlists,
	}, nil
}