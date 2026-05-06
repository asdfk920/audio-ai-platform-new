package logic

import (
	"context"

	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/content/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type QQMusicFavoriteLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewQQMusicFavoriteLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QQMusicFavoriteLogic {
	return &QQMusicFavoriteLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// GetFavoriteTracks 获取用户收藏的歌曲列表
func (l *QQMusicFavoriteLogic) GetFavoriteTracks(userID int64, limit, offset int) (*types.QQMusicFavoriteListResp, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	// QQ 音乐暂时使用模拟数据
	// TODO: 实现真实的 QQ 音乐收藏歌曲 API 调用
	return l.getMockFavoriteTracks(limit, offset)
}

// getMockFavoriteTracks 返回模拟的收藏歌曲数据
func (l *QQMusicFavoriteLogic) getMockFavoriteTracks(limit, offset int) (*types.QQMusicFavoriteListResp, error) {
	// 模拟数据
	allTracks := []types.QQMusicFavoriteTrack{
		{
			ID:          "1001",
			Name:        "晴天",
			Artists:     []string{"周杰伦"},
			Album:       "叶惠美",
			AlbumImage:  "https://y.gtimg.cn/music/photo_new/T002R300x300M000003RMaRI1iFoYd.jpg",
			DurationMs:  240000,
			ExternalURL: "https://y.qq.com/n/ryqq/songDetail/1001",
			AddedAt:     "2026-01-15T10:30:00Z",
		},
		{
			ID:          "1002",
			Name:        "稻香",
			Artists:     []string{"周杰伦"},
			Album:       "魔杰座",
			AlbumImage:  "https://y.gtimg.cn/music/photo_new/T002R300x300M000004AlfUb0cVkN1.jpg",
			DurationMs:  220000,
			ExternalURL: "https://y.qq.com/n/ryqq/songDetail/1002",
			AddedAt:     "2026-01-16T14:20:00Z",
		},
		{
			ID:          "1003",
			Name:        "告白气球",
			Artists:     []string{"周杰伦"},
			Album:       "床边故事",
			AlbumImage:  "https://y.gtimg.cn/music/photo_new/T002R300x300M0000025NhlN2yWrP4.jpg",
			DurationMs:  235000,
			ExternalURL: "https://y.qq.com/n/ryqq/songDetail/1003",
			AddedAt:     "2026-01-17T09:15:00Z",
		},
		{
			ID:          "1004",
			Name:        "青花瓷",
			Artists:     []string{"周杰伦"},
			Album:       "我很忙",
			AlbumImage:  "https://y.gtimg.cn/music/photo_new/T002R300x300M000003DFRzD192KKD.jpg",
			DurationMs:  239000,
			ExternalURL: "https://y.qq.com/n/ryqq/songDetail/1004",
			AddedAt:     "2026-01-18T16:45:00Z",
		},
		{
			ID:          "1005",
			Name:        "七里香",
			Artists:     []string{"周杰伦"},
			Album:       "七里香",
			AlbumImage:  "https://y.gtimg.cn/music/photo_new/T002R300x300M000000f01724b3pWc.jpg",
			DurationMs:  299000,
			ExternalURL: "https://y.qq.com/n/ryqq/songDetail/1005",
			AddedAt:     "2026-01-19T11:30:00Z",
		},
		{
			ID:          "2001",
			Name:        "十年",
			Artists:     []string{"陈奕迅"},
			Album:       "黑白灰",
			AlbumImage:  "https://y.gtimg.cn/music/photo_new/T002R300x300M000003cQCUf2pYq0V.jpg",
			DurationMs:  205000,
			ExternalURL: "https://y.qq.com/n/ryqq/songDetail/2001",
			AddedAt:     "2026-01-20T20:00:00Z",
		},
		{
			ID:          "2002",
			Name:        "浮夸",
			Artists:     []string{"陈奕迅"},
			Album:       "U87",
			AlbumImage:  "https://y.gtimg.cn/music/photo_new/T002R300x300M000001pJOG5TzthMx.jpg",
			DurationMs:  287000,
			ExternalURL: "https://y.qq.com/n/ryqq/songDetail/2002",
			AddedAt:     "2026-01-21T08:30:00Z",
		},
		{
			ID:          "2003",
			Name:        "K 歌之王",
			Artists:     []string{"陈奕迅"},
			Album:       "反正是我",
			AlbumImage:  "https://y.gtimg.cn/music/photo_new/T002R300x300M000002MzS21yWrP4.jpg",
			DurationMs:  226000,
			ExternalURL: "https://y.qq.com/n/ryqq/songDetail/2003",
			AddedAt:     "2026-01-22T15:20:00Z",
		},
		{
			ID:          "3001",
			Name:        "传奇",
			Artists:     []string{"王菲"},
			Album:       "传奇",
			AlbumImage:  "https://y.gtimg.cn/music/photo_new/T002R300x300M000004NfRzD192KKD.jpg",
			DurationMs:  245000,
			ExternalURL: "https://y.qq.com/n/ryqq/songDetail/3001",
			AddedAt:     "2026-01-23T12:10:00Z",
		},
		{
			ID:          "3002",
			Name:        "红豆",
			Artists:     []string{"王菲"},
			Album:       "唱游",
			AlbumImage:  "https://y.gtimg.cn/music/photo_new/T002R300x300M000005AlfUb0cVkN1.jpg",
			DurationMs:  253000,
			ExternalURL: "https://y.qq.com/n/ryqq/songDetail/3002",
			AddedAt:     "2026-01-24T18:45:00Z",
		},
		{
			ID:          "4001",
			Name:        "小幸运",
			Artists:     []string{"田馥甄"},
			Album:       "我的少女时代",
			AlbumImage:  "https://y.gtimg.cn/music/photo_new/T002R300x300M000006RMaRI1iFoYd.jpg",
			DurationMs:  278000,
			ExternalURL: "https://y.qq.com/n/ryqq/songDetail/4001",
			AddedAt:     "2026-01-25T10:15:00Z",
		},
		{
			ID:          "4002",
			Name:        "魔鬼中的天使",
			Artists:     []string{"田馥甄"},
			Album:       "My Love",
			AlbumImage:  "https://y.gtimg.cn/music/photo_new/T002R300x300M000007NhlN2yWrP4.jpg",
			DurationMs:  264000,
			ExternalURL: "https://y.qq.com/n/ryqq/songDetail/4002",
			AddedAt:     "2026-01-26T14:30:00Z",
		},
	}

	// 计算分页
	total := len(allTracks)
	start := offset
	if start >= total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}

	var items []types.QQMusicFavoriteTrack
	if start < total {
		items = allTracks[start:end]
	} else {
		items = []types.QQMusicFavoriteTrack{}
	}

	logx.Infof("获取 QQ 音乐模拟收藏歌曲：total=%d, limit=%d, offset=%d, returned=%d",
		total, limit, offset, len(items))

	return &types.QQMusicFavoriteListResp{
		Total:  total,
		Limit:  limit,
		Offset: offset,
		Items:  items,
	}, nil
}
