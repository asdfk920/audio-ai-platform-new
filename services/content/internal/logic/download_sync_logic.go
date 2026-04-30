package logic

import (
	"context"
	"fmt"
	"time"

	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/content/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// DownloadSyncLogic 下载同步逻辑
type DownloadSyncLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewDownloadSyncLogic 创建下载同步逻辑实例
func NewDownloadSyncLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DownloadSyncLogic {
	return &DownloadSyncLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// AddToSyncQueue 加入同步队列
func (l *DownloadSyncLogic) AddToSyncQueue(req *types.DownloadSyncReq, userID int64) error {
	if userID <= 0 {
		return fmt.Errorf("请先登录")
	}

	if len(req.RecordIDs) == 0 {
		return fmt.Errorf("请选择要同步的记录")
	}

	if len(req.RecordIDs) > 100 {
		return fmt.Errorf("单次最多同步100条记录")
	}

	if l.svcCtx.DB == nil {
		return fmt.Errorf("数据库未就绪")
	}

	for _, recordID := range req.RecordIDs {
		if recordID <= 0 {
			return fmt.Errorf("记录 ID 格式无效")
		}
	}

	var count int64
	err := l.svcCtx.DB.Table("user_downloads").
		Where("id IN ? AND user_id = ? AND sync_status = 0", req.RecordIDs, userID).
		Count(&count).Error
	if err != nil {
		logx.Errorf("查询下载记录失败: %v", err)
		return fmt.Errorf("加入队列失败")
	}

	if count == 0 {
		return fmt.Errorf("记录不存在或已处于同步状态")
	}

	if count != int64(len(req.RecordIDs)) {
		logx.Infof("部分记录不存在或已处于同步状态: 请求%d条，验证通过%d条", len(req.RecordIDs), count)
	}

	result := l.svcCtx.DB.Table("user_downloads").
		Where("id IN ? AND user_id = ? AND sync_status = 0", req.RecordIDs, userID).
		Update("sync_status", 1)
	if result.Error != nil {
		logx.Errorf("更新同步状态失败: %v", result.Error)
		return fmt.Errorf("加入队列失败")
	}

	logx.Infof("加入同步队列成功: userID=%d, recordIDs=%v, count=%d", userID, req.RecordIDs, result.RowsAffected)

	return nil
}

// GetSyncList 获取待同步列表
func (l *DownloadSyncLogic) GetSyncList(userID int64) (*types.DownloadSyncListResp, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("请先登录")
	}

	if l.svcCtx.DB == nil {
		return nil, fmt.Errorf("数据库未就绪")
	}

	type syncItem struct {
		ID           int64
		ContentID    int64
		ContentTitle string
		CoverURL     string
		SyncStatus   int16
		DownloadTime time.Time
	}

	var items []syncItem
	err := l.svcCtx.DB.Table("user_downloads ud").
		Select("ud.id, ud.content_id, ud.content_title, c.cover_url, ud.sync_status, ud.download_time").
		Joins("LEFT JOIN content c ON ud.content_id = c.id").
		Where("ud.user_id = ? AND ud.sync_status IN (1, 2, 4)", userID).
		Order("ud.download_time DESC").
		Find(&items).Error
	if err != nil {
		logx.Errorf("查询待同步列表失败: %v", err)
		return nil, fmt.Errorf("查询失败")
	}

	var list []types.DownloadSyncItem
	for _, item := range items {
		statusText := l.getSyncStatusText(item.SyncStatus)
		list = append(list, types.DownloadSyncItem{
			RecordID:       item.ID,
			ContentID:      item.ContentID,
			ContentName:    item.ContentTitle,
			CoverURL:       item.CoverURL,
			SyncStatus:     item.SyncStatus,
			SyncStatusText: statusText,
			DownloadTime:   item.DownloadTime.Format("2006-01-02 15:04:05"),
		})
	}

	return &types.DownloadSyncListResp{
		Total: int64(len(list)),
		List:  list,
	}, nil
}

// ConfirmSyncDelete 确认删除云端记录
func (l *DownloadSyncLogic) ConfirmSyncDelete(req *types.DownloadSyncConfirmReq, userID int64) (*types.DownloadSyncConfirmResp, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("请先登录")
	}

	if len(req.RecordIDs) == 0 {
		return nil, fmt.Errorf("请选择要删除的记录")
	}

	if len(req.RecordIDs) > 100 {
		return nil, fmt.Errorf("单次最多删除100条记录")
	}

	if l.svcCtx.DB == nil {
		return nil, fmt.Errorf("数据库未就绪")
	}

	for _, recordID := range req.RecordIDs {
		if recordID <= 0 {
			return nil, fmt.Errorf("记录 ID 格式无效")
		}
	}

	var count int64
	err := l.svcCtx.DB.Table("user_downloads").
		Where("id IN ? AND user_id = ? AND sync_status = 1", req.RecordIDs, userID).
		Count(&count).Error
	if err != nil {
		logx.Errorf("查询待同步记录失败: %v", err)
		return nil, fmt.Errorf("删除失败")
	}

	if count == 0 {
		return nil, fmt.Errorf("没有待同步的记录")
	}

	if count != int64(len(req.RecordIDs)) {
		logx.Infof("部分记录不处于待同步状态: 请求%d条，验证通过%d条", len(req.RecordIDs), count)
	}

	result := l.svcCtx.DB.Table("user_downloads").
		Where("id IN ? AND user_id = ? AND sync_status = 1", req.RecordIDs, userID).
		Update("sync_status", 3)
	if result.Error != nil {
		logx.Errorf("更新同步状态失败: %v", result.Error)
		return nil, fmt.Errorf("删除失败")
	}

	logx.Infof("确认删除云端记录成功: userID=%d, recordIDs=%v, deleted=%d", userID, req.RecordIDs, result.RowsAffected)

	return &types.DownloadSyncConfirmResp{
		DeletedCount: result.RowsAffected,
		FailedCount:  count - result.RowsAffected,
	}, nil
}

// CancelSync 取消同步
func (l *DownloadSyncLogic) CancelSync(req *types.DownloadSyncCancelReq, userID int64) error {
	if userID <= 0 {
		return fmt.Errorf("请先登录")
	}

	if len(req.RecordIDs) == 0 {
		return fmt.Errorf("请选择要取消的记录")
	}

	if len(req.RecordIDs) > 100 {
		return fmt.Errorf("单次最多取消100条记录")
	}

	if l.svcCtx.DB == nil {
		return fmt.Errorf("数据库未就绪")
	}

	for _, recordID := range req.RecordIDs {
		if recordID <= 0 {
			return fmt.Errorf("记录 ID 格式无效")
		}
	}

	var count int64
	err := l.svcCtx.DB.Table("user_downloads").
		Where("id IN ? AND user_id = ? AND sync_status IN (1, 2, 4)", req.RecordIDs, userID).
		Count(&count).Error
	if err != nil {
		logx.Errorf("查询同步记录失败: %v", err)
		return fmt.Errorf("取消失败")
	}

	if count == 0 {
		return fmt.Errorf("没有可取消的同步记录")
	}

	result := l.svcCtx.DB.Table("user_downloads").
		Where("id IN ? AND user_id = ? AND sync_status IN (1, 2, 4)", req.RecordIDs, userID).
		Update("sync_status", 0)
	if result.Error != nil {
		logx.Errorf("更新同步状态失败: %v", result.Error)
		return fmt.Errorf("取消失败")
	}

	logx.Infof("取消同步成功: userID=%d, recordIDs=%v, count=%d", userID, req.RecordIDs, result.RowsAffected)

	return nil
}

// getSyncStatusText 获取同步状态文本
func (l *DownloadSyncLogic) getSyncStatusText(status int16) string {
	switch status {
	case 0:
		return "正常"
	case 1:
		return "待同步"
	case 2:
		return "删除中"
	case 3:
		return "已删除"
	case 4:
		return "删除失败"
	default:
		return "未知"
	}
}
