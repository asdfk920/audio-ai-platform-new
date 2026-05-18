package logic

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

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

	sn := strings.TrimSpace(req.Sn)
	if req.ShareId == 0 && sn == "" {
		return errorx.NewCodeError(errorx.CodeInvalidParam, "请提供 share_id 或设备 sn")
	}

	shareID := req.ShareId
	if shareID == 0 {
		id, lookupErr := dao.FindLatestPendingShareIDForReceiverAndDeviceSN(l.ctx, l.svcCtx.DB, userId, sn)
		if errors.Is(lookupErr, sql.ErrNoRows) {
			return errorx.NewCodeError(errorx.CodeDeviceShareNotFound, "未找到待处理的共享邀请")
		}
		if lookupErr != nil {
			l.Logger.Errorf("RejectDeviceShare: 按 SN 查找待接受共享失败, sn=%s, err=%v", sn, lookupErr)
			return errorx.NewCodeError(errorx.CodeInternalError, "系统繁忙，请稍后重试")
		}
		shareID = id
	}

	tx, err := l.svcCtx.DB.BeginTx(l.ctx, nil)
	if err != nil {
		l.Logger.Errorf("RejectDeviceShare: 开启事务失败, err=%v", err)
		return errorx.NewCodeError(errorx.CodeInternalError, "系统繁忙，请稍后重试")
	}
	defer func() { _ = tx.Rollback() }()

	receiverID, shareStatus, findErr := dao.FindDeviceShareQuitSnapshot(l.ctx, tx, shareID)
	if findErr != nil {
		if errors.Is(findErr, sql.ErrNoRows) {
			return errorx.NewCodeError(errorx.CodeDeviceShareNotFound, "共享记录不存在")
		}
		l.Logger.Errorf("RejectDeviceShare: 查询共享记录失败, shareId=%d, err=%v", shareID, findErr)
		return errorx.NewCodeError(errorx.CodeInternalError, "系统繁忙，请稍后重试")
	}

	if receiverID != userId {
		return errorx.NewCodeError(errorx.CodeNoPermission, "无权操作该共享记录")
	}

	if shareStatus != dao.DeviceShareStatusPending {
		return errorx.NewCodeError(errorx.CodeInvalidParam, "仅待接受的邀请可拒绝")
	}

	if updateErr := dao.UpdateDeviceShareStatus(l.ctx, tx, shareID, dao.DeviceShareStatusRejected); updateErr != nil {
		l.Logger.Errorf("RejectDeviceShare: 更新状态失败, err=%v", updateErr)
		return errorx.NewCodeError(errorx.CodeInternalError, "系统繁忙，请稍后重试")
	}

	if commitErr := tx.Commit(); commitErr != nil {
		l.Logger.Errorf("RejectDeviceShare: 提交事务失败, err=%v", commitErr)
		return errorx.NewCodeError(errorx.CodeInternalError, "系统繁忙，请稍后重试")
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
	case json.Number:
		n, _ := id.Int64()
		return n
	case string:
		n, _ := strconv.ParseInt(id, 10, 64)
		return n
	default:
		return 0
	}
}
