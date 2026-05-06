package logic

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/content/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// ImportPlaylistLogic 导入歌单逻辑
type ImportPlaylistLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewImportPlaylistLogic 创建导入歌单逻辑实例
func NewImportPlaylistLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ImportPlaylistLogic {
	return &ImportPlaylistLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// ImportPlaylist 导入第三方歌单
func (l *ImportPlaylistLogic) ImportPlaylist(req *types.ImportPlaylistReq, userID int64) (*types.ImportPlaylistResp, error) {
	// 1. 验证参数
	if err := l.validateRequest(req); err != nil {
		return nil, err
	}

	// 2. 检查用户歌单数量限制
	if err := l.checkUserPlaylistLimit(userID); err != nil {
		return nil, err
	}

	// 3. 检查是否已导入过该歌单
	existingPlaylistID, err := l.checkExistingImport(userID, req.Platform, req.PlaylistID)
	if err != nil {
		return nil, err
	}
	if existingPlaylistID > 0 {
		return &types.ImportPlaylistResp{
			Success:      false,
			Message:      "该歌单已导入",
			PlaylistID:   existingPlaylistID,
			PlaylistName: "",
			ImportedCount: 0,
			TotalCount:   0,
		}, nil
	}

	// 4. 获取第三方歌单信息
	playlistInfo, err := l.getThirdPartyPlaylistInfo(req, userID)
	if err != nil {
		return nil, fmt.Errorf("获取第三方歌单信息失败: %v", err)
	}

	// 5. 获取第三方歌单歌曲列表
	tracks, err := l.getThirdPartyPlaylistTracks(req, userID)
	if err != nil {
		return nil, fmt.Errorf("获取第三方歌单歌曲失败: %v", err)
	}

	// 6. 创建本地歌单
	playlistID, err := l.createLocalPlaylist(userID, req, playlistInfo)
	if err != nil {
		return nil, fmt.Errorf("创建本地歌单失败: %v", err)
	}

	// 7. 导入歌曲到本地歌单
	importedCount, err := l.importSongsToPlaylist(userID, playlistID, tracks)
	if err != nil {
		return nil, fmt.Errorf("导入歌曲失败: %v", err)
	}

	// 8. 更新歌单歌曲数量
	if err := l.updatePlaylistSongCount(playlistID, importedCount); err != nil {
		logx.Errorf("更新歌单歌曲数量失败: %v", err)
	}

	// 9. 返回导入结果
	playlistName := req.PlaylistName
	if playlistName == "" {
		playlistName = playlistInfo.Name
	}

	return &types.ImportPlaylistResp{
		Success:       true,
		Message:       "导入歌单成功",
		PlaylistID:    playlistID,
		PlaylistName:  playlistName,
		ImportedCount: importedCount,
		TotalCount:    len(tracks),
	}, nil
}

// validateRequest 验证请求参数
func (l *ImportPlaylistLogic) validateRequest(req *types.ImportPlaylistReq) error {
	if req.Platform == "" {
		return fmt.Errorf("平台不能为空")
	}
	if req.PlaylistID == "" {
		return fmt.Errorf("歌单ID不能为空")
	}
	if req.Platform != "spotify" && req.Platform != "qq-music" {
		return fmt.Errorf("不支持的平台: %s", req.Platform)
	}
	return nil
}

// checkUserPlaylistLimit 检查用户歌单数量限制
func (l *ImportPlaylistLogic) checkUserPlaylistLimit(userID int64) error {
	var playlistCount int64
	err := l.svcCtx.DB.Table("playlists").
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Count(&playlistCount).Error
	if err != nil {
		return fmt.Errorf("查询用户歌单数量失败: %v", err)
	}

	const maxPlaylists = 100
	if playlistCount >= maxPlaylists {
		return fmt.Errorf("已达到最大歌单数量限制（%d个）", maxPlaylists)
	}

	return nil
}

// checkExistingImport 检查是否已导入过该歌单
func (l *ImportPlaylistLogic) checkExistingImport(userID int64, platform, playlistID string) (int64, error) {
	var existingPlaylist struct {
		ID int64
	}

	err := l.svcCtx.DB.Table("playlists").
		Select("id").
		Where("user_id = ? AND source = ? AND source_id = ? AND deleted_at IS NULL", 
			userID, platform, playlistID).
		First(&existingPlaylist).Error

	if err != nil {
		// 如果没有找到记录，返回0表示未导入
		if strings.Contains(err.Error(), "record not found") {
			return 0, nil
		}
		return 0, fmt.Errorf("查询已导入歌单失败: %v", err)
	}

	return existingPlaylist.ID, nil
}

// getThirdPartyPlaylistInfo 获取第三方歌单信息
func (l *ImportPlaylistLogic) getThirdPartyPlaylistInfo(req *types.ImportPlaylistReq, userID int64) (*types.QQMusicPlaylistInfo, error) {
	switch req.Platform {
	case "qq-music":
		return l.getQQMusicPlaylistInfo(req.PlaylistID, userID)
	case "spotify":
		return l.getSpotifyPlaylistInfo(req.PlaylistID, userID)
	default:
		return nil, fmt.Errorf("不支持的平台: %s", req.Platform)
	}
}

// getQQMusicPlaylistInfo 获取QQ音乐歌单信息
func (l *ImportPlaylistLogic) getQQMusicPlaylistInfo(playlistID string, userID int64) (*types.QQMusicPlaylistInfo, error) {
	// 使用现有的QQ音乐歌单逻辑
	qqMusicLogic := NewQQMusicPlaylistListLogic(l.ctx, l.svcCtx)
	
	// 获取歌单列表
	resp, err := qqMusicLogic.GetPlaylists(userID)
	if err != nil {
		return nil, err
	}

	// 在歌单列表中查找指定的歌单
	for _, playlist := range resp.Items {
		if playlist.ID == playlistID {
			return &playlist, nil
		}
	}

	return nil, fmt.Errorf("未找到QQ音乐歌单: %s", playlistID)
}

// getSpotifyPlaylistInfo 获取Spotify歌单信息
func (l *ImportPlaylistLogic) getSpotifyPlaylistInfo(playlistID string, userID int64) (*types.QQMusicPlaylistInfo, error) {
	// 使用现有的Spotify歌单逻辑
	spotifyLogic := NewSpotifyPlaylistListLogic(l.ctx, l.svcCtx)
	
	// 获取歌单列表
	resp, err := spotifyLogic.GetPlaylists(userID)
	if err != nil {
		return nil, err
	}

	// 在歌单列表中查找指定的歌单
	for _, playlist := range resp.Items {
		if playlist.ID == playlistID {
			// 转换为QQMusicPlaylistInfo格式（结构相似）
			return &types.QQMusicPlaylistInfo{
				ID:          playlist.ID,
				Name:        playlist.Name,
				Cover:       playlist.Cover,
				SongCount:   playlist.SongCount,
				Owner:       playlist.Owner,
				Description: playlist.Description,
			}, nil
		}
	}

	return nil, fmt.Errorf("未找到Spotify歌单: %s", playlistID)
}

// getThirdPartyPlaylistTracks 获取第三方歌单歌曲列表
func (l *ImportPlaylistLogic) getThirdPartyPlaylistTracks(req *types.ImportPlaylistReq, userID int64) ([]types.QQMusicTrackInfo, error) {
	switch req.Platform {
	case "qq-music":
		return l.getQQMusicPlaylistTracks(req.PlaylistID, userID)
	case "spotify":
		return l.getSpotifyPlaylistTracks(req.PlaylistID, userID)
	default:
		return nil, fmt.Errorf("不支持的平台: %s", req.Platform)
	}
}

// getQQMusicPlaylistTracks 获取QQ音乐歌单歌曲
func (l *ImportPlaylistLogic) getQQMusicPlaylistTracks(playlistID string, userID int64) ([]types.QQMusicTrackInfo, error) {
	qqMusicLogic := NewQQMusicPlaylistLogic(l.ctx, l.svcCtx)
	
	resp, err := qqMusicLogic.GetPlaylistTracks(userID, playlistID, 0, 0)
	if err != nil {
		return nil, err
	}

	return resp.Items, nil
}

// getSpotifyPlaylistTracks 获取Spotify歌单歌曲
func (l *ImportPlaylistLogic) getSpotifyPlaylistTracks(playlistID string, userID int64) ([]types.QQMusicTrackInfo, error) {
	spotifyLogic := NewSpotifyPlaylistLogic(l.ctx, l.svcCtx)
	
	resp, err := spotifyLogic.GetPlaylistTracks(userID, playlistID, 0, 0)
	if err != nil {
		return nil, err
	}

	// 转换为QQMusicTrackInfo格式
	tracks := make([]types.QQMusicTrackInfo, len(resp.Items))
	for i, track := range resp.Items {
		tracks[i] = types.QQMusicTrackInfo{
			ID:         track.ID,
			Name:       track.Name,
			Artists:    track.Artists,
			Album:      track.Album,
			AlbumImage: track.AlbumImage,
			DurationMs: track.DurationMs,
			ExternalURL: track.ExternalURL,
			PreviewURL: track.PreviewURL,
		}
	}

	return tracks, nil
}

// createLocalPlaylist 创建本地歌单
func (l *ImportPlaylistLogic) createLocalPlaylist(userID int64, req *types.ImportPlaylistReq, playlistInfo *types.QQMusicPlaylistInfo) (int64, error) {
	playlistName := req.PlaylistName
	if playlistName == "" {
		playlistName = playlistInfo.Name
	}

	now := time.Now()
	playlistData := map[string]interface{}{
		"user_id":     userID,
		"name":        playlistName,
		"description": playlistInfo.Description,
		"cover_url":   playlistInfo.Cover,
		"song_count":  0,
		"is_public":   1,
		"status":      1,
		"source":      req.Platform,
		"source_id":   req.PlaylistID,
		"created_at":  now,
		"updated_at":  now,
	}

	result := l.svcCtx.DB.Table("playlists").Create(playlistData)
	if result.Error != nil {
		return 0, result.Error
	}

	// 获取创建的ID
	var playlistID int64
	l.svcCtx.DB.Table("playlists").
		Select("id").
		Where("user_id = ? AND name = ? AND source = ? AND source_id = ?", 
			userID, playlistName, req.Platform, req.PlaylistID).
		First(&playlistID)

	logx.Infof("创建本地歌单成功: userID=%d, playlistID=%d, name=%s, source=%s",
		userID, playlistID, playlistName, req.Platform)

	return playlistID, nil
}

// importSongsToPlaylist 导入歌曲到歌单
func (l *ImportPlaylistLogic) importSongsToPlaylist(userID, playlistID int64, tracks []types.QQMusicTrackInfo) (int, error) {
	if len(tracks) == 0 {
		return 0, nil
	}

	importedCount := 0
	now := time.Now()

	for i, track := range tracks {
		// 1. 检查歌曲是否已存在
		songID, err := l.findOrCreateSong(userID, track)
		if err != nil {
			logx.Errorf("处理歌曲失败: %v", err)
			continue
		}

		// 2. 检查歌曲是否已在歌单中
		var existingSong struct {
			SongID int64
		}
		err = l.svcCtx.DB.Table("playlist_songs").
			Select("song_id").
			Where("playlist_id = ? AND song_id = ?", playlistID, songID).
			First(&existingSong).Error

		if err == nil {
			// 歌曲已存在，跳过
			continue
		}

		// 3. 添加歌曲到歌单
		playlistSongData := map[string]interface{}{
			"playlist_id": playlistID,
			"song_id":     songID,
			"sort_order":  i + 1,
			"added_at":    now,
		}

		result := l.svcCtx.DB.Table("playlist_songs").Create(playlistSongData)
		if result.Error != nil {
			logx.Errorf("添加歌曲到歌单失败: %v", result.Error)
			continue
		}

		importedCount++
	}

	logx.Infof("导入歌曲完成: playlistID=%d, total=%d, imported=%d", 
		playlistID, len(tracks), importedCount)

	return importedCount, nil
}

// findOrCreateSong 查找或创建歌曲
func (l *ImportPlaylistLogic) findOrCreateSong(userID int64, track types.QQMusicTrackInfo) (int64, error) {
	// 1. 尝试根据source_id查找歌曲
	var existingSong struct {
		ID int64
	}
	err := l.svcCtx.DB.Table("songs").
		Select("id").
		Where("source_id = ?", track.ID).
		First(&existingSong).Error

	if err == nil {
		// 歌曲已存在，返回ID
		return existingSong.ID, nil
	}

	// 2. 创建新歌曲
	now := time.Now()
	artist := ""
	if len(track.Artists) > 0 {
		artist = track.Artists[0]
	}

	songData := map[string]interface{}{
		"title":       track.Name,
		"artist":      artist,
		"album":       track.Album,
		"cover_url":   track.AlbumImage,
		"duration":    track.DurationMs / 1000,
		"external_url": track.ExternalURL,
		"source":      "qq-music", // 这里简化处理，实际应该根据平台区分
		"source_id":   track.ID,
		"status":      1,
		"created_at":  now,
		"updated_at":  now,
	}

	result := l.svcCtx.DB.Table("songs").Create(songData)
	if result.Error != nil {
		return 0, result.Error
	}

	// 获取创建的ID
	var songID int64
	l.svcCtx.DB.Table("songs").
		Select("id").
		Where("source_id = ?", track.ID).
		First(&songID)

	return songID, nil
}

// updatePlaylistSongCount 更新歌单歌曲数量
func (l *ImportPlaylistLogic) updatePlaylistSongCount(playlistID int64, songCount int) error {
	result := l.svcCtx.DB.Table("playlists").
		Where("id = ?", playlistID).
		Update("song_count", songCount)

	return result.Error
}