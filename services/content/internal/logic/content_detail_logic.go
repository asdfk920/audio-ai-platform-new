package logic

import (
	"context"
	"fmt"
	"time"

	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/content/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// ContentDetailLogic 内容详情逻辑
type ContentDetailLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewContentDetailLogic 创建内容详情逻辑实例
func NewContentDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ContentDetailLogic {
	return &ContentDetailLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// ContentDetail 获取内容详情
// 流程：
//  1. 验证内容 ID
//  2. 查询内容基本信息
//  3. 判断内容是否存在
//  4. 获取用户会员等级
//  5. 检查内容有效期
//  6. 检查用户播放权限
//  7. 查询点赞状态
//  8. 组装响应返回
func (l *ContentDetailLogic) ContentDetail(contentID int64, userID int64) (*types.ContentDetailResp, error) {
	if contentID <= 0 {
		return nil, fmt.Errorf("内容 ID 无效")
	}

	// 1. 查询内容基本信息
	content, err := l.queryContent(contentID)
	if err != nil {
		return nil, fmt.Errorf("查询内容失败: %v", err)
	}
	if content == nil {
		return nil, fmt.Errorf("内容不存在")
	}

	// 2. 获取用户会员等级
	userVipLevel := int16(0)
	if userID > 0 {
		vipLevel, err := l.getUserVipLevel(userID)
		if err != nil {
			logx.Errorf("获取用户会员等级失败: user_id=%d, err=%v", userID, err)
		} else {
			userVipLevel = vipLevel
		}
	}

	// 3. 检查内容有效期
	now := time.Now()
	if content.AudioValidFrom != nil && now.Before(*content.AudioValidFrom) {
		return nil, fmt.Errorf("内容尚未开放")
	}
	if content.AudioValidUntil != nil && now.After(*content.AudioValidUntil) {
		return nil, fmt.Errorf("内容已过期下架")
	}

	// 4. 检查用户播放权限
	canPlay := userVipLevel >= content.VipLevel
	canPlayFull := canPlay
	previewSeconds := 0

	if !canPlay && content.VipLevel > 0 {
		// 非会员用户可试听
		canPlay = true
		canPlayFull = false
		previewSeconds = l.svcCtx.Config.ContentAuth.PreviewSeconds
		if previewSeconds <= 0 {
			previewSeconds = 60
		}
	}

	// 5. 查询点赞状态
	isLiked := false
	if userID > 0 {
		liked, err := l.checkUserLiked(userID, contentID)
		if err != nil {
			logx.Errorf("查询点赞状态失败: user_id=%d, content_id=%d, err=%v", userID, contentID, err)
		} else {
			isLiked = liked
		}
	}

	// 6. 组装响应
	resp := &types.ContentDetailResp{
		ID:             content.ID,
		Title:          content.Title,
		CoverURL:       content.CoverURL,
		AudioURL:       content.AudioURL,
		Artist:         content.Artist,
		Duration:       content.DurationSec,
		VipLevel:       content.VipLevel,
		IsVipContent:   content.VipLevel > 0,
		CanPlay:        canPlay,
		CanPlayFull:    canPlayFull,
		PreviewSeconds: previewSeconds,
		Format:         content.Format,
		SizeBytes:      content.SizeBytes,
		SpatialParams:  content.SpatialParams,
		CategoryID:     content.CategoryID,
		PlayCount:      0,
		LikeCount:      0,
		IsLiked:        isLiked,
		PublishedAt:    content.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if content.AudioValidFrom != nil {
		resp.AudioValidFrom = content.AudioValidFrom.Format("2006-01-02T15:04:05Z")
	}
	if content.AudioValidUntil != nil {
		resp.AudioValidUntil = content.AudioValidUntil.Format("2006-01-02T15:04:05Z")
	}

	return resp, nil
}

// contentInfo 内容信息结构
type contentInfo struct {
	ID              int64
	Title           string
	CoverURL        string
	AudioURL        string
	Artist          string
	DurationSec     int
	VipLevel        int16
	Format          string
	SizeBytes       int64
	SpatialParams   string
	CategoryID      int64
	CreatedAt       time.Time
	AudioValidFrom  *time.Time
	AudioValidUntil *time.Time
}

// queryContent 查询内容基本信息
func (l *ContentDetailLogic) queryContent(contentID int64) (*contentInfo, error) {
	if l.svcCtx.DB == nil {
		return nil, fmt.Errorf("数据库未就绪")
	}

	query := `
		SELECT 
			c.id, c.title, c.cover_url, c.audio_url, c.artist, 
			c.duration_sec, c.vip_level, c.format, c.size_bytes,
			c.spatial_params, c.category_id,
			c.created_at, c.audio_valid_from, c.audio_valid_until
		FROM content c
		WHERE c.id = $1 AND c.is_deleted = 0 AND c.status = 1
	`

	var info contentInfo
	err := l.svcCtx.DB.Raw(query, contentID).Scan(&info).Error
	if err != nil {
		return nil, err
	}

	return &info, nil
}

// checkUserLiked 检查用户是否点赞
func (l *ContentDetailLogic) checkUserLiked(userID, contentID int64) (bool, error) {
	if l.svcCtx.DB == nil {
		return false, fmt.Errorf("数据库未就绪")
	}

	query := `
		SELECT COUNT(*) > 0 
		FROM user_likes 
		WHERE user_id = $1 AND content_id = $2
	`

	var liked bool
	err := l.svcCtx.DB.Raw(query, userID, contentID).Scan(&liked).Error
	if err != nil {
		logx.Errorf("查询点赞状态失败(表可能不存在): %v", err)
		return false, nil
	}

	return liked, nil
}

// getUserVipLevel 获取用户会员等级
func (l *ContentDetailLogic) getUserVipLevel(userID int64) (int16, error) {
	if l.svcCtx.DB == nil {
		return 0, fmt.Errorf("数据库未就绪")
	}

	query := `
		SELECT level 
		FROM user_member 
		WHERE user_id = $1 AND status = 1 
		AND (is_permanent = 1 OR expire_at > NOW())
		LIMIT 1
	`

	var level int16
	err := l.svcCtx.DB.Raw(query, userID).Scan(&level).Error
	if err != nil {
		return 0, err
	}

	return level, nil
}
