package logic

import (
	"context"
	"fmt"

	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
)

// SyncDeleteDownloadRecordsLogic 批量同步删除下载记录逻辑
// 处理 App 清理本地缓存后，同步删除云端对应下载记录的业务逻辑
type SyncDeleteDownloadRecordsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewSyncDeleteDownloadRecordsLogic 创建批量同步删除下载记录逻辑实例
// 参数：
//   - ctx: 请求上下文
//   - svcCtx: 服务上下文，包含数据库连接等依赖
func NewSyncDeleteDownloadRecordsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SyncDeleteDownloadRecordsLogic {
	return &SyncDeleteDownloadRecordsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// SyncDeleteDownloadRecords 批量同步删除下载记录（软删除）
// 流程：
//  1. 验证 file_ids 列表不为空，为空则直接返回删除数量为 0
//  2. 调用 Repository 层批量软删除指定 file_id 的下载记录
//  3. 返回实际删除的记录数量
//
// 参数：
//   - userID: 当前登录用户 ID（确保只删除当前用户的记录）
//   - req: 批量删除请求，包含 file_ids 数组
//
// 返回：
//   - 成功时返回 DeletedCount（实际删除的记录数）
//   - 数据库操作失败时返回错误
func (l *SyncDeleteDownloadRecordsLogic) SyncDeleteDownloadRecords(userID int64, req *types.SyncDeleteDownloadRecordsReq) (*types.SyncDeleteDownloadRecordsResp, error) {
	// 验证 file_ids 不为空
	if len(req.FileIDs) == 0 {
		return &types.SyncDeleteDownloadRecordsResp{
			DeletedCount: 0,
		}, nil
	}

	// 批量软删除
	deletedCount, err := l.svcCtx.DownloadRecord.BatchDeleteByFileIDs(l.ctx, userID, req.FileIDs)
	if err != nil {
		return nil, fmt.Errorf("批量删除下载记录失败: %v", err)
	}

	return &types.SyncDeleteDownloadRecordsResp{
		DeletedCount: deletedCount,
	}, nil
}
