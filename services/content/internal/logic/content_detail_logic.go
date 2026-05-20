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

// ContentDetail 获取内容详情（完整版）
// 返回字段：content_id, title, cover_url, description, duration, format, file_size,
//
//	play_url(有权限), category, tags, view_count, like_count, comment_count,
//	create_time, update_time, permission(必带)
func (l *ContentDetailLogic) ContentDetail(contentID int64, userID int64) (*types.ContentDetailResp, error) {
	if contentID <= 0 {
		return nil, fmt.Errorf("内容 ID 无效")
	}

	logx.Infof("====================================")
	logx.Infof("[Content Detail] 获取内容详情...")
	logx.Infof("   Content ID: %d", contentID)
	logx.Infof("   User ID:    %d", userID)
	logx.Infof("====================================")

	// 1. 查询内容基本信息
	content, err := l.queryContent(contentID)
	if err != nil {
		return nil, fmt.Errorf("查询内容失败: %v", err)
	}
	if content == nil {
		return nil, fmt.Errorf("内容不存在")
	}

	logx.Infof("[Content Detail] 查询到内容: %s", content.Title)

	// 2. 获取用户会员等级
	userVipLevel := int16(0)
	if userID > 0 {
		vipLevel, err := l.getUserVipLevel(userID)
		if err != nil {
			logx.Errorf("[Content Detail] 获取用户会员等级失败: user_id=%d, err=%v", userID, err)
		} else {
			userVipLevel = vipLevel
			logx.Infof("[Content Detail] 用户会员等级: %d", vipLevel)
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

	// 4. 判断权限
	canPlay := userVipLevel >= content.VipLevel
	canDownload := canPlay && userVipLevel >= 2 // 需要等级>=2才能下载

	if !canPlay && content.VipLevel > 0 {
		canPlay = true // 非会员可试听（但可能有限制）
	}

	logx.Infof("[Content Detail] 权限判断:")
	logx.Infof("   内容VIP等级: %d", content.VipLevel)
	logx.Infof("   用户VIP等级: %d", userVipLevel)
	logx.Infof("   可播放:     %v", canPlay)
	logx.Infof("   可下载:     %v", canDownload)

	// 5. 构建播放地址（有权限才返回完整URL）
	playURL := ""
	if canPlay && content.AudioURL != "" {
		playURL = content.AudioURL
	} else if !canPlay && content.AudioURL != "" {
		playURL = "" // 无权限时不返回播放地址
	}

	// 6. 查询分类名称
	categoryName := ""
	if content.CategoryID > 0 {
		catName, err := l.queryCategoryName(content.CategoryID)
		if err != nil {
			logx.Errorf("[Content Detail] 查询分类名称失败: %v", err)
		} else {
			categoryName = catName
		}
	}

	// 7. 查询标签
	tags := []string{}
	tagList, err := l.queryContentTags(contentID)
	if err != nil {
		logx.Errorf("[Content Detail] 查询标签失败: %v", err)
	} else {
		tags = tagList
	}

	// 8. 查询统计数据
	viewCount, _ := l.queryViewCount(contentID)
	likeCount, _ := l.queryLikeCount(contentID)
	commentCount, _ := l.queryCommentCount(contentID)

	logx.Infof("[Content Detail] 统计数据:")
	logx.Infof("   播放量: %d", viewCount)
	logx.Infof("   点赞数: %d", likeCount)
	logx.Infof("   评论数: %d", commentCount)

	// 9. 组装响应
	resp := &types.ContentDetailResp{
		ContentID:    content.ID,
		Title:        content.Title,
		CoverURL:     content.CoverURL,
		Description:  content.Description,
		Duration:     content.DurationSec,
		Format:       content.Format,
		FileSize:     content.SizeBytes,
		PlayURL:      playURL,
		Category:     categoryName,
		Tags:         tags,
		ViewCount:    viewCount,
		LikeCount:    likeCount,
		CommentCount: commentCount,
		CreateTime:   content.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdateTime:   content.UpdatedAt.Format("2006-01-02 15:04:05"),
		Permission: types.PermissionInfo{
			NeedVIP:       content.VipLevel > 0,
			RequiredLevel: content.VipLevel,
			CanPlay:       canPlay,
			CanDownload:   canDownload,
		},
	}

	logx.Infof("\n====================================")
	logx.Infof("[Content Detail] ✅ 详情获取成功!")
	logx.Infof("====================================")
	logx.Infof("  标题:       %s", resp.Title)
	logx.Infof("  时长:       %d秒 (%.1f分钟)", resp.Duration, float64(resp.Duration)/60)
	logx.Infof("  文件大小:   %.2fMB", float64(resp.FileSize)/(1024*1024))
	logx.Infof("  分类:       %s", resp.Category)
	logx.Infof("  标签:       %s", strings.Join(resp.Tags, ", "))
	logx.Infof("  权限信息:")
	logx.Infof("    - 需要VIP:   %v", resp.Permission.NeedVIP)
	logx.Infof("    - 所需等级:  %d", resp.Permission.RequiredLevel)
	logx.Infof("    - 可播放:    %v", resp.Permission.CanPlay)
	logx.Infof("    - 可下载:    %v", resp.Permission.CanDownload)
	logx.Infof("  创建时间:   %s", resp.CreateTime)
	logx.Infof("  更新时间:   %s", resp.UpdateTime)
	logx.Infof("====================================")

	return resp, nil
}

// contentInfo 内容信息结构（扩展版）
type contentInfo struct {
	ID              int64
	Title           string
	CoverURL        string
	Description     string
	AudioURL        string
	DurationSec     int
	VipLevel        int16
	Format          string
	SizeBytes       int64
	CategoryID      int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
	AudioValidFrom  *time.Time
	AudioValidUntil *time.Time
}

// queryContent 查询基本信息（兼容新旧数据库结构）
func (l *ContentDetailLogic) queryContent(contentID int64) (*contentInfo, error) {
	if l.svcCtx.DB == nil {
		return nil, fmt.Errorf("数据库未就绪")
	}

	query := `
		SELECT 
			c.id, c.title, c.cover_url, c.audio_url,
			c.duration_sec, c.vip_level, c.format, c.size_bytes,
			c.category_id,
			c.created_at,
			c.audio_valid_from, c.audio_valid_until
		FROM content c
		WHERE c.id = $1 AND c.is_deleted = 0 AND c.status = 1
	`

	var info contentInfo
	err := l.svcCtx.DB.Raw(query, contentID).Scan(&info).Error
	if err != nil {
		return nil, err
	}

	info.Description = ""
	info.UpdatedAt = info.CreatedAt

	description, descErr := l.queryFieldString(contentID, "description")
	if descErr == nil && description != "" {
		info.Description = description
	}

	updatedAt, updateErr := l.queryFieldTime(contentID, "updated_at")
	if updateErr == nil {
		info.UpdatedAt = updatedAt
	}

	return &info, nil
}

// queryCategoryName 查询分类名称
func (l *ContentDetailLogic) queryCategoryName(categoryID int64) (string, error) {
	if l.svcCtx.DB == nil {
		return "", fmt.Errorf("数据库未就绪")
	}

	query := `SELECT name FROM content_category WHERE id = $1 AND status = 1 LIMIT 1`

	var name string
	err := l.svcCtx.DB.Raw(query, categoryID).Scan(&name).Error
	if err != nil {
		return "", err
	}

	return name, nil
}

// queryContentTags 查询内容标签
func (l *ContentDetailLogic) queryContentTags(contentID int64) ([]string, error) {
	if l.svcCtx.DB == nil {
		return []string{}, fmt.Errorf("数据库未就绪")
	}

	query := `
		SELECT t.name 
		FROM tag t
		INNER JOIN content_tag ct ON t.id = ct.tag_id
		WHERE ct.content_id = $1 AND t.status = 1
		ORDER BY t.name ASC
	`

	var tags []string
	err := l.svcCtx.DB.Raw(query, contentID).Pluck("name", &tags).Error
	if err != nil {
		logx.Infof("[Content Detail] 查询标签失败（表可能不存在）: %v", err)
		return []string{}, nil
	}

	return tags, nil
}

// queryViewCount 查询播放量
func (l *ContentDetailLogic) queryViewCount(contentID int64) (int64, error) {
	if l.svcCtx.DB == nil {
		return 0, fmt.Errorf("数据库未就绪")
	}

	query := `
		SELECT COALESCE(play_count, 0) 
		FROM content 
		WHERE id = $1
	`

	var count int64
	err := l.svcCtx.DB.Raw(query, contentID).Scan(&count).Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

// queryLikeCount 查询点赞数
func (l *ContentDetailLogic) queryLikeCount(contentID int64) (int64, error) {
	if l.svcCtx.DB == nil {
		return 0, fmt.Errorf("数据库未就绪")
	}

	query := `
		SELECT COUNT(*) 
		FROM user_likes 
		WHERE content_id = $1
	`

	var count int64
	err := l.svcCtx.DB.Raw(query, contentID).Scan(&count).Error
	if err != nil {
		logx.Infof("[Content Detail] 查询点赞数失败: %v", err)
		return 0, nil
	}

	return count, nil
}

// queryCommentCount 查询评论数
func (l *ContentDetailLogic) queryCommentCount(contentID int64) (int64, error) {
	if l.svcCtx.DB == nil {
		return 0, fmt.Errorf("数据库未就绪")
	}

	query := `
		SELECT COUNT(*) 
		FROM comments 
		WHERE content_id = $1 AND status = 1 AND is_deleted = 0
	`

	var count int64
	err := l.svcCtx.DB.Raw(query, contentID).Scan(&count).Error
	if err != nil {
		logx.Infof("[Content Detail] 查询评论数失败（表可能不存在）: %v", err)
		return 0, nil
	}

	return count, nil
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

// queryFieldString 查询字符串字段（字段可能不存在）
func (l *ContentDetailLogic) queryFieldString(contentID int64, fieldName string) (string, error) {
	if l.svcCtx.DB == nil {
		return "", fmt.Errorf("数据库未就绪")
	}

	query := fmt.Sprintf("SELECT %s FROM content WHERE id = $1", fieldName)
	var value string
	err := l.svcCtx.DB.Raw(query, contentID).Scan(&value).Error
	return value, err
}

// queryFieldTime 查询时间字段（字段可能不存在）
func (l *ContentDetailLogic) queryFieldTime(contentID int64, fieldName string) (time.Time, error) {
	if l.svcCtx.DB == nil {
		return time.Time{}, fmt.Errorf("数据库未就绪")
	}

	query := fmt.Sprintf("SELECT %s FROM content WHERE id = $1", fieldName)
	var value time.Time
	err := l.svcCtx.DB.Raw(query, contentID).Scan(&value).Error
	return value, err
}
