package logic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/content/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type SpotifyFavoriteLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSpotifyFavoriteLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SpotifyFavoriteLogic {
	return &SpotifyFavoriteLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// GetFavoriteTracks 获取用户收藏的歌曲列表
func (l *SpotifyFavoriteLogic) GetFavoriteTracks(userID int64, limit, offset int) (*types.SpotifyFavoriteListResp, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	// 检查是否启用模拟模式
	if l.svcCtx.Config.Spotify.MockMode {
		return l.getMockFavoriteTracks(limit, offset)
	}

	accessToken, err := l.getValidAccessToken(userID)
	if err != nil {
		return nil, fmt.Errorf("获取 AccessToken 失败：%v", err)
	}

	// 调用 Spotify API 获取收藏的歌曲
	url := fmt.Sprintf("https://api.spotify.com/v1/me/tracks?limit=%d&offset=%d", limit, offset)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败：%v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 Spotify API 失败：%v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败：%v", err)
	}

	if resp.StatusCode != http.StatusOK {
		logx.Errorf("Spotify API 请求失败：status=%d, body=%s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("Spotify API 请求失败：%s", string(body))
	}

	var favoriteResp SpotifyFavoriteResponse
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&favoriteResp); err != nil {
		return nil, fmt.Errorf("解析 Spotify 响应失败：%v", err)
	}

	// 转换为响应格式
	tracks := make([]types.SpotifyFavoriteTrack, 0, len(favoriteResp.Items))
	for _, item := range favoriteResp.Items {
		if item.Track.ID == "" {
			continue
		}

		artists := make([]string, 0, len(item.Track.Artists))
		for _, artist := range item.Track.Artists {
			artists = append(artists, artist.Name)
		}

		albumImage := ""
		if len(item.Track.Album.Images) > 0 {
			albumImage = item.Track.Album.Images[0].URL
		}

		// 解析添加时间
		addedAt := item.AddedAt
		if addedAt == "" {
			addedAt = time.Now().Format(time.RFC3339)
		}

		track := types.SpotifyFavoriteTrack{
			ID:          item.Track.ID,
			Name:        item.Track.Name,
			Artists:     artists,
			Album:       item.Track.Album.Name,
			AlbumImage:  albumImage,
			DurationMs:  item.Track.DurationMs,
			ExternalURL: item.Track.ExternalURLs.Spotify,
			PreviewURL:  item.Track.PreviewURL,
			AddedAt:     addedAt,
		}
		tracks = append(tracks, track)
	}

	logx.Infof("获取 Spotify 收藏歌曲成功：userID=%d, total=%d, limit=%d, offset=%d, returned=%d",
		userID, favoriteResp.Total, limit, offset, len(tracks))

	return &types.SpotifyFavoriteListResp{
		Total:  favoriteResp.Total,
		Limit:  favoriteResp.Limit,
		Offset: favoriteResp.Offset,
		Items:  tracks,
	}, nil
}

// getValidAccessToken 获取有效的访问令牌
func (l *SpotifyFavoriteLogic) getValidAccessToken(userID int64) (string, error) {
	// 从数据库获取用户的 Spotify 绑定信息
	var binding struct {
		ID           int64  `gorm:"column:id"`
		UserID       int64  `gorm:"column:user_id"`
		AccessToken  string `gorm:"column:access_token"`
		RefreshToken string `gorm:"column:refresh_token"`
		ExpiresAt    string `gorm:"column:expires_at"`
		IsActive     bool   `gorm:"column:is_active"`
	}

	err := l.svcCtx.DB.Table("spotify_bindings").
		Select("id, user_id, access_token, refresh_token, expires_at").
		Where("user_id = ? AND is_active = ?", userID, true).
		First(&binding).Error

	if err != nil {
		return "", fmt.Errorf("未找到 Spotify 绑定信息")
	}

	// 检查 token 是否过期
	expiresAt, err := time.Parse(time.RFC3339, binding.ExpiresAt)
	if err != nil {
		return binding.AccessToken, nil
	}

	// 如果 token 已过期或即将过期（5 分钟内），刷新 token
	if time.Now().Add(5 * time.Minute).After(expiresAt) {
		newToken, err := l.refreshAccessToken(binding.RefreshToken)
		if err != nil {
			logx.Errorf("刷新 Spotify token 失败：%v", err)
			return binding.AccessToken, nil
		}
		return newToken, nil
	}

	return binding.AccessToken, nil
}

// refreshAccessToken 刷新访问令牌
func (l *SpotifyFavoriteLogic) refreshAccessToken(refreshToken string) (string, error) {
	// TODO: 实现刷新 token 的逻辑
	// 这里需要根据 Spotify OAuth2 文档实现刷新逻辑
	return "", fmt.Errorf("刷新 token 功能待实现")
}

// getMockFavoriteTracks 返回模拟的收藏歌曲数据
func (l *SpotifyFavoriteLogic) getMockFavoriteTracks(limit, offset int) (*types.SpotifyFavoriteListResp, error) {
	// 模拟数据
	allTracks := []types.SpotifyFavoriteTrack{
		{
			ID:          "101",
			Name:        "晴天",
			Artists:     []string{"周杰伦"},
			Album:       "叶惠美",
			AlbumImage:  "https://example.com/song1.jpg",
			DurationMs:  240000,
			ExternalURL: "https://open.spotify.com/track/101",
			AddedAt:     "2026-01-15T10:30:00Z",
		},
		{
			ID:          "102",
			Name:        "稻香",
			Artists:     []string{"周杰伦"},
			Album:       "魔杰座",
			AlbumImage:  "https://example.com/song2.jpg",
			DurationMs:  220000,
			ExternalURL: "https://open.spotify.com/track/102",
			AddedAt:     "2026-01-16T14:20:00Z",
		},
		{
			ID:          "103",
			Name:        "告白气球",
			Artists:     []string{"周杰伦"},
			Album:       "床边故事",
			AlbumImage:  "https://example.com/song3.jpg",
			DurationMs:  235000,
			ExternalURL: "https://open.spotify.com/track/103",
			AddedAt:     "2026-01-17T09:15:00Z",
		},
		{
			ID:          "201",
			Name:        "Stronger",
			Artists:     []string{"Kanye West"},
			Album:       "Graduation",
			AlbumImage:  "https://example.com/song4.jpg",
			DurationMs:  312000,
			ExternalURL: "https://open.spotify.com/track/201",
			AddedAt:     "2026-01-18T16:45:00Z",
		},
		{
			ID:          "202",
			Name:        "Can't Hold Us",
			Artists:     []string{"Macklemore & Ryan Lewis"},
			Album:       "The Heist",
			AlbumImage:  "https://example.com/song5.jpg",
			DurationMs:  258000,
			ExternalURL: "https://open.spotify.com/track/202",
			AddedAt:     "2026-01-19T11:30:00Z",
		},
		{
			ID:          "301",
			Name:        "月光",
			Artists:     []string{"贝多芬"},
			Album:       "钢琴奏鸣曲",
			AlbumImage:  "https://example.com/song6.jpg",
			DurationMs:  360000,
			ExternalURL: "https://open.spotify.com/track/301",
			AddedAt:     "2026-01-20T20:00:00Z",
		},
		{
			ID:          "302",
			Name:        "River Flows in You",
			Artists:     []string{"Yiruma"},
			Album:       "First Love",
			AlbumImage:  "https://example.com/song7.jpg",
			DurationMs:  218000,
			ExternalURL: "https://open.spotify.com/track/302",
			AddedAt:     "2026-01-21T08:30:00Z",
		},
		{
			ID:          "401",
			Name:        "Bohemian Rhapsody",
			Artists:     []string{"Queen"},
			Album:       "A Night at the Opera",
			AlbumImage:  "https://example.com/song8.jpg",
			DurationMs:  354000,
			ExternalURL: "https://open.spotify.com/track/401",
			AddedAt:     "2026-01-22T15:20:00Z",
		},
		{
			ID:          "402",
			Name:        "Hotel California",
			Artists:     []string{"Eagles"},
			Album:       "Hotel California",
			AlbumImage:  "https://example.com/song9.jpg",
			DurationMs:  391000,
			ExternalURL: "https://open.spotify.com/track/402",
			AddedAt:     "2026-01-23T12:10:00Z",
		},
		{
			ID:          "403",
			Name:        "Stairway to Heaven",
			Artists:     []string{"Led Zeppelin"},
			Album:       "Led Zeppelin IV",
			AlbumImage:  "https://example.com/song10.jpg",
			DurationMs:  482000,
			ExternalURL: "https://open.spotify.com/track/403",
			AddedAt:     "2026-01-24T18:45:00Z",
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

	var items []types.SpotifyFavoriteTrack
	if start < total {
		items = allTracks[start:end]
	} else {
		items = []types.SpotifyFavoriteTrack{}
	}

	logx.Infof("获取 Spotify 模拟收藏歌曲：total=%d, limit=%d, offset=%d, returned=%d",
		total, limit, offset, len(items))

	return &types.SpotifyFavoriteListResp{
		Total:  total,
		Limit:  limit,
		Offset: offset,
		Items:  items,
	}, nil
}

// SpotifyFavoriteResponse Spotify 收藏歌曲 API 响应结构
type SpotifyFavoriteResponse struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Items  []struct {
		AddedAt string `json:"added_at"`
		Track   struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Artists []struct {
				Name string `json:"name"`
			} `json:"artists"`
			Album struct {
				Name   string `json:"name"`
				Images []struct {
					URL string `json:"url"`
				} `json:"images"`
			} `json:"album"`
			DurationMs   int `json:"duration_ms"`
			ExternalURLs struct {
				Spotify string `json:"spotify"`
			} `json:"external_urls"`
			PreviewURL string `json:"preview_url"`
		} `json:"track"`
	} `json:"items"`
}
