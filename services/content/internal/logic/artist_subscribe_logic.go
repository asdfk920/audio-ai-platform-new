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

// ArtistSubscribeLogic 艺术家订阅业务逻辑
type ArtistSubscribeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewArtistSubscribeLogic 创建艺术家订阅逻辑实例
func NewArtistSubscribeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ArtistSubscribeLogic {
	return &ArtistSubscribeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Subscribe 订阅艺术家
// 完整流程：
// 1. 校验用户是否登录
// 2. 校验艺术家是否存在
// 3. 检查是否已订阅（去重）
// 4. 使用事务保证数据一致性：
//    - 新增/更新订阅记录 + 艺术家粉丝数+1
// 5. 返回订阅成功信息和最新粉丝数
func (l *ArtistSubscribeLogic) Subscribe(artistID int64, userID int64) (*types.ArtistSubscribeResp, error) {
	logx.Infof("====================================")
	logx.Infof("[ArtistSubscribe] 开始处理订阅请求...")
	logx.Infof("   Artist ID: %d", artistID)
	logx.Infof("   User ID:   %d", userID)
	logx.Infof("====================================")

	if artistID <= 0 {
		return nil, fmt.Errorf("艺术家 ID 无效")
	}
	if userID <= 0 {
		return nil, fmt.Errorf("用户未登录")
	}

	var artist struct {
		ID       int64
		Name     string
		Status   int16
		FanCount int64
	}

	err := l.svcCtx.DB.Table("content").
		Select("id, title as name, status, COALESCE(subscribe_count, 0) as fan_count").
		Where("id = ? AND status = 1 AND is_deleted = 0", artistID).
		First(&artist).Error
	if err != nil {
		logx.Errorf("[ArtistSubscribe] 查询艺术家失败: artistID=%d error=%v", artistID, err)
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("艺术家不存在或已下架")
		}
		return nil, fmt.Errorf("查询艺术家失败: %v", err)
	}

	var existingSub struct {
		ID        int64
		Status    int16
		CreatedAt time.Time
	}
	existingErr := l.svcCtx.DB.Table("artist_subscriptions").
		Select("id, status, created_at").
		Where("user_id = ? AND artist_id = ?", userID, artistID).
		First(&existingSub).Error

	isSubscribed := existingErr == nil && existingSub.ID > 0 && existingSub.Status == 1

	if isSubscribed {
		logx.Infof("[ArtistSubscribe] 已订阅，无需重复: userID=%d, artistID=%d", userID, artistID)

		return &types.ArtistSubscribeResp{
			Success:    true,
			Message:    "已订阅该艺术家",
			ArtistID:   artistID,
			ArtistName: artist.Name,
			FanCount:   artist.FanCount,
		}, nil
	}

	var resp *types.ArtistSubscribeResp

	err = l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		if existingSub.ID > 0 && existingSub.Status == 0 {
			updateResult := tx.Table("artist_subscriptions").
				Where("user_id = ? AND artist_id = ?", userID, artistID).
				Updates(map[string]interface{}{
					"status":     1,
					"artist_name": artist.Name,
					"updated_at": time.Now(),
				})
			if updateResult.Error != nil {
				logx.Errorf("[ArtistSubscribe] 更新订阅状态失败: error=%v", updateResult.Error)
				return fmt.Errorf("更新订阅状态失败: %v", updateResult.Error)
			}
		} else {
			createResult := tx.Table("artist_subscriptions").Create(map[string]interface{}{
				"user_id":     userID,
				"artist_id":   artistID,
				"artist_name": artist.Name,
				"status":      1,
				"created_at":  time.Now(),
				"updated_at":  time.Now(),
			})
			if createResult.Error != nil {
				if containsDuplicateKeyError(createResult.Error) {
					resp = &types.ArtistSubscribeResp{
						Success:    true,
						Message:    "已订阅该艺术家",
						ArtistID:   artistID,
						ArtistName: artist.Name,
						FanCount:   artist.FanCount,
					}
					return nil
				}
				logx.Errorf("[ArtistSubscribe] 创建订阅记录失败: error=%v", createResult.Error)
				return fmt.Errorf("订阅失败: %v", createResult.Error)
			}
		}

		countUpdateResult := tx.Table("content").
			Where("id = ?", artistID).
			Update("subscribe_count", gorm.Expr("COALESCE(subscribe_count, 0) + 1"))
		if countUpdateResult.Error != nil {
			logx.Errorf("[ArtistSubscribe] 更新粉丝数失败: error=%v", countUpdateResult.Error)
			return fmt.Errorf("更新粉丝数失败: %v", countUpdateResult.Error)
		}

		artist.FanCount++

		resp = &types.ArtistSubscribeResp{
			Success:    true,
			Message:    "订阅成功",
			ArtistID:   artistID,
			ArtistName: artist.Name,
			FanCount:   artist.FanCount,
		}

		logx.Infof("[ArtistSubscribe] ✅ 订阅成功: userID=%d, artistID=%d, artistName=%s, fanCount=%d",
			userID, artistID, artist.Name, artist.FanCount)

		return nil
	})

	if err != nil {
		logx.Errorf("[ArtistSubscribe] ❌ 事务执行失败: error=%v", err)
		return nil, err
	}

	logx.Infof("[ArtistSubscribe] 处理完成: success=%v, message=%v, fanCount=%d",
		resp.Success, resp.Message, resp.FanCount)

	return resp, nil
}

// Unsubscribe 取消订阅艺术家
// 完整流程：
// 1. 校验用户是否登录
// 2. 校验艺术家是否存在
// 3. 查询是否存在有效的订阅记录
// 4. 使用事务保证数据一致性：
//    - 更新订阅记录状态为0 + 艺术家粉丝数-1
// 5. 返回取消订阅成功信息和最新粉丝数
func (l *ArtistSubscribeLogic) Unsubscribe(artistID int64, userID int64) (*types.ArtistUnsubscribeResp, error) {
	logx.Infof("====================================")
	logx.Infof("[ArtistUnsubscribe] 开始处理取消订阅请求...")
	logx.Infof("   Artist ID: %d", artistID)
	logx.Infof("   User ID:   %d", userID)
	logx.Infof("====================================")

	if artistID <= 0 {
		return nil, fmt.Errorf("艺术家 ID 无效")
	}
	if userID <= 0 {
		return nil, fmt.Errorf("用户未登录")
	}

	var artist struct {
		ID       int64
		Name     string
		FanCount int64
	}

	err := l.svcCtx.DB.Table("content").
		Select("id, title as name, COALESCE(subscribe_count, 0) as fan_count").
		Where("id = ?", artistID).
		First(&artist).Error
	if err != nil {
		logx.Errorf("[ArtistUnsubscribe] 查询艺术家失败: artistID=%d error=%v", artistID, err)
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("艺术家不存在")
		}
		return nil, fmt.Errorf("查询艺术家失败: %v", err)
	}

	var existingSub struct {
		ID        int64
		Status    int16
	}
	existingErr := l.svcCtx.DB.Table("artist_subscriptions").
		Select("id, status").
		Where("user_id = ? AND artist_id = ? AND status = 1", userID, artistID).
		First(&existingSub).Error

	isSubscribed := existingErr == nil && existingSub.ID > 0 && existingSub.Status == 1

	if !isSubscribed {
		logx.Infof("[ArtistUnsubscribe] 未找到订阅记录: userID=%d, artistID=%d", userID, artistID)

		return &types.ArtistUnsubscribeResp{
			Success:    false,
			Message:    "未找到订阅记录",
			ArtistID:   artistID,
			ArtistName: artist.Name,
			FanCount:   artist.FanCount,
		}, nil
	}

	var resp *types.ArtistUnsubscribeResp

	err = l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		updateResult := tx.Table("artist_subscriptions").
			Where("user_id = ? AND artist_id = ?", userID, artistID).
			Update("status", 0)
		if updateResult.Error != nil {
			logx.Errorf("[ArtistUnsubscribe] 更新订阅状态失败: error=%v", updateResult.Error)
			return fmt.Errorf("更新订阅状态失败: %v", updateResult.Error)
		}

		countUpdateResult := tx.Table("content").
			Where("id = ?", artistID).
			Update("subscribe_count", gorm.Expr("GREATEST(COALESCE(subscribe_count, 0) - 1, 0)"))
		if countUpdateResult.Error != nil {
			logx.Errorf("[ArtistUnsubscribe] 更新粉丝数失败: error=%v", countUpdateResult.Error)
			return fmt.Errorf("更新粉丝数失败: %v", countUpdateResult.Error)
		}

		artist.FanCount--
		if artist.FanCount < 0 {
			artist.FanCount = 0
		}

		resp = &types.ArtistUnsubscribeResp{
			Success:    true,
			Message:    "取消订阅成功",
			ArtistID:   artistID,
			ArtistName: artist.Name,
			FanCount:   artist.FanCount,
		}

		logx.Infof("[ArtistUnsubscribe] ✅ 取消订阅成功: userID=%d, artistID=%d, artistName=%s, fanCount=%d",
			userID, artistID, artist.Name, artist.FanCount)

		return nil
	})

	if err != nil {
		logx.Errorf("[ArtistUnsubscribe] ❌ 事务执行失败: error=%v", err)
		return nil, err
	}

	logx.Infof("[ArtistUnsubscribe] 处理完成: success=%v, message=%v, fanCount=%d",
		resp.Success, resp.Message, resp.FanCount)

	return resp, nil
}

// CheckSubscriptionStatus 检查用户对某艺术家的订阅状态
func (l *ArtistSubscribeLogic) CheckSubscriptionStatus(artistID int64, userID int64) (bool, error) {
	if artistID <= 0 || userID <= 0 {
		return false, nil
	}

	var count int64
	err := l.svcCtx.DB.Table("artist_subscriptions").
		Where("user_id = ? AND artist_id = ? AND status = 1", userID, artistID).
		Count(&count).Error
	if err != nil {
		logx.Errorf("[ArtistSubscribe] 检查订阅状态失败: error=%v", err)
		return false, err
	}

	return count > 0, nil
}

// GetSubscriptionList 获取用户的艺术家订阅列表
func (l *ArtistSubscribeLogic) GetSubscriptionList(userID int64, page, pageSize int32) (*types.SubscribeListResp, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("用户未登录")
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var total int64
	countQuery := l.svcCtx.DB.Table("artist_subscriptions AS sub").
		Joins("LEFT JOIN content c ON c.id = sub.artist_id").
		Where("sub.user_id = ? AND sub.status = 1", userID).
		Where("c.is_deleted = 0 OR c.is_deleted IS NULL")

	if err := countQuery.Count(&total).Error; err != nil {
		logx.Errorf("[ArtistSubscribe] 查询订阅总数失败: error=%v", err)
		return nil, fmt.Errorf("查询失败: %v", err)
	}

	offset := (page - 1) * pageSize

	var items []struct {
		ID           int64
		ArtistID     int64
		ArtistName   string
		CoverURL     string
		SubscribedAt time.Time
	}

	query := l.svcCtx.DB.Table("artist_subscriptions AS sub").
		Select("sub.id, sub.artist_id, sub.artist_name, c.cover_url, sub.created_at as subscribed_at").
		Joins("LEFT JOIN content c ON c.id = sub.artist_id").
		Where("sub.user_id = ? AND sub.status = 1", userID).
		Where("c.is_deleted = 0 OR c.is_deleted IS NULL").
		Order("sub.created_at DESC").
		Limit(int(pageSize)).
		Offset(int(offset)).
		Find(&items)

	if query.Error != nil {
		logx.Errorf("[ArtistSubscribe] 查询订阅列表失败: error=%v", query.Error)
		return nil, fmt.Errorf("查询失败: %v", query.Error)
	}

	list := make([]types.SubscribeListItem, 0, len(items))
	for _, item := range items {
		list = append(list, types.SubscribeListItem{
			ID:            item.ID,
			ContentID:     item.ArtistID,
			Title:         item.ArtistName,
			Artist:        item.ArtistName,
			CoverURL:      item.CoverURL,
			SubscribeType: 2,
			SubscribedAt:  item.SubscribedAt.Format("2006-01-02 15:04:05"),
		})
	}

	totalPages := int32(total / int64(pageSize))
	if total%int64(pageSize) > 0 {
		totalPages++
	}

	return &types.SubscribeListResp{
		Total:    total,
		List:     list,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func containsDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return containsIgnoreCase(errStr, "unique") ||
		containsIgnoreCase(errStr, "duplicate key") ||
		containsIgnoreCase(errStr, "SQLSTATE 23505") ||
		containsIgnoreCase(errStr, "duplicate key value violates unique constraint")
}

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		(len(substr) == 0 ||
			(s[0] == substr[0] || absByteDiff(s[0], substr[0]) == 32) &&
				equalFold(s[1:], substr[1:]))
}

func equalFold(s, t string) bool {
	for len(s) > 0 && len(t) > 0 {
		if s[0] == t[0] || absByteDiff(s[0], t[0]) == 32 {
			s = s[1:]
			t = t[1:]
		} else {
			return false
		}
	}
	return len(s) == len(t)
}

func absByteDiff(a, b byte) byte {
	if a < b {
		return b - a
	}
	return a - b
}
