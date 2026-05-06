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

type SpotifyPlaylistLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSpotifyPlaylistLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SpotifyPlaylistLogic {
	return &SpotifyPlaylistLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SpotifyPlaylistLogic) GetPlaylistTracks(userID int64, playlistID string, limit, offset int) (*types.SpotifyPlaylistTrackResp, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// 检查是否启用模拟模式
	if l.svcCtx.Config.Spotify.MockMode {
		return l.getMockPlaylistTracks(playlistID, limit, offset)
	}

	accessToken, err := l.getValidAccessToken(userID)
	if err != nil {
		return nil, fmt.Errorf("获取AccessToken失败: %v", err)
	}

	url := fmt.Sprintf("https://api.spotify.com/v1/playlists/%s/tracks?limit=%d&offset=%d", playlistID, limit, offset)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求Spotify API失败: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Spotify API请求失败: %s", string(body))
	}

	var playlistResp SpotifyPlaylistResponse
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&playlistResp); err != nil {
		return nil, fmt.Errorf("解析Spotify响应失败: %v", err)
	}

	tracks := make([]types.SpotifyTrackInfo, 0, len(playlistResp.Items))
	for _, item := range playlistResp.Items {
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

		track := types.SpotifyTrackInfo{
			ID:          item.Track.ID,
			Name:        item.Track.Name,
			Artists:     artists,
			Album:       item.Track.Album.Name,
			AlbumImage:  albumImage,
			DurationMs:  item.Track.DurationMs,
			ExternalURL: item.Track.ExternalURLs.Spotify,
			PreviewURL:  item.Track.PreviewURL,
		}
		tracks = append(tracks, track)
	}

	return &types.SpotifyPlaylistTrackResp{
		Total:  playlistResp.Total,
		Limit:  playlistResp.Limit,
		Offset: playlistResp.Offset,
		Items:  tracks,
	}, nil
}

// getMockPlaylistTracks 返回模拟歌单歌曲数据
func (l *SpotifyPlaylistLogic) getMockPlaylistTracks(playlistID string, limit, offset int) (*types.SpotifyPlaylistTrackResp, error) {
	// 根据歌单ID返回不同的模拟数据
	var tracks []types.SpotifyTrackInfo

	switch playlistID {
	case "1": // 我喜欢的音乐
		tracks = []types.SpotifyTrackInfo{
			{
				ID:          "101",
				Name:        "晴天",
				Artists:     []string{"周杰伦"},
				Album:       "叶惠美",
				AlbumImage:  "https://example.com/song1.jpg",
				DurationMs:  240000,
				ExternalURL: "https://open.spotify.com/track/101",
			},
			{
				ID:          "102",
				Name:        "稻香",
				Artists:     []string{"周杰伦"},
				Album:       "魔杰座",
				AlbumImage:  "https://example.com/song2.jpg",
				DurationMs:  220000,
				ExternalURL: "https://open.spotify.com/track/102",
			},
			{
				ID:          "103",
				Name:        "告白气球",
				Artists:     []string{"周杰伦"},
				Album:       "床边故事",
				AlbumImage:  "https://example.com/song3.jpg",
				DurationMs:  235000,
				ExternalURL: "https://open.spotify.com/track/103",
			},
		}
	case "2": // 健身歌单
		tracks = []types.SpotifyTrackInfo{
			{
				ID:          "201",
				Name:        "Stronger",
				Artists:     []string{"Kanye West"},
				Album:       "Graduation",
				AlbumImage:  "https://example.com/song4.jpg",
				DurationMs:  312000,
				ExternalURL: "https://open.spotify.com/track/201",
			},
			{
				ID:          "202",
				Name:        "Can't Hold Us",
				Artists:     []string{"Macklemore & Ryan Lewis"},
				Album:       "The Heist",
				AlbumImage:  "https://example.com/song5.jpg",
				DurationMs:  258000,
				ExternalURL: "https://open.spotify.com/track/202",
			},
		}
	case "3": // 睡前音乐
		tracks = []types.SpotifyTrackInfo{
			{
				ID:          "301",
				Name:        "月光",
				Artists:     []string{"贝多芬"},
				Album:       "钢琴奏鸣曲",
				AlbumImage:  "https://example.com/song6.jpg",
				DurationMs:  360000,
				ExternalURL: "https://open.spotify.com/track/301",
			},
			{
				ID:          "302",
				Name:        "River Flows in You",
				Artists:     []string{"Yiruma"},
				Album:       "First Love",
				AlbumImage:  "https://example.com/song7.jpg",
				DurationMs:  218000,
				ExternalURL: "https://open.spotify.com/track/302",
			},
		}
	default:
		tracks = []types.SpotifyTrackInfo{
			{
				ID:          "999",
				Name:        "默认歌曲",
				Artists:     []string{"默认歌手"},
				Album:       "默认专辑",
				AlbumImage:  "https://example.com/default.jpg",
				DurationMs:  240000,
				ExternalURL: "https://open.spotify.com/track/default",
			},
		}
	}

	return &types.SpotifyPlaylistTrackResp{
		Total:  len(tracks),
		Limit:  limit,
		Offset: offset,
		Items:  tracks,
	}, nil
}

func (l *SpotifyPlaylistLogic) getValidAccessToken(userID int64) (string, error) {
	var binding struct {
		AccessToken    string    `json:"access_token"`
		RefreshToken   string    `json:"refresh_token"`
		TokenExpiresAt time.Time `json:"token_expires_at"`
	}

	result := l.svcCtx.DB.Table("spotify_bindings").
		Select("access_token, refresh_token, token_expires_at").
		Where("user_id = ? AND is_active = true", userID).
		First(&binding)

	if result.RowsAffected == 0 {
		return "", fmt.Errorf("未找到Spotify绑定记录")
	}

	if time.Now().Add(5 * time.Minute).Before(binding.TokenExpiresAt) {
		return binding.AccessToken, nil
	}

	if binding.RefreshToken == "" {
		return "", fmt.Errorf("Token已过期且无RefreshToken")
	}

	logx.Infof("Spotify Token即将过期，尝试刷新: userID=%d", userID)

	newToken, err := l.refreshSpotifyToken(binding.RefreshToken)
	if err != nil {
		return "", fmt.Errorf("刷新Token失败: %v", err)
	}

	expiresAt := time.Now().Add(time.Duration(newToken.ExpiresIn) * time.Second)

	l.svcCtx.DB.Table("spotify_bindings").
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"access_token":     newToken.AccessToken,
			"refresh_token":    newToken.RefreshToken,
			"token_expires_at": expiresAt,
			"updated_at":       time.Now(),
		})

	logx.Infof("Spotify Token刷新成功: userID=%d", userID)
	return newToken.AccessToken, nil
}

func (l *SpotifyPlaylistLogic) refreshSpotifyToken(refreshToken string) (*SpotifyTokenResponse, error) {
	data := "grant_type=refresh_token&refresh_token=" + refreshToken

	req, err := http.NewRequest("POST", "https://accounts.spotify.com/api/token", bytes.NewBufferString(data))
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
		return nil, fmt.Errorf("刷新Token失败: %s", string(body))
	}

	var tokenResp SpotifyTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

type SpotifyPlaylistResponse struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Items  []struct {
		Track struct {
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
