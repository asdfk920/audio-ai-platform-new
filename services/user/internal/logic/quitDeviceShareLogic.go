// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

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

type QuitDeviceShareLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewQuitDeviceShareLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QuitDeviceShareLogic {
	return &QuitDeviceShareLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// QuitDeviceShare 被共享方主动退出：share_id 或设备 sn 二选一。
func (l *QuitDeviceShareLogic) QuitDeviceShare(req *types.DeviceShareQuitReq) error {
	userId := l.getUserIdFromCtx()
	if userId == 0 {
		return errorx.NewCodeError(errorx.CodeTokenInvalid, "登录已过期")
	}

	sn := strings.TrimSpace(req.Sn)
	if req.ShareId == 0 && sn == "" {
		return errorx.NewCodeError(errorx.CodeInvalidParam, "请提供 share_id 或设备 sn")
	}

	shareID := req.ShareId
	if shareID == 0 {
		id, lookupErr := dao.FindLatestShareIDForReceiverAndDeviceSN(l.ctx, l.svcCtx.DB, userId, sn)
		if errors.Is(lookupErr, sql.ErrNoRows) {
			return errorx.NewCodeError(errorx.CodeDeviceShareNotFound, "未找到可退出的共享记录")
		}
		if lookupErr != nil {
			l.Logger.Errorf("QuitDeviceShare: 按 SN 查找共享失败, sn=%s, err=%v", sn, lookupErr)
			return errorx.NewCodeError(errorx.CodeInternalError, "系统繁忙，请稍后重试")
		}
		shareID = id
	}

	tx, err := l.svcCtx.DB.BeginTx(l.ctx, nil)
	if err != nil {
		l.Logger.Errorf("QuitDeviceShare: 开启事务失败, err=%v", err)
		return errorx.NewCodeError(errorx.CodeInternalError, "系统繁忙，请稍后重试")
	}
	defer func() { _ = tx.Rollback() }()

	receiverID, shareStatus, findErr := dao.FindDeviceShareQuitSnapshot(l.ctx, tx, shareID)
	if findErr != nil {
		if errors.Is(findErr, sql.ErrNoRows) {
			return errorx.NewCodeError(errorx.CodeDeviceShareNotFound, "共享记录不存在")
		}
		l.Logger.Errorf("QuitDeviceShare: 查询共享失败, shareId=%d, err=%v", shareID, findErr)
		return errorx.NewCodeError(errorx.CodeInternalError, "系统繁忙，请稍后重试")
	}

	if receiverID != userId {
		return errorx.NewCodeError(errorx.CodeDeviceShareNoPermission, "仅被共享人可退出该共享")
	}

	switch shareStatus {
	case dao.DeviceShareStatusRevoked, dao.DeviceShareStatusRejected, dao.DeviceShareStatusExpired, dao.DeviceShareStatusQuit:
		return errorx.NewCodeError(errorx.CodeDeviceShareAlreadyCanceled, "该共享已结束，无需退出")
	}

	if shareStatus != dao.DeviceShareStatusActive && shareStatus != dao.DeviceShareStatusPending {
		return errorx.NewCodeError(errorx.CodeInvalidParam, "当前状态不可退出")
	}

	if updErr := dao.UpdateDeviceShareStatus(l.ctx, tx, shareID, dao.DeviceShareStatusQuit); updErr != nil {
		l.Logger.Errorf("QuitDeviceShare: 更新状态失败, err=%v", updErr)
		return errorx.NewCodeError(errorx.CodeInternalError, "系统繁忙，请稍后重试")
	}

	if commitErr := tx.Commit(); commitErr != nil {
		l.Logger.Errorf("QuitDeviceShare: 提交事务失败, err=%v", commitErr)
		return errorx.NewCodeError(errorx.CodeInternalError, "系统繁忙，请稍后重试")
	}
	return nil
}

func (l *QuitDeviceShareLogic) getUserIdFromCtx() int64 {
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
