package logic

import (
	"context"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/repo/dao"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CancelDeviceShareLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCancelDeviceShareLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CancelDeviceShareLogic {
	return &CancelDeviceShareLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CancelDeviceShareLogic) CancelDeviceShare(req *types.DeviceShareCancelReq) error {
	userId := l.getUserIdFromCtx()
	if userId == 0 {
		l.Logger.Errorf("CancelDeviceShare: 获取用户ID失败")
		return errorx.NewCodeError(errorx.CodeTokenInvalid, "请重新登录")
	}

	if req.ShareId == 0 {
		return errorx.NewCodeError(errorx.CodeInvalidParam, "共享记录ID不能为空")
	}

	tx, err := l.svcCtx.DB.BeginTx(l.ctx, nil)
	if err != nil {
		l.Logger.Errorf("CancelDeviceShare: 开启事务失败, err=%v", err)
		return err
	}
	defer func() { _ = tx.Rollback() }()

	share, findErr := dao.FindDeviceShareByID(l.ctx, tx, req.ShareId)
	if findErr != nil {
		l.Logger.Errorf("CancelDeviceShare: 查询共享记录失败, shareId=%d, err=%v", req.ShareId, findErr)
		return findErr
	}
	if share == nil {
		return errorx.NewCodeError(errorx.CodeInvalidParam, "共享记录不存在")
	}

	if share.OwnerUserID != userId {
		return errorx.NewCodeError(errorx.CodeNoPermission, "只有发起人才能撤销共享")
	}

	if share.Status != dao.DeviceShareStatusPending && share.Status != dao.DeviceShareStatusActive {
		return errorx.NewCodeError(errorx.CodeInvalidParam, "该共享记录无法撤销")
	}

	updateErr := dao.UpdateDeviceShareStatus(l.ctx, tx, req.ShareId, dao.DeviceShareStatusRevoked)
	if updateErr != nil {
		l.Logger.Errorf("CancelDeviceShare: 更新状态失败, err=%v", updateErr)
		return updateErr
	}

	if commitErr := tx.Commit(); commitErr != nil {
		l.Logger.Errorf("CancelDeviceShare: 提交事务失败, err=%v", commitErr)
		return commitErr
	}

	return nil
}

func (l *CancelDeviceShareLogic) getUserIdFromCtx() int64 {
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
