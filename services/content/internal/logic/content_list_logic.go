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

type ContentListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewContentListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ContentListLogic {
	return &ContentListLogic{ctx: ctx, svcCtx: svcCtx}
}

func (l *ContentListLogic) List(req *types.ContentListReq, userID int64) (*types.ContentListResp, error) {
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

	q := l.svcCtx.DB.Table("content AS c").
		Where("c.status = 1 AND c.is_deleted = 0")

	if req != nil {
		if req.CategoryID > 0 {
			q = q.Where("c.category_id = ?", req.CategoryID)
		}
		if kw := strings.TrimSpace(req.Keyword); kw != "" {
			like := "%" + kw + "%"
			q = q.Where("(c.title ILIKE ? OR c.artist ILIKE ?)", like, like)
		} else if t := strings.TrimSpace(req.Title); t != "" {
			q = q.Where("c.title ILIKE ?", "%"+t+"%")
		}
		if req.IsVip == 1 {
			q = q.Where("c.vip_level > 0")
		}
	}

	sort := int32(0)
	if req != nil {
		sort = req.Sort
	}
	switch sort {
	case 1:
		q = q.Order("c.created_at DESC")
	case 2:
		q = q.Order("COALESCE(c.play_count, 0) DESC, c.id DESC")
	case 3:
		q = q.Order("c.sort_order ASC, c.id DESC")
	default:
		q = q.Order("c.sort_order ASC, COALESCE(c.play_count, 0) DESC, c.id DESC")
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		logx.Errorf("[Content List] count: %v", err)
		return nil, fmt.Errorf("查询内容总数失败")
	}

	type row struct {
		ID          int64     `gorm:"column:id"`
		Title       string    `gorm:"column:title"`
		CoverURL    string    `gorm:"column:cover_url"`
		Artist      string    `gorm:"column:artist"`
		DurationSec int       `gorm:"column:duration_sec"`
		Format      string    `gorm:"column:format"`
		SizeBytes   *int64    `gorm:"column:size_bytes"`
		VipLevel    int16     `gorm:"column:vip_level"`
		PlayCount   int64     `gorm:"column:play_count"`
		CreatedAt   time.Time `gorm:"column:created_at"`
	}
	var rows []row
	offset := (int(page) - 1) * int(pageSize)
	err := q.Select(`c.id, c.title, c.cover_url, c.artist, c.duration_sec, c.format, c.size_bytes,
		c.vip_level, COALESCE(c.play_count, 0) AS play_count, c.created_at`).
		Limit(int(pageSize)).Offset(offset).Find(&rows).Error
	if err != nil {
		logx.Errorf("[Content List] query: %v", err)
		return nil, fmt.Errorf("查询内容列表失败")
	}

	cdnBase := strings.TrimRight(l.svcCtx.Config.Storage.CdnBaseUrl, "/")
	list := make([]types.ContentListItem, 0, len(rows))
	for _, r := range rows {
		fileSize := int64(0)
		if r.SizeBytes != nil {
			fileSize = *r.SizeBytes
		}
		canPlay := r.VipLevel == 0 || userID > 0
		list = append(list, types.ContentListItem{
			ContentID:   r.ID,
			Title:       strings.TrimSpace(r.Title),
			CoverURL:    resolveCoverURL(cdnBase, r.CoverURL),
			Artist:      strings.TrimSpace(r.Artist),
			Duration:    r.DurationSec,
			Format:      strings.TrimSpace(r.Format),
			FileSize:    fileSize,
			ViewCount:   r.PlayCount,
			LikeCount:   0,
			PublishedAt: r.CreatedAt.Format("2006-01-02 15:04:05"),
			Permission: types.ListItemPermission{
				NeedVIP:       r.VipLevel > 0,
				RequiredLevel: r.VipLevel,
				CanPlay:       canPlay,
			},
		})
	}

	totalPages := 0
	if pageSize > 0 {
		totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}

	return &types.ContentListResp{
		Total:      total,
		List:       list,
		Page:       int(page),
		PageSize:   int(pageSize),
		HasMore:    int64(page*pageSize) < total,
		TotalPages: totalPages,
	}, nil
}

func resolveCoverURL(cdnBase, raw string) string {
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
