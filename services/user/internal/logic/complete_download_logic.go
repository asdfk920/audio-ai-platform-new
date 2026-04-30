package logic

import (
	"context"
	"fmt"

	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
)

// CompleteDownloadLogic 完成下载逻辑
type CompleteDownloadLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewCompleteDownloadLogic 创建完成下载逻辑实例
func NewCompleteDownloadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CompleteDownloadLogic {
	return &CompleteDownloadLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// CompleteDownload 完成下载
// 1. 查询下载记录是否存在
// 2. 验证下载状态是否为下载中
// 3. 更新本地路径并将状态标记为已完成
func (l *CompleteDownloadLogic) CompleteDownload(userID int64, req *types.CompleteDownloadReq) (*types.CompleteDownloadResp, error) {
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

	// 更新本地路径并标记为已完成
	if err := l.svcCtx.DownloadRecord.UpdateLocalPath(l.ctx, req.RecordID, userID, req.LocalPath); err != nil {
		return nil, fmt.Errorf("更新下载记录失败: %v", err)
	}

	return &types.CompleteDownloadResp{
		Success: true,
	}, nil
}
