package logic

import (
	"context"
	"fmt"

	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
)

// ResumeDownloadLogic 断点续传下载逻辑
type ResumeDownloadLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewResumeDownloadLogic 创建断点续传下载逻辑实例
func NewResumeDownloadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResumeDownloadLogic {
	return &ResumeDownloadLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// ResumeDownload 断点续传下载
// 1. 查询下载记录是否存在
// 2. 验证下载是否已完成（已完成则不允许续传）
// 3. 如果状态为已取消，则更新为下载中
// 4. 返回下载地址、断点位置、文件信息供前端继续下载
func (l *ResumeDownloadLogic) ResumeDownload(userID int64, req *types.ResumeDownloadReq) (*types.ResumeDownloadResp, error) {
	// 查询下载记录
	record, err := l.svcCtx.DownloadRecord.FindByID(l.ctx, req.RecordID, userID)
	if err != nil {
		return nil, fmt.Errorf("查询下载记录失败: %v", err)
	}

	// 验证记录是否存在
	if record == nil {
		return nil, fmt.Errorf("下载记录不存在")
	}

	// 验证下载是否已完成
	if record.Status == "completed" {
		return nil, fmt.Errorf("该下载已完成")
	}

	// 如果状态为已取消，更新为下载中
	if record.Status == "cancelled" {
		if err := l.svcCtx.DownloadRecord.UpdateStatus(l.ctx, req.RecordID, "downloading", record.DownloadedSize); err != nil {
			return nil, fmt.Errorf("更新下载状态失败: %v", err)
		}
	}

	// 生成下载地址
	downloadURL := fmt.Sprintf("/api/v1/user/content/download/stream?content_id=%d", record.ContentID)

	// 返回断点续传信息
	return &types.ResumeDownloadResp{
		DownloadURL:    downloadURL,
		RangeStart:     record.DownloadedSize,
		FileName:       record.FileName,
		FileSize:       record.FileSize,
		DownloadedSize: record.DownloadedSize,
	}, nil
}
