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

// ContentFavoriteLogic 用户收藏/取消收藏业务逻辑
type ContentFavoriteLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewContentFavoriteLogic 创建收藏逻辑实例
func NewContentFavoriteLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ContentFavoriteLogic {
	return &ContentFavoriteLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// AddFavorite 添加收藏
// 完整流程：
// 1. 校验内容是否存在且可用
// 2. 校验用户是否已收藏（防止重复收藏）
// 3. 使用事务保证数据一致性：
//    - 新增收藏记录 + 收藏数+1
// 4. 返回最新状态和总数
func (l *ContentFavoriteLogic) AddFavorite(contentID int64, userID int64, favoriteType string) (*types.ContentFavoriteResp, error) {
	logx.Infof("====================================")
	logx.Infof("[Favorite] 开始处理收藏请求...")
	logx.Infof("   Content ID: %d", contentID)
	logx.Infof("   User ID:    %d", userID)
	logx.Infof("   Type:       %s", favoriteType)
	logx.Infof("====================================")

	if contentID <= 0 {
		return nil, fmt.Errorf("内容 ID 无效")
	}
	if userID <= 0 {
		return nil, fmt.Errorf("用户未登录")
	}

	favoriteType = strings.TrimSpace(favoriteType)
	if favoriteType == "" {
		favoriteType = "song"
	}

	var content struct {
		ID            int64
		Title         string
		Status        int16
		IsDeleted     int16
		FavoriteCount int64
	}

	err := l.svcCtx.DB.Table("content").
		Select("id, title, status, is_deleted, COALESCE(favorite_count, 0) as favorite_count").
		Where("id = ? AND status = 1 AND is_deleted = 0", contentID).
		First(&content).Error
	if err != nil {
		logx.Errorf("[Favorite] 查询内容失败: contentID=%d error=%v", contentID, err)
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("歌曲不存在或已下架")
		}
		return nil, fmt.Errorf("查询内容失败: %v", err)
	}

	var existingRecord struct {
		ID        int64
		Status    int16
		CreatedAt time.Time
	}
	existingErr := l.svcCtx.DB.Table("user_favorites").
		Select("id, status, created_at").
		Where("user_id = ? AND content_id = ? AND status = 1", userID, contentID).
		First(&existingRecord).Error

	isFavorited := existingErr == nil && existingRecord.ID > 0 && existingRecord.Status == 1

	if isFavorited {
		logx.Infof("[Favorite] 已收藏: userID=%d, contentID=%d", userID, contentID)

		return &types.ContentFavoriteResp{
			Success:       true,
			Message:       "已收藏",
			Favorited:     true,
			FavoriteCount: content.FavoriteCount,
		}, nil
	}

	var resp *types.ContentFavoriteResp

	err = l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {

		createResult := tx.Table("user_favorites").Create(map[string]interface{}{
			"user_id":        userID,
			"content_id":     contentID,
			"favorite_type":  favoriteType,
			"status":         1,
			"created_at":     time.Now(),
			"updated_at":     time.Now(),
		})
		if createResult.Error != nil {

			if strings.Contains(createResult.Error.Error(), "unique") || strings.Contains(createResult.Error.Error(), "duplicate key") {
				resp = &types.ContentFavoriteResp{
					Success:       true,
					Message:       "已收藏",
					Favorited:     true,
					FavoriteCount: content.FavoriteCount,
				}
				return nil
			}

			logx.Errorf("[Favorite] 收藏失败: error=%v", createResult.Error)
			return fmt.Errorf("收藏失败: %v", createResult.Error)
		}

		updateResult := tx.Table("content").
			Where("id = ?", contentID).
			Update("favorite_count", gorm.Expr("COALESCE(favorite_count, 0) + 1"))
		if updateResult.Error != nil {
			logx.Errorf("[Favorite] 更新收藏数失败: error=%v", updateResult.Error)
			return fmt.Errorf("更新收藏数失败: %v", updateResult.Error)
		}

		content.FavoriteCount++

		resp = &types.ContentFavoriteResp{
			Success:       true,
			Message:       "收藏成功",
			Favorited:     true,
			FavoriteCount: content.FavoriteCount,
		}

		logx.Infof("[Favorite] ✅ 收藏成功: userID=%d, contentID=%d, title=%s, type=%s, favoriteCount=%d",
			userID, contentID, content.Title, favoriteType, content.FavoriteCount)

		return nil
	})

	if err != nil {
		logx.Errorf("[Favorite] ❌ 事务执行失败: error=%v", err)
		return nil, err
	}

	logx.Infof("[Favorite] 处理完成: success=%v, favorited=%v, favoriteCount=%d",
		resp.Success, resp.Favorited, resp.FavoriteCount)

	return resp, nil
}

// RemoveFavorite 取消收藏
// 完整流程：
// 1. 校验内容是否存在
// 2. 查询是否存在有效的收藏记录
// 3. 使用事务保证数据一致性：
//    - 更新收藏记录状态为0 + 收藏数-1
// 4. 返回最新状态和总数
func (l *ContentFavoriteLogic) RemoveFavorite(contentID int64, userID int64) (*types.ContentFavoriteResp, error) {
	logx.Infof("====================================")
	logx.Infof("[Favorite] 开始处理取消收藏请求...")
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
		ID            int64
		Title         string
		FavoriteCount int64
	}

	err := l.svcCtx.DB.Table("content").
		Select("id, title, COALESCE(favorite_count, 0) as favorite_count").
		Where("id = ?", contentID).
		First(&content).Error
	if err != nil {
		logx.Errorf("[Favorite] 查询内容失败: contentID=%d error=%v", contentID, err)
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("歌曲不存在")
		}
		return nil, fmt.Errorf("查询内容失败: %v", err)
	}

	var existingRecord struct {
		ID        int64
		Status    int16
	}
	existingErr := l.svcCtx.DB.Table("user_favorites").
		Select("id, status").
		Where("user_id = ? AND content_id = ? AND status = 1", userID, contentID).
		First(&existingRecord).Error

	isFavorited := existingErr == nil && existingRecord.ID > 0 && existingRecord.Status == 1

	if !isFavorited {
		logx.Infof("[Favorite] 未找到收藏记录: userID=%d, contentID=%d", userID, contentID)

		return &types.ContentFavoriteResp{
			Success:       false,
			Message:       "未找到收藏记录",
			Favorited:     false,
			FavoriteCount: content.FavoriteCount,
		}, nil
	}

	var resp *types.ContentFavoriteResp

	err = l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {

		updateResult := tx.Table("user_favorites").
			Where("user_id = ? AND content_id = ?", userID, contentID).
			Update("status", 0)
		if updateResult.Error != nil {
			logx.Errorf("[Favorite] 更新收藏状态失败: error=%v", updateResult.Error)
			return fmt.Errorf("更新收藏状态失败: %v", updateResult.Error)
		}

		countUpdateResult := tx.Table("content").
			Where("id = ?", contentID).
			Update("favorite_count", gorm.Expr("GREATEST(COALESCE(favorite_count, 0) - 1, 0)"))
		if countUpdateResult.Error != nil {
			logx.Errorf("[Favorite] 更新收藏数失败: error=%v", countUpdateResult.Error)
			return fmt.Errorf("更新收藏数失败: %v", countUpdateResult.Error)
		}

		content.FavoriteCount--
		if content.FavoriteCount < 0 {
			content.FavoriteCount = 0
		}

		resp = &types.ContentFavoriteResp{
			Success:       true,
			Message:       "取消收藏成功",
			Favorited:     false,
			FavoriteCount: content.FavoriteCount,
		}

		logx.Infof("[Favorite] ✅ 取消收藏成功: userID=%d, contentID=%d, title=%s, favoriteCount=%d",
			userID, contentID, content.Title, content.FavoriteCount)

		return nil
	})

	if err != nil {
		logx.Errorf("[Favorite] ❌ 事务执行失败: error=%v", err)
		return nil, err
	}

	logx.Infof("[Favorite] 处理完成: success=%v, favorited=%v, favoriteCount=%d",
		resp.Success, resp.Favorited, resp.FavoriteCount)

	return resp, nil
}
