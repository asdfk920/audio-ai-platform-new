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

type ArtistSubscribeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewArtistSubscribeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ArtistSubscribeLogic {
	return &ArtistSubscribeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Subscribe 订阅艺术家
func (l *ArtistSubscribeLogic) Subscribe(req *types.ArtistSubscribeReq, userID int64) (*types.ArtistSubscribeResp, error) {
	// 1. 验证参数
	if req.ArtistID <= 0 {
		return nil, fmt.Errorf("艺术家 ID 不能为空")
	}

	// 2. 检查艺术家是否存在
	artist, err := l.getArtistByID(req.ArtistID)
	if err != nil {
		return nil, err
	}

	// 3. 检查是否已订阅
	isSubscribed, err := l.checkIfSubscribed(userID, req.ArtistID)
	if err != nil {
		return nil, fmt.Errorf("检查订阅状态失败：%v", err)
	}
	if isSubscribed {
		return &types.ArtistSubscribeResp{
			Success:    false,
			Message:    "已订阅该艺术家",
			ArtistID:   req.ArtistID,
			ArtistName: artist.Name,
			FanCount:   artist.FanCount,
		}, nil
	}

	// 4. 保存订阅记录
	err = l.saveSubscription(userID, req.ArtistID, artist.Name)
	if err != nil {
		return nil, fmt.Errorf("保存订阅记录失败：%v", err)
	}

	// 5. 更新艺术家粉丝数
	newFanCount, err := l.updateArtistFanCount(req.ArtistID, 1)
	if err != nil {
		logx.Errorf("更新艺术家粉丝数失败：artistID=%d, error=%v", req.ArtistID, err)
		// 粉丝数更新失败不影响订阅结果，继续返回成功
		newFanCount = artist.FanCount + 1
	}

	logx.Infof("订阅艺术家成功：userID=%d, artistID=%d, artistName=%s",
		userID, req.ArtistID, artist.Name)

	return &types.ArtistSubscribeResp{
		Success:    true,
		Message:    "订阅成功",
		ArtistID:   req.ArtistID,
		ArtistName: artist.Name,
		FanCount:   newFanCount,
	}, nil
}

// Unsubscribe 取消订阅艺术家
func (l *ArtistSubscribeLogic) Unsubscribe(req *types.ArtistUnsubscribeReq, userID int64) (*types.ArtistUnsubscribeResp, error) {
	// 1. 验证参数
	if req.ArtistID <= 0 {
		return nil, fmt.Errorf("艺术家 ID 不能为空")
	}

	// 2. 检查艺术家是否存在
	artist, err := l.getArtistByID(req.ArtistID)
	if err != nil {
		return nil, err
	}

	// 3. 检查是否已订阅
	isSubscribed, err := l.checkIfSubscribed(userID, req.ArtistID)
	if err != nil {
		return nil, fmt.Errorf("检查订阅状态失败：%v", err)
	}
	if !isSubscribed {
		return &types.ArtistUnsubscribeResp{
			Success:    false,
			Message:    "未订阅该艺术家",
			ArtistID:   req.ArtistID,
			ArtistName: artist.Name,
			FanCount:   artist.FanCount,
		}, nil
	}

	// 4. 删除订阅记录
	err = l.deleteSubscription(userID, req.ArtistID)
	if err != nil {
		return nil, fmt.Errorf("删除订阅记录失败：%v", err)
	}

	// 5. 更新艺术家粉丝数
	newFanCount, err := l.updateArtistFanCount(req.ArtistID, -1)
	if err != nil {
		logx.Errorf("更新艺术家粉丝数失败：artistID=%d, error=%v", req.ArtistID, err)
		// 粉丝数更新失败不影响取消订阅结果，继续返回成功
		newFanCount = artist.FanCount - 1
	}

	logx.Infof("取消订阅艺术家成功：userID=%d, artistID=%d, artistName=%s",
		userID, req.ArtistID, artist.Name)

	return &types.ArtistUnsubscribeResp{
		Success:    true,
		Message:    "取消订阅成功",
		ArtistID:   req.ArtistID,
		ArtistName: artist.Name,
		FanCount:   newFanCount,
	}, nil
}

// getArtistByID 根据 ID 获取艺术家信息
func (l *ArtistSubscribeLogic) getArtistByID(artistID int64) (*ArtistInfo, error) {
	type ArtistInfoLocal struct {
		ID        int64
		Name      string
		AvatarURL string
		Bio       string
		FanCount  int64
		SongCount int64
		Status    int16
	}

	var artist ArtistInfoLocal
	err := l.svcCtx.DB.Table("artists").
		Select("id, name, avatar_url, bio, fan_count, song_count, status").
		Where("id = ?", artistID).
		First(&artist).Error

	if err != nil {
		return nil, fmt.Errorf("艺术家不存在")
	}

	if artist.Status != 1 {
		return nil, fmt.Errorf("艺术家已下架")
	}

	return &ArtistInfo{
		ID:        artist.ID,
		Name:      artist.Name,
		AvatarURL: artist.AvatarURL,
		Bio:       artist.Bio,
		FanCount:  artist.FanCount,
		SongCount: artist.SongCount,
		Status:    artist.Status,
	}, nil
}

// checkIfSubscribed 检查用户是否已订阅艺术家
func (l *ArtistSubscribeLogic) checkIfSubscribed(userID int64, artistID int64) (bool, error) {
	var count int64
	err := l.svcCtx.DB.Table("user_subscriptions").
		Where("user_id = ? AND subscribe_type = 1 AND target_id = ?", userID, artistID).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// saveSubscription 保存订阅记录
func (l *ArtistSubscribeLogic) saveSubscription(userID int64, artistID int64, artistName string) error {
	subscription := map[string]interface{}{
		"user_id":        userID,
		"subscribe_type": 1, // 1-艺术家
		"target_id":      artistID,
		"target_name":    artistName,
		"created_at":     time.Now(),
	}

	result := l.svcCtx.DB.Table("user_subscriptions").Create(subscription)
	return result.Error
}

// deleteSubscription 删除订阅记录
func (l *ArtistSubscribeLogic) deleteSubscription(userID int64, artistID int64) error {
	result := l.svcCtx.DB.Table("user_subscriptions").
		Where("user_id = ? AND subscribe_type = 1 AND target_id = ?", userID, artistID).
		Delete(nil)
	return result.Error
}

// updateArtistFanCount 更新艺术家粉丝数
func (l *ArtistSubscribeLogic) updateArtistFanCount(artistID int64, delta int64) (int64, error) {
	// 使用原子操作更新粉丝数
	result := l.svcCtx.DB.Table("artists").
		Where("id = ?", artistID).
		UpdateColumn("fan_count", gorm.Expr("fan_count + ?", delta))

	if result.Error != nil {
		return 0, result.Error
	}

	// 获取更新后的粉丝数
	var newFanCount int64
	err := l.svcCtx.DB.Table("artists").
		Select("fan_count").
		Where("id = ?", artistID).
		First(&newFanCount).Error

	if err != nil {
		return 0, err
	}

	return newFanCount, nil
}

// ArtistInfo 艺术家信息（内部使用）
type ArtistInfo struct {
	ID        int64
	Name      string
	AvatarURL string
	Bio       string
	FanCount  int64
	SongCount int64
	Status    int16
}
