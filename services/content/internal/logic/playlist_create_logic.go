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

const maxPlaylistsPerUser = 50

type PlaylistCreateLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPlaylistCreateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PlaylistCreateLogic {
	return &PlaylistCreateLogic{ctx: ctx, svcCtx: svcCtx}
}

func (l *PlaylistCreateLogic) Create(req *types.PlaylistCreateReq, userID int64) (*types.PlaylistCreateResp, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("用户未登录")
	}
	if req == nil {
		return nil, fmt.Errorf("请求不能为空")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("歌单名称不能为空")
	}
	if len(name) < 1 || len(name) > 100 {
		return nil, fmt.Errorf("歌单名称长度必须在1-100个字符之间")
	}
	desc := strings.TrimSpace(req.Description)
	if len(desc) > 500 {
		return nil, fmt.Errorf("歌单描述长度不能超过500个字符")
	}

	var count int64
	if err := l.svcCtx.DB.Table("playlists").
		Where("user_id = ? AND status = 1 AND deleted_at IS NULL", userID).
		Count(&count).Error; err != nil {
		return nil, fmt.Errorf("查询歌单数量失败")
	}
	if count >= maxPlaylistsPerUser {
		return nil, fmt.Errorf("已达到最大歌单数量限制")
	}

	var dup int64
	_ = l.svcCtx.DB.Table("playlists").
		Where("user_id = ? AND name = ? AND status = 1 AND deleted_at IS NULL", userID, name).
		Count(&dup).Error
	if dup > 0 {
		return nil, fmt.Errorf("歌单名称已存在")
	}

	isPublic := int16(1)
	if req.IsPublic != nil && !*req.IsPublic {
		isPublic = 0
	}

	now := time.Now()
	coverURL := strings.TrimSpace(req.CoverURL)
	var id int64
	err := l.svcCtx.DB.Raw(`
		INSERT INTO playlists (user_id, name, description, cover_url, song_count, is_public, status, source, created_at, updated_at)
		VALUES (?, ?, ?, ?, 0, ?, 1, 'local', ?, ?)
		RETURNING id
	`, userID, name, desc, coverURL, isPublic, now, now).Scan(&id).Error
	if err != nil {
		logx.Errorf("[Playlist Create] insert: %v", err)
		return nil, fmt.Errorf("创建歌单失败")
	}

	return &types.PlaylistCreateResp{
		ID:          id,
		Name:        name,
		Description: desc,
		CoverURL:    coverURL,
		SongCount:   0,
		IsPublic:    isPublic == 1,
		UserID:      userID,
		CreatedAt:   now.Format("2006-01-02 15:04:05"),
		UpdatedAt:   now.Format("2006-01-02 15:04:05"),
	}, nil
}
