package logic

import (
	"context"
	"fmt"

	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
)

// UpdateDownloadProgressLogic 更新下载进度逻辑
type UpdateDownloadProgressLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewUpdateDownloadProgressLogic 创建更新下载进度逻辑实例
func NewUpdateDownloadProgressLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateDownloadProgressLogic {
	return &UpdateDownloadProgressLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// UpdateDownloadProgress 更新下载进度
// 1. 查询下载记录是否存在
// 2. 验证下载状态是否为下载中
// 3. 验证已下载大小不超过文件总大小
// 4. 更新已下载大小到数据库
func (l *UpdateDownloadProgressLogic) UpdateDownloadProgress(userID int64, req *types.UpdateDownloadProgressReq) (*types.UpdateDownloadProgressResp, error) {
	// 查询下载记录
	record, err := l.svcCtx.DownloadRecord.FindByID(l.ctx, req.RecordID, userID)
	if err != nil {
		return nil, fmt.Errorf("查询下载记录失败: %v", err)
	}

	// 验证记录是否存在
	if record == nil {
		return nil, fmt.Errorf("下载记录不存在")
	}

	// 验证下载状态
	if record.Status != "downloading" {
		return nil, fmt.Errorf("下载状态异常，当前状态: %s", record.Status)
	}

	// 验证已下载大小
	if req.DownloadedSize > record.FileSize {
		return nil, fmt.Errorf("已下载大小不能超过文件总大小")
	}

	// 更新已下载大小
	if err := l.svcCtx.DownloadRecord.UpdateDownloadedSize(l.ctx, req.RecordID, userID, req.DownloadedSize); err != nil {
		return nil, fmt.Errorf("更新下载进度失败: %v", err)
	}

	return &types.UpdateDownloadProgressResp{
		Success: true,
	}, nil
}
