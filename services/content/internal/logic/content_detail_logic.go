package logic

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/content/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type ContentDetailLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewContentDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ContentDetailLogic {
	return &ContentDetailLogic{ctx: ctx, svcCtx: svcCtx}
}

func (l *ContentDetailLogic) Detail(contentID int64, userID int64) (*types.ContentDetailResp, error) {
	if contentID <= 0 {
		return nil, fmt.Errorf("内容 ID 无效")
	}
	if l.svcCtx == nil || l.svcCtx.DB == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}

	type row struct {
		ID           int64
		Title        string
		CoverURL     string
		AudioURL     string
		DurationSec  int
		Format       string
		SizeBytes    *int64
		Artist       string
		VipLevel     int16
		PlayCount    int64
		CreatedAt    time.Time
		UpdatedAt    time.Time
	}
	var r row
	err := l.svcCtx.DB.Table("content").
		Select(`id, title, cover_url, audio_url, duration_sec, format, size_bytes, artist,
			vip_level, COALESCE(play_count, 0) AS play_count, created_at, updated_at`).
		Where("id = ? AND status = 1 AND is_deleted = 0", contentID).
		First(&r).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("内容不存在或已下架")
		}
		logx.Errorf("[Content Detail] query: %v", err)
		return nil, fmt.Errorf("查询内容详情失败")
	}

	cdnBase := strings.TrimRight(l.svcCtx.Config.Storage.CdnBaseUrl, "/")
	canPlay := r.VipLevel == 0 || userID > 0
	canDownload := canPlay

	playURL := ""
	if canPlay {
		playURL = resolveCoverURL(cdnBase, r.AudioURL)
	}

	fileSize := int64(0)
	if r.SizeBytes != nil {
		fileSize = *r.SizeBytes
	}

	return &types.ContentDetailResp{
		ContentID:    r.ID,
		Title:        strings.TrimSpace(r.Title),
		CoverURL:     resolveCoverURL(cdnBase, r.CoverURL),
		Description:  "",
		Duration:     r.DurationSec,
		Format:       strings.TrimSpace(r.Format),
		FileSize:     fileSize,
		PlayURL:      playURL,
		Category:     "",
		Tags:         []string{},
		ViewCount:    r.PlayCount,
		LikeCount:    0,
		CommentCount: 0,
		CreateTime:   r.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdateTime:   r.UpdatedAt.Format("2006-01-02 15:04:05"),
		Permission: types.PermissionInfo{
			NeedVIP:       r.VipLevel > 0,
			RequiredLevel: r.VipLevel,
			CanPlay:       canPlay,
			CanDownload:   canDownload,
		},
	}, nil
}
