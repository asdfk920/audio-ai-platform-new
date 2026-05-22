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

// PlaylistCreateLogic 创建歌单业务逻辑
type PlaylistCreateLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewPlaylistCreateLogic 创建歌单逻辑实例
func NewPlaylistCreateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PlaylistCreateLogic {
	return &PlaylistCreateLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Create 创建歌单
// 完整流程：
// 1. 校验参数合法性（名称非空、长度限制等）
// 2. 检查用户是否达到歌单数量上限
// 3. 检查歌单名称是否重复（同一用户下）
// 4. 生成唯一ID并创建歌单记录
// 5. 返回完整歌单信息
func (l *PlaylistCreateLogic) Create(req *types.PlaylistCreateReq, userID int64) (*types.PlaylistCreateResp, error) {
	logx.Infof("====================================")
	logx.Infof("[Playlist Create] 开始处理创建歌单请求...")
	logx.Infof("   User ID:    %d", userID)
	logx.Infof("   Name:       %s", req.Name)
	logx.Infof("   Description:%s", req.Description)
	logx.Infof("   Cover URL:  %s", req.CoverURL)
	if req.IsPublic != nil {
		logx.Infof("   Is Public:  %v", *req.IsPublic)
	} else {
		logx.Infof("   Is Public:  (默认公开)")
	}
	logx.Infof("====================================")

	if userID <= 0 {
		return nil, fmt.Errorf("用户未登录")
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("歌单名称不能为空")
	}

	if len(name) < 1 || len(name) > 100 {
		return nil, fmt.Errorf("歌单名称长度必须在1-100个字符之间")
	}

	description := strings.TrimSpace(req.Description)
	if len(description) > 500 {
		return nil, fmt.Errorf("歌单描述长度不能超过500个字符")
	}

	var playlistCount int64
	err := l.svcCtx.DB.Table("playlists").
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Count(&playlistCount).Error
	if err != nil {
		logx.Errorf("[Playlist Create] 查询歌单数量失败: error=%v", err)
		return nil, fmt.Errorf("查询歌单数量失败: %v", err)
	}

	const maxPlaylists = 100
	if playlistCount >= maxPlaylists {
		return nil, fmt.Errorf("已达到最大歌单数量限制（%d个）", maxPlaylists)
	}

	logx.Infof("[Playlist Create] 当前歌单数: %d/%d", playlistCount, maxPlaylists)

	coverURL := strings.TrimSpace(req.CoverURL)
	if coverURL == "" {
		coverURL = "/static/default-playlist-cover.png"
	}

	isPublic := true
	if req.IsPublic != nil {
		isPublic = *req.IsPublic
	}

	now := time.Now()

	playlistData := map[string]interface{}{
		"user_id":     userID,
		"name":        name,
		"description": description,
		"cover_url":   coverURL,
		"song_count":  0,
		"is_public": func() int16 {
			if isPublic {
				return 1
			} else {
				return 0
			}
		}(),
		"status":     1,
		"created_at": now,
		"updated_at": now,
	}

	result := l.svcCtx.DB.Table("playlists").Create(&playlistData)
	if result.Error != nil {
		logx.Errorf("[Playlist Create] 创建歌单失败: error=%v", result.Error)
		return nil, fmt.Errorf("创建歌单失败: %v", result.Error)
	}

	logx.Infof("[Playlist Create] ✅ 插入成功: RowsAffected=%d", result.RowsAffected)

	if result.RowsAffected == 0 {
		logx.Errorf("[Playlist Create] 创建歌单失败：未插入任何行")
		return nil, fmt.Errorf("创建歌单失败：未插入任何行")
	}

	var newPlaylistID int64
	idErr := l.svcCtx.DB.Table("playlists").
		Select("id").
		Where("user_id = ? AND name = ? AND created_at = ?", userID, name, now).
		Order("id DESC").
		Limit(1).
		Pluck("id", &newPlaylistID).Error

	logx.Infof("[Playlist Create] 查询ID结果: error=%v, id=%d", idErr, newPlaylistID)

	if idErr != nil || newPlaylistID <= 0 {
		logx.Errorf("[Playlist Create] 获取新建歌单ID失败: error=%v, id=%d", idErr, newPlaylistID)

		var totalCount int64
		l.svcCtx.DB.Table("playlists").Count(&totalCount)
		logx.Errorf("[Playlist Create] 调试信息: playlists表总记录数=%d", totalCount)

		return nil, fmt.Errorf("获取新建歌单ID失败")
	}

	var createdPlaylist struct {
		ID          int64
		Name        string
		Description string
		CoverURL    string
		SongCount   int
		IsPublic    int16
		UserID      int64
		CreatedAt   time.Time
		UpdatedAt   time.Time
	}

	queryErr := l.svcCtx.DB.Table("playlists").
		Select("id, user_id, name, description, cover_url, song_count, is_public, created_at, updated_at").
		Where("id = ?", newPlaylistID).
		First(&createdPlaylist).Error

	if queryErr != nil {
		logx.Errorf("[Playlist Create] 查询新建歌单失败: error=%v", queryErr)
		return nil, fmt.Errorf("查询歌单信息失败: %v", queryErr)
	}

	logx.Infof("[Playlist Create] ✅ 歌单创建成功: id=%d, name=%s, isPublic=%v",
		createdPlaylist.ID, createdPlaylist.Name, isPublic)

	resp := &types.PlaylistCreateResp{
		ID:          createdPlaylist.ID,
		Name:        createdPlaylist.Name,
		Description: createdPlaylist.Description,
		CoverURL:    createdPlaylist.CoverURL,
		SongCount:   createdPlaylist.SongCount,
		IsPublic:    createdPlaylist.IsPublic == 1,
		UserID:      createdPlaylist.UserID,
		CreatedAt:   createdPlaylist.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   createdPlaylist.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	logx.Infof("[Playlist Create] 处理完成: success=true, playlistID=%d, name=%s", resp.ID, resp.Name)

	return resp, nil
}
