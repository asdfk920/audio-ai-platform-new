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

type PlaylistListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPlaylistListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PlaylistListLogic {
	return &PlaylistListLogic{ctx: ctx, svcCtx: svcCtx}
}

// List 我的歌单列表（当前登录用户）
func (l *PlaylistListLogic) List(userID int64, req *types.PlaylistListReq) (*types.PlaylistListResp, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("用户未登录")
	}
	if l.svcCtx == nil || l.svcCtx.DB == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}

	page := int32(1)
	pageSize := int32(20)
	if req != nil {
		if req.Page > 0 {
			page = req.Page
		}
		if req.PageSize > 0 {
			pageSize = req.PageSize
		}
	}
	if pageSize > 100 {
		pageSize = 100
	}

	q := l.svcCtx.DB.Table("playlists").Where("user_id = ?", userID)
	includeDeleted := req != nil && req.IncludeDeleted
	if !includeDeleted {
		q = q.Where("status = 1 AND deleted_at IS NULL")
	}
	if req != nil {
		if src := strings.TrimSpace(req.Source); src != "" {
			q = q.Where("source = ?", src)
		}
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		logx.Errorf("[Playlist List] count: %v", err)
		return nil, fmt.Errorf("查询歌单总数失败")
	}

	type row struct {
		ID          int64      `gorm:"column:id"`
		Name        string     `gorm:"column:name"`
		Description string     `gorm:"column:description"`
		CoverURL    string     `gorm:"column:cover_url"`
		SongCount   int        `gorm:"column:song_count"`
		IsPublic    int16      `gorm:"column:is_public"`
		Status      int16      `gorm:"column:status"`
		Source      string     `gorm:"column:source"`
		CreatedAt   time.Time  `gorm:"column:created_at"`
		UpdatedAt   time.Time  `gorm:"column:updated_at"`
		DeletedAt   *time.Time `gorm:"column:deleted_at"`
	}
	var rows []row
	offset := (int(page) - 1) * int(pageSize)
	err := q.Select(`id, name, description, cover_url, song_count, is_public, status,
		COALESCE(NULLIF(TRIM(source), ''), 'local') AS source, created_at, updated_at, deleted_at`).
		Order("updated_at DESC, id DESC").
		Limit(int(pageSize)).Offset(offset).Find(&rows).Error
	if err != nil {
		logx.Errorf("[Playlist List] query: %v", err)
		return nil, fmt.Errorf("查询歌单列表失败")
	}

	cdnBase := strings.TrimRight(l.svcCtx.Config.Storage.CdnBaseUrl, "/")
	list := make([]types.MyPlaylistItem, 0, len(rows))
	for _, r := range rows {
		item := types.MyPlaylistItem{
			ID:          r.ID,
			Name:        strings.TrimSpace(r.Name),
			CoverURL:    resolvePlaylistCoverURL(cdnBase, r.CoverURL),
			SongCount:   r.SongCount,
			IsPublic:    r.IsPublic == 1,
			Status:      r.Status,
			CreatedAt:   r.CreatedAt.Format("2006-01-02 15:04:05"),
			Description: strings.TrimSpace(r.Description),
			Source:      strings.TrimSpace(r.Source),
			UpdatedAt:   r.UpdatedAt.Format("2006-01-02 15:04:05"),
		}
		if r.DeletedAt != nil {
			item.DeletedAt = r.DeletedAt.Format("2006-01-02 15:04:05")
		}
		list = append(list, item)
	}

	totalPages := 0
	if pageSize > 0 {
		totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}

	return &types.PlaylistListResp{
		Total:      total,
		List:       list,
		Page:       page,
		PageSize:   pageSize,
		HasMore:    int64(page*pageSize) < total,
		TotalPages: totalPages,
	}, nil
}

func resolvePlaylistCoverURL(cdnBase, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw
	}
	cdnBase = strings.TrimRight(cdnBase, "/")
	if cdnBase == "" {
		return raw
	}
	if strings.HasPrefix(raw, "/") {
		return cdnBase + raw
	}
	return cdnBase + "/" + raw
}
