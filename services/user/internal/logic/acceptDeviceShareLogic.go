package logic

import (
	"context"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/repo/dao"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AcceptDeviceShareLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAcceptDeviceShareLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AcceptDeviceShareLogic {
	return &AcceptDeviceShareLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AcceptDeviceShareLogic) AcceptDeviceShare(req *types.DeviceShareAcceptReq) (resp *types.DeviceShareItem, err error) {
	userId := l.getUserIdFromCtx()
	if userId == 0 {
		l.Logger.Errorf("AcceptDeviceShare: 获取用户ID失败")
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "请重新登录")
	}

	if req.ShareId == 0 {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "共享记录ID不能为空")
	}

	tx, err := l.svcCtx.DB.BeginTx(l.ctx, nil)
	if err != nil {
		l.Logger.Errorf("AcceptDeviceShare: 开启事务失败, err=%v", err)
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	share, findErr := dao.FindDeviceShareByID(l.ctx, tx, req.ShareId)
	if findErr != nil {
		l.Logger.Errorf("AcceptDeviceShare: 查询共享记录失败, shareId=%d, err=%v", req.ShareId, findErr)
		return nil, findErr
	}
	if share == nil {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "共享记录不存在")
	}

	if share.Status != dao.DeviceShareStatusPending {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "该共享记录不是待确认状态")
	}

	if share.SharedUserID != userId {
		return nil, errorx.NewCodeError(errorx.CodeNoPermission, "无权操作该共享记录")
	}

	updateErr := dao.UpdateDeviceShareStatus(l.ctx, tx, req.ShareId, dao.DeviceShareStatusActive)
	if updateErr != nil {
		l.Logger.Errorf("AcceptDeviceShare: 更新状态失败, err=%v", updateErr)
		return nil, updateErr
	}

	if commitErr := tx.Commit(); commitErr != nil {
		l.Logger.Errorf("AcceptDeviceShare: 提交事务失败, err=%v", commitErr)
		return nil, commitErr
	}

	return &types.DeviceShareItem{
		ShareId:    share.ID,
		Sn:         share.DeviceSN,
		FromUserId: share.OwnerUserID,
		ToUserId:   share.SharedUserID,
		ToAccount:  share.TargetAccount,
		Status:     1,
		CreatedAt:  share.CreatedAt.Format("2006-01-02 15:04:05"),
		EndAt:      formatNullTime(share.EndAt),
	}, nil
}

func (l *AcceptDeviceShareLogic) getUserIdFromCtx() int64 {
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
