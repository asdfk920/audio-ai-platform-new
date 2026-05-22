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

// ContentLikeLogic 点赞/取消点赞业务逻辑
type ContentLikeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewContentLikeLogic 创建点赞逻辑实例
func NewContentLikeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ContentLikeLogic {
	return &ContentLikeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// ToggleLike 切换点赞状态（点赞/取消点赞）
// 完整流程：
// 1. 校验内容是否存在且可用
// 2. 查询用户是否已点赞
// 3. 使用事务保证数据一致性：
//    - 未点赞：新增记录 + 点赞数+1
//    - 已点赞：删除记录 + 点赞数-1
// 4. 返回最新状态和总数
func (l *ContentLikeLogic) ToggleLike(contentID int64, userID int64) (*types.ContentLikeResp, error) {
	logx.Infof("====================================")
	logx.Infof("[Like] 开始处理点赞请求...")
	logx.Infof("   Content ID: %d", contentID)
	logx.Infof("   User ID:    %d", userID)
	logx.Infof("====================================")

	if contentID <= 0 {
		return nil, fmt.Errorf("内容 ID 无效")
	}
	if userID <= 0 {
		return nil, fmt.Errorf("用户未登录")
	}

	var content struct {
		ID        int64
		Title     string
		Status    int16
		IsDeleted int16
		LikeCount int64
	}

	err := l.svcCtx.DB.Table("content").
		Select("id, title, status, is_deleted, like_count").
		Where("id = ? AND status = 1 AND is_deleted = 0", contentID).
		First(&content).Error
	if err != nil {
		logx.Errorf("[Like] 查询内容失败: contentID=%d error=%v", contentID, err)
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("歌曲不存在或已下架")
		}
		return nil, fmt.Errorf("查询内容失败: %v", err)
	}

	var likeRecord struct {
		ID int64
	}
	likeErr := l.svcCtx.DB.Table("user_likes").
		Select("id").
		Where("user_id = ? AND content_id = ?", userID, contentID).
		First(&likeRecord).Error

	isLiked := likeErr == nil && likeRecord.ID > 0
	logx.Infof("[Like] 当前状态: isLiked=%v", isLiked)

	var resp *types.ContentLikeResp

	err = l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		if isLiked {

			deleteResult := tx.Table("user_likes").
				Where("user_id = ? AND content_id = ?", userID, contentID).
				Delete(&struct{}{})
			if deleteResult.Error != nil {
				logx.Errorf("[Like] 取消点赞失败: error=%v", deleteResult.Error)
				return fmt.Errorf("取消点赞失败: %v", deleteResult.Error)
			}

			updateResult := tx.Table("content").
				Where("id = ?", contentID).
				Update("like_count", gorm.Expr("GREATEST(like_count - 1, 0)"))
			if updateResult.Error != nil {
				logx.Errorf("[Like] 更新点赞数失败: error=%v", updateResult.Error)
				return fmt.Errorf("更新点赞数失败: %v", updateResult.Error)
			}

			content.LikeCount--
			if content.LikeCount < 0 {
				content.LikeCount = 0
			}

			resp = &types.ContentLikeResp{
				Success:   true,
				Message:   "取消点赞成功",
				Liked:     false,
				LikeCount: content.LikeCount,
			}

			logx.Infof("[Like] ✅ 取消点赞成功: userID=%d, contentID=%d, title=%s, likeCount=%d",
				userID, contentID, content.Title, content.LikeCount)

		} else {

			createResult := tx.Table("user_likes").Create(map[string]interface{}{
				"user_id":    userID,
				"content_id": contentID,
				"created_at": time.Now(),
			})
			if createResult.Error != nil {
				logx.Errorf("[Like] 点赞失败: error=%v", createResult.Error)
				return fmt.Errorf("点赞失败: %v", createResult.Error)
			}

			updateResult := tx.Table("content").
				Where("id = ?", contentID).
				Update("like_count", gorm.Expr("COALESCE(like_count, 0) + 1"))
			if updateResult.Error != nil {
				logx.Errorf("[Like] 更新点赞数失败: error=%v", updateResult.Error)
				return fmt.Errorf("更新点赞数失败: %v", updateResult.Error)
			}

			content.LikeCount++

			resp = &types.ContentLikeResp{
				Success:   true,
				Message:   "点赞成功",
				Liked:     true,
				LikeCount: content.LikeCount,
			}

			logx.Infof("[Like] ✅ 点赞成功: userID=%d, contentID=%d, title=%s, likeCount=%d",
				userID, contentID, content.Title, content.LikeCount)
		}

		return nil
	})

	if err != nil {
		logx.Errorf("[Like] ❌ 事务执行失败: error=%v", err)
		return nil, err
	}

	logx.Infof("[Like] 处理完成: success=%v, liked=%v, likeCount=%d",
		resp.Success, resp.Liked, resp.LikeCount)

	return resp, nil
}
