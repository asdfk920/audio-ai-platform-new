package logic

import (
	"context"
	"fmt"

	"github.com/jacklau/audio-ai-platform/services/user/internal/repo/dao"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
)

type ListDownloadRecordsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListDownloadRecordsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListDownloadRecordsLogic {
	return &ListDownloadRecordsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListDownloadRecordsLogic) ListDownloadRecords(userID int64, req *types.ListDownloadRecordsReq) (*types.ListDownloadRecordsResp, error) {
	repoReq := &dao.ListDownloadRecordsReq{
		UserID:   userID,
		Status:   req.Status,
		Keyword:  req.Keyword,
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	result, err := l.svcCtx.DownloadRecord.List(l.ctx, repoReq)
	if err != nil {
		return nil, fmt.Errorf("查询下载记录失败: %v", err)
	}

	list := make([]*types.DownloadRecordItem, 0, len(result.List))
	for _, record := range result.List {
		progress := 0
		if record.FileSize > 0 {
			progress = int(float64(record.DownloadedSize) / float64(record.FileSize) * 100)
			if progress > 100 {
				progress = 100
			}
		}

		item := &types.DownloadRecordItem{
			ID:             record.ID,
			ContentID:      record.ContentID,
			FileID:         record.FileID,
			FileName:       record.FileName,
			FileSize:       record.FileSize,
			DownloadedSize: record.DownloadedSize,
			Status:         record.Status,
			Progress:       progress,
			LocalPath:      record.LocalPath,
			CreateTime:     record.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdateTime:     record.UpdatedAt.Format("2006-01-02 15:04:05"),
		}
		list = append(list, item)
	}

	return &types.ListDownloadRecordsResp{
		List:     list,
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	}, nil
}
