package logic

import (
	"context"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/repo/dao"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RejectDeviceShareLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRejectDeviceShareLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RejectDeviceShareLogic {
	return &RejectDeviceShareLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RejectDeviceShareLogic) RejectDeviceShare(req *types.DeviceShareRejectReq) error {
	userId := l.getUserIdFromCtx()
	if userId == 0 {
		l.Logger.Errorf("RejectDeviceShare: 获取用户ID失败")
		return errorx.NewCodeError(errorx.CodeTokenInvalid, "请重新登录")
	}

	if req.ShareId == 0 {
		return errorx.NewCodeError(errorx.CodeInvalidParam, "共享记录ID不能为空")
	}

	tx, err := l.svcCtx.DB.BeginTx(l.ctx, nil)
	if err != nil {
		l.Logger.Errorf("RejectDeviceShare: 开启事务失败, err=%v", err)
		return err
	}
	defer func() { _ = tx.Rollback() }()

	share, findErr := dao.FindDeviceShareByID(l.ctx, tx, req.ShareId)
	if findErr != nil {
		l.Logger.Errorf("RejectDeviceShare: 查询共享记录失败, shareId=%d, err=%v", req.ShareId, findErr)
		return findErr
	}
	if share == nil {
		return errorx.NewCodeError(errorx.CodeInvalidParam, "共享记录不存在")
	}

	if share.Status != dao.DeviceShareStatusPending {
		return errorx.NewCodeError(errorx.CodeInvalidParam, "该共享记录不是待确认状态")
	}

	if share.SharedUserID != userId {
		return errorx.NewCodeError(errorx.CodeNoPermission, "无权操作该共享记录")
	}

	updateErr := dao.UpdateDeviceShareStatus(l.ctx, tx, req.ShareId, dao.DeviceShareStatusRejected)
	if updateErr != nil {
		l.Logger.Errorf("RejectDeviceShare: 更新状态失败, err=%v", updateErr)
		return updateErr
	}

	if commitErr := tx.Commit(); commitErr != nil {
		l.Logger.Errorf("RejectDeviceShare: 提交事务失败, err=%v", commitErr)
		return commitErr
	}

	return nil
}

func (l *RejectDeviceShareLogic) getUserIdFromCtx() int64 {
	v := l.ctx.Value("userId")
	if v == nil {
		return 0
	}
	switch id := v.(type) {
	case int64:
		return id
	case int:
		return int64(id)
	case float64:
		return int64(id)
	default:
		return 0
	}
}
