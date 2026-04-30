package logic

import (
	"context"
	"fmt"

	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
)

// DeleteDownloadRecordLogic 删除下载记录逻辑
// 处理用户主动删除单条下载记录的业务逻辑
type DeleteDownloadRecordLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewDeleteDownloadRecordLogic 创建删除下载记录逻辑实例
// 参数：
//   - ctx: 请求上下文
//   - svcCtx: 服务上下文，包含数据库连接等依赖
func NewDeleteDownloadRecordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteDownloadRecordLogic {
	return &DeleteDownloadRecordLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// DeleteDownloadRecord 删除单条下载记录（软删除）
// 流程：
//  1. 根据 record_id 和 user_id 查询下载记录是否存在
//  2. 验证记录归属当前用户（防止越权删除）
//  3. 软删除记录（标记 is_deleted = TRUE，不物理删除数据）
//
// 参数：
//   - userID: 当前登录用户 ID
//   - req: 删除请求，包含 record_id
//
// 返回：
//   - 成功时返回 Success: true
//   - 记录不存在或不属于当前用户时返回错误
func (l *DeleteDownloadRecordLogic) DeleteDownloadRecord(userID int64, req *types.DeleteDownloadRecordReq) (*types.DeleteDownloadRecordResp, error) {
	// 查询下载记录
	record, err := l.svcCtx.DownloadRecord.FindByID(l.ctx, req.RecordID, userID)
	if err != nil {
		return nil, fmt.Errorf("查询下载记录失败: %v", err)
	}

	// 验证记录是否存在
	if record == nil {
		return nil, fmt.Errorf("下载记录不存在")
	}

	// 软删除记录
	if err := l.svcCtx.DownloadRecord.Delete(l.ctx, req.RecordID, userID); err != nil {
		return nil, fmt.Errorf("删除下载记录失败: %v", err)
	}

	return &types.DeleteDownloadRecordResp{
		Success: true,
	}, nil
}
