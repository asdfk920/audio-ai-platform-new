package logic

import (
	"context"
	"fmt"
	"time"

	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/content/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type ContentLikeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewContentLikeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ContentLikeLogic {
	return &ContentLikeLogic{ctx: ctx, svcCtx: svcCtx}
}

// ToggleLike 点赞/取消点赞切换
func (l *ContentLikeLogic) ToggleLike(contentID, userID int64) (*types.ContentLikeResp, error) {
	if contentID <= 0 {
		return nil, fmt.Errorf("内容 ID 无效")
	}
	if userID <= 0 {
		return nil, fmt.Errorf("用户未登录")
	}
	if l.svcCtx == nil || l.svcCtx.DB == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}

	var content struct {
		ID     int64
		Title  string
		Status int16
	}
	if err := l.svcCtx.DB.Table("content").
		Select("id, title, status").
		Where("id = ? AND status = 1 AND is_deleted = 0", contentID).
		First(&content).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("歌曲不存在或已下架")
		}
		return nil, fmt.Errorf("查询内容失败")
	}

	var existing struct {
		ID int64
	}
	liked := false
	err := l.svcCtx.DB.Table("user_likes").
		Select("id").
		Where("user_id = ? AND content_id = ?", userID, contentID).
		First(&existing).Error
	if err == nil && existing.ID > 0 {
		liked = true
	}

	var resp *types.ContentLikeResp
	if liked {
		if err := l.svcCtx.DB.Table("user_likes").
			Where("user_id = ? AND content_id = ?", userID, contentID).
			Delete(nil).Error; err != nil {
			logx.Errorf("[Like] delete: %v", err)
			return nil, fmt.Errorf("取消点赞失败")
		}
		resp = &types.ContentLikeResp{
			Success: true,
			Message: "已取消点赞",
			Liked:   false,
		}
	} else {
		now := time.Now()
		if err := l.svcCtx.DB.Table("user_likes").Create(map[string]interface{}{
			"user_id":    userID,
			"content_id": contentID,
			"created_at": now,
		}).Error; err != nil {
			logx.Errorf("[Like] create: %v", err)
			return nil, fmt.Errorf("点赞失败")
		}
		resp = &types.ContentLikeResp{
			Success: true,
			Message: "点赞成功",
			Liked:   true,
		}
	}

	var likeCount int64
	_ = l.svcCtx.DB.Table("user_likes").Where("content_id = ?", contentID).Count(&likeCount).Error
	resp.LikeCount = likeCount

	return resp, nil
}
