package logic

import (
	"context"
	"fmt"

	"github.com/jacklau/audio-ai-platform/services/user/internal/repo/dao"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
)

// CreateDownloadRecordLogic 创建下载记录逻辑
type CreateDownloadRecordLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewCreateDownloadRecordLogic 创建下载记录逻辑实例
func NewCreateDownloadRecordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateDownloadRecordLogic {
	return &CreateDownloadRecordLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// CreateDownloadRecord 创建下载记录
// 1. 先查询是否已有未完成的下载记录，如果有则返回断点位置实现续传
// 2. 如果没有则创建新的下载记录，初始状态为下载中
// 3. 返回记录ID、下载地址和起始下载位置（断点位置）
func (l *CreateDownloadRecordLogic) CreateDownloadRecord(userID int64, req *types.CreateDownloadRecordReq) (*types.CreateDownloadRecordResp, error) {
	// 查询是否已有未完成的下载记录
	existing, err := l.svcCtx.DownloadRecord.FindByContentID(l.ctx, userID, req.ContentID)
	if err != nil {
		return nil, fmt.Errorf("查询下载记录失败: %v", err)
	}

	// 如果存在未完成的记录，返回断点位置供续传
	if existing != nil && existing.Status != "completed" {
		rangeStart := existing.DownloadedSize
		downloadURL := fmt.Sprintf("/api/v1/user/content/download/stream?content_id=%d", req.ContentID)
		return &types.CreateDownloadRecordResp{
			RecordID:    existing.ID,
			DownloadURL: downloadURL,
			RangeStart:  rangeStart,
		}, nil
	}

	// 创建新的下载记录
	record := &dao.UserDownloadRecord{
		UserID:         userID,
		ContentID:      req.ContentID,
		FileID:         req.ContentID,
		FileName:       req.FileName,
		FileSize:       req.FileSize,
		DownloadedSize: 0,
		Status:         "downloading",
		LocalPath:      "",
	}

	if err := l.svcCtx.DownloadRecord.Create(l.ctx, record); err != nil {
		return nil, fmt.Errorf("创建下载记录失败: %v", err)
	}

	// 生成下载地址
	downloadURL := fmt.Sprintf("/api/v1/user/content/download/stream?content_id=%d", req.ContentID)

	return &types.CreateDownloadRecordResp{
		RecordID:    record.ID,
		DownloadURL: downloadURL,
		RangeStart:  0,
	}, nil
}
