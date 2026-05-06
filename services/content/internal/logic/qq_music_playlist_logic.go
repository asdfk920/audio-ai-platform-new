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

type QQMusicPlaylistLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewQQMusicPlaylistLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QQMusicPlaylistLogic {
	return &QQMusicPlaylistLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QQMusicPlaylistLogic) GetPlaylistTracks(userID int64, playlistID string, limit, offset int) (*types.QQMusicPlaylistTrackResp, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// 检查是否启用模拟模式
	if l.svcCtx.Config.QQMusic.MockMode {
		return l.getMockPlaylistTracks(playlistID, limit, offset)
	}

	accessToken, err := l.getValidAccessToken(userID)
	if err != nil {
		return nil, fmt.Errorf("获取AccessToken失败: %v", err)
	}

	url := fmt.Sprintf("https://c.y.qq.com/rsc/fcgi-bin/fcg_get_diss_by_tag.fcg?sin=0&ein=50&start_num=0&end_num=50&sort_id=1&sort=5&order=0&g_tk=5381&format=json&inCharset=utf8&outCharset=utf8&notice=0&platform=h5&needNewCode=1&cid=205360838&category_id=10000000&reqtype=1&loginUin=0&hostUin=0&format=json&inCharset=utf8&outCharset=utf8&notice=0&platform=yqq.json&needNewCode=0&uin=0&hostUin=0&format=json&inCharset=utf8&outCharset=utf8&notice=0&platform=yqq&needNewCode=0&data={\"comm\":{\"ct\":24,\"cv\":0},\"playlist\":{\"method\":\"get_playlist_detail\",\"param\":{\"disstid\":%s,\"dirid\":1,\"tag\":1,\"is_sub\":0},\"module\":\"music.srfDissInfo.aiDissInfo\"}}", playlistID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Referer", "https://y.qq.com/")
	req.Header.Set("Cookie", fmt.Sprintf("qqmusic_key=%s", accessToken))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求QQ音乐API失败: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("QQ音乐API请求失败: %s", string(body))
	}

	var playlistResp QQMusicPlaylistResponse
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&playlistResp); err != nil {
		return nil, fmt.Errorf("解析QQ音乐响应失败: %v", err)
	}

	if playlistResp.Code != 0 {
		return nil, fmt.Errorf("QQ音乐API返回错误: %s", playlistResp.Msg)
	}

	tracks := make([]types.QQMusicTrackInfo, 0, len(playlistResp.Data.SongList))
	for _, song := range playlistResp.Data.SongList {
		artists := make([]string, 0, len(song.Singer))
		for _, singer := range song.Singer {
			artists = append(artists, singer.Name)
		}

		albumImage := ""
		if song.Album.Mid != "" {
			albumImage = fmt.Sprintf("https://y.gtimg.cn/music/photo_new/T002R300x300M000%s.jpg", song.Album.Mid)
		}

		track := types.QQMusicTrackInfo{
			ID:         song.Mid,
			Name:       song.Name,
			Artists:    artists,
			Album:      song.Album.Name,
			AlbumImage: albumImage,
			DurationMs: song.Interval * 1000,
			ExternalURL: fmt.Sprintf("https://y.qq.com/n/ryqq/songDetail/%s", song.Mid),
			PreviewURL: "",
		}
		tracks = append(tracks, track)
	}

	return &types.QQMusicPlaylistTrackResp{
		Total:  len(playlistResp.Data.SongList),
		Limit:  limit,
		Offset: offset,
		Items:  tracks,
	}, nil
}

// getMockPlaylistTracks 返回模拟QQ音乐歌单歌曲数据
func (l *QQMusicPlaylistLogic) getMockPlaylistTracks(playlistID string, limit, offset int) (*types.QQMusicPlaylistTrackResp, error) {
	// 根据歌单ID返回不同的模拟数据
	var tracks []types.QQMusicTrackInfo

	switch playlistID {
	case "1001": // 我喜欢的音乐
		tracks = []types.QQMusicTrackInfo{
			{
				ID:         "001",
				Name:       "晴天",
				Artists:    []string{"周杰伦"},
				Album:      "叶惠美",
				AlbumImage: "https://y.gtimg.cn/music/photo_new/T002R300x300M000003RMaRI1iFoYd.jpg",
				DurationMs: 240000,
				ExternalURL: "https://y.qq.com/n/ryqq/songDetail/001",
			},
			{
				ID:         "002",
				Name:       "稻香",
				Artists:    []string{"周杰伦"},
				Album:      "魔杰座",
				AlbumImage: "https://y.gtimg.cn/music/photo_new/T002R300x300M000004AlfUb0cVkN1.jpg",
				DurationMs: 220000,
				ExternalURL: "https://y.qq.com/n/ryqq/songDetail/002",
			},
			{
				ID:         "003",
				Name:       "告白气球",
				Artists:    []string{"周杰伦"},
				Album:      "床边故事",
				AlbumImage: "https://y.gtimg.cn/music/photo_new/T002R300x300M0000025NhlN2yWrP4.jpg",
				DurationMs: 235000,
				ExternalURL: "https://y.qq.com/n/ryqq/songDetail/003",
			},
		}
	case "1002": // 华语流行
		tracks = []types.QQMusicTrackInfo{
			{
				ID:         "101",
				Name:       "七里香",
				Artists:    []string{"周杰伦"},
				Album:      "七里香",
				AlbumImage: "https://y.gtimg.cn/music/photo_new/T002R300x300M000003DFRzD192KKD.jpg",
				DurationMs: 300000,
				ExternalURL: "https://y.qq.com/n/ryqq/songDetail/101",
			},
			{
				ID:         "102",
				Name:       "青花瓷",
				Artists:    []string{"周杰伦"},
				Album:      "我很忙",
				AlbumImage: "https://y.gtimg.cn/music/photo_new/T002R300x300M000000f01724b3pWc.jpg",
				DurationMs: 280000,
				ExternalURL: "https://y.qq.com/n/ryqq/songDetail/102",
			},
		}
	case "1003": // 欧美热歌
		tracks = []types.QQMusicTrackInfo{
			{
				ID:         "201",
				Name:       "Shape of You",
				Artists:    []string{"Ed Sheeran"},
				Album:      "÷",
				AlbumImage: "https://y.gtimg.cn/music/photo_new/T002R300x300M000003cQCUf2pYq0V.jpg",
				DurationMs: 233000,
				ExternalURL: "https://y.qq.com/n/ryqq/songDetail/201",
			},
			{
				ID:         "202",
				Name:       "Blinding Lights",
				Artists:    []string{"The Weeknd"},
				Album:      "After Hours",
				AlbumImage: "https://y.gtimg.cn/music/photo_new/T002R300x300M000002jLGWe16Tf1H.jpg",
				DurationMs: 200000,
				ExternalURL: "https://y.qq.com/n/ryqq/songDetail/202",
			},
		}
	case "1004": // 纯音乐
		tracks = []types.QQMusicTrackInfo{
			{
				ID:         "301",
				Name:       "River Flows in You",
				Artists:    []string{"Yiruma"},
				Album:      "First Love",
				AlbumImage: "https://y.gtimg.cn/music/photo_new/T002R300x300M000001BLpXF2DyJe2.jpg",
				DurationMs: 218000,
				ExternalURL: "https://y.qq.com/n/ryqq/songDetail/301",
			},
		}
	case "1005": // KTV必点
		tracks = []types.QQMusicTrackInfo{
			{
				ID:         "401",
				Name:       "小幸运",
				Artists:    []string{"田馥甄"},
				Album:      "我的少女时代",
				AlbumImage: "https://y.gtimg.cn/music/photo_new/T002R300x300M000003Ow85E3pnoiB.jpg",
				DurationMs: 270000,
				ExternalURL: "https://y.qq.com/n/ryqq/songDetail/401",
			},
		}
	case "1006": // 车载音乐
		tracks = []types.QQMusicTrackInfo{
			{
				ID:         "501",
				Name:       "平凡之路",
				Artists:    []string{"朴树"},
				Album:      "后会无期",
				AlbumImage: "https://y.gtimg.cn/music/photo_new/T002R300x300M000002eFUFm2XYZ7z.jpg",
				DurationMs: 320000,
				ExternalURL: "https://y.qq.com/n/ryqq/songDetail/501",
			},
		}
	default:
		tracks = []types.QQMusicTrackInfo{
			{
				ID:         "999",
				Name:       "默认歌曲",
				Artists:    []string{"默认歌手"},
				Album:      "默认专辑",
				AlbumImage: "https://y.gtimg.cn/music/photo_new/T002R300x300M00000000000000000.jpg",
				DurationMs: 240000,
				ExternalURL: "https://y.qq.com/n/ryqq/songDetail/default",
			},
		}
	}

	return &types.QQMusicPlaylistTrackResp{
		Total:  len(tracks),
		Limit:  limit,
		Offset: offset,
		Items:  tracks,
	}, nil
}

func (l *QQMusicPlaylistLogic) getValidAccessToken(userID int64) (string, error) {
	var binding struct {
		AccessToken    string    `json:"access_token"`
		RefreshToken   string    `json:"refresh_token"`
		TokenExpiresAt time.Time `json:"token_expires_at"`
	}

	result := l.svcCtx.DB.Table("qq_music_bindings").
		Select("access_token, refresh_token, token_expires_at").
		Where("user_id = ? AND is_active = true", userID).
		First(&binding)

	if result.RowsAffected == 0 {
		return "", fmt.Errorf("未找到QQ音乐绑定记录")
	}

	if time.Now().Before(binding.TokenExpiresAt.Add(-5 * time.Minute)) {
		return binding.AccessToken, nil
	}

	if binding.RefreshToken == "" {
		return "", fmt.Errorf("Token已过期且无RefreshToken")
	}

	newToken, err := l.refreshQQMusicToken(binding.RefreshToken)
	if err != nil {
		return "", fmt.Errorf("刷新Token失败: %v", err)
	}

	expiresAt := time.Now().Add(time.Duration(newToken.ExpiresIn) * time.Second)

	l.svcCtx.DB.Table("qq_music_bindings").
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"access_token":     newToken.AccessToken,
			"refresh_token":    newToken.RefreshToken,
			"token_expires_at": expiresAt,
			"updated_at":       time.Now(),
		})

	logx.Infof("QQ音乐Token刷新成功: userID=%d", userID)
	return newToken.AccessToken, nil
}

func (l *QQMusicPlaylistLogic) refreshQQMusicToken(refreshToken string) (*QQMusicTokenResponse, error) {
	data := "grant_type=refresh_token&refresh_token=" + refreshToken + "&client_id=" + l.svcCtx.Config.QQMusic.AppID + "&client_secret=" + l.svcCtx.Config.QQMusic.AppSecret

	req, err := http.NewRequest("POST", "https://graph.qq.com/oauth2.0/token", bytes.NewBufferString(data))
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

	var tokenResp QQMusicTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

type QQMusicPlaylistResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		SongList []struct {
			Name     string `json:"name"`
			Mid      string `json:"mid"`
			ID       int64  `json:"id"`
			Interval int    `json:"interval"`
			Singer   []struct {
				Name string `json:"name"`
				Mid  string `json:"mid"`
			} `json:"singer"`
			Album struct {
				Name string `json:"name"`
				Mid  string `json:"mid"`
			} `json:"album"`
		} `json:"songlist"`
	} `json:"data"`
}
