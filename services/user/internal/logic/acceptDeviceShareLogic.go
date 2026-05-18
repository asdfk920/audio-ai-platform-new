package logic

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

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

// AcceptDeviceShare 单条同意：share_id 或 sn 二选一。
func (l *AcceptDeviceShareLogic) AcceptDeviceShare(req *types.DeviceShareAcceptReq) (*types.DeviceShareItem, error) {
	userId := l.getUserIdFromCtx()
	if userId == 0 {
		l.Logger.Errorf("AcceptDeviceShare: 获取用户ID失败")
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "请重新登录")
	}

	sn := strings.TrimSpace(req.Sn)
	shareID := req.ShareId
	if shareID == 0 && sn == "" {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "请提供 share_id 或设备 sn")
	}

	return l.acceptOneShare(userId, shareID, sn)
}

// AcceptDeviceSharesBatch 批量同意：前端传入收到的设备列表（每项含 share_id 和/或 sn）。
func (l *AcceptDeviceShareLogic) AcceptDeviceSharesBatch(devices []types.DeviceShareAcceptDeviceRef) (*types.DeviceShareAcceptBatchResp, error) {
	userId := l.getUserIdFromCtx()
	if userId == 0 {
		l.Logger.Errorf("AcceptDeviceSharesBatch: 获取用户ID失败")
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "请重新登录")
	}
	if len(devices) == 0 {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "设备列表不能为空")
	}

	out := &types.DeviceShareAcceptBatchResp{
		Accepted: make([]types.DeviceShareItem, 0, len(devices)),
		Failed:   make([]types.DeviceShareAcceptFailItem, 0),
	}
	for _, d := range devices {
		sid := d.ShareId
		sn := strings.TrimSpace(d.Sn)
		if sid == 0 && sn == "" {
			out.Failed = append(out.Failed, types.DeviceShareAcceptFailItem{
				Code: errorx.CodeInvalidParam,
				Msg:  "share_id 与 sn 不能同时为空",
			})
			continue
		}
		item, err := l.acceptOneShare(userId, sid, sn)
		if err != nil {
			var ce *errorx.CodeError
			if errors.As(err, &ce) {
				out.Failed = append(out.Failed, types.DeviceShareAcceptFailItem{
					ShareId: sid, Sn: sn, Code: ce.Code, Msg: ce.Msg,
				})
			} else {
				out.Failed = append(out.Failed, types.DeviceShareAcceptFailItem{
					ShareId: sid, Sn: sn, Code: errorx.CodeInternalError, Msg: "系统繁忙，请稍后重试",
				})
			}
			continue
		}
		out.Accepted = append(out.Accepted, *item)
	}
	return out, nil
}

func (l *AcceptDeviceShareLogic) acceptOneShare(userId int64, shareID int64, sn string) (*types.DeviceShareItem, error) {
	if shareID == 0 {
		id, lookupErr := dao.FindLatestShareIDForReceiverAndDeviceSN(l.ctx, l.svcCtx.DB, userId, sn)
		if errors.Is(lookupErr, sql.ErrNoRows) {
			return nil, errorx.NewCodeError(errorx.CodeDeviceShareNotFound, "未找到该设备的待接受共享")
		}
		if lookupErr != nil {
			l.Logger.Errorf("acceptOneShare: 按 SN 查找共享失败, sn=%s, err=%v", sn, lookupErr)
			return nil, errorx.NewCodeError(errorx.CodeInternalError, "系统繁忙，请稍后重试")
		}
		shareID = id
	}

	tx, err := l.svcCtx.DB.BeginTx(l.ctx, nil)
	if err != nil {
		l.Logger.Errorf("acceptOneShare: 开启事务失败, err=%v", err)
		return nil, errorx.NewCodeError(errorx.CodeInternalError, "系统繁忙，请稍后重试")
	}
	defer func() { _ = tx.Rollback() }()

	share, findErr := dao.FindDeviceShareByID(l.ctx, tx, shareID)
	if findErr != nil {
		l.Logger.Errorf("acceptOneShare: 查询共享记录失败, shareId=%d, err=%v", shareID, findErr)
		return nil, errorx.NewCodeError(errorx.CodeInternalError, "系统繁忙，请稍后重试")
	}
	if share == nil {
		return nil, errorx.NewCodeError(errorx.CodeDeviceShareNotFound, "共享记录不存在")
	}

	if share.SharedUserID != userId {
		return nil, errorx.NewCodeError(errorx.CodeNoPermission, "无权操作该共享记录")
	}

	if share.EndAt.Valid && share.EndAt.Time.Before(time.Now()) {
		return nil, errorx.NewCodeError(errorx.CodeDeviceShareExpired, "")
	}

	if share.Status == dao.DeviceShareStatusActive {
		return deviceShareItemFromView(share), nil
	}

	if share.Status != dao.DeviceShareStatusPending {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "该共享已失效或不可接受")
	}

	if updErr := dao.UpdateDeviceShareAccepted(l.ctx, tx, shareID); updErr != nil {
		l.Logger.Errorf("acceptOneShare: 更新为已接受失败, err=%v", updErr)
		return nil, errorx.NewCodeError(errorx.CodeInternalError, "系统繁忙，请稍后重试")
	}

	if commitErr := tx.Commit(); commitErr != nil {
		l.Logger.Errorf("acceptOneShare: 提交事务失败, err=%v", commitErr)
		return nil, errorx.NewCodeError(errorx.CodeInternalError, "系统繁忙，请稍后重试")
	}

	return deviceShareItemFromView(share), nil
}

func deviceShareItemFromView(share *dao.DeviceShareViewRow) *types.DeviceShareItem {
	return &types.DeviceShareItem{
		ShareId:    share.ID,
		Sn:         share.DeviceSN,
		ShareTo:    share.TargetAccount,
		ExpireDays: 0,
		Status:     dao.DeviceShareStatusToLegacyInt(dao.DeviceShareStatusActive),
		CreatedAt:  share.CreatedAt.Format("2006-01-02 15:04:05"),
		EndAt:      formatNullTime(share.EndAt),
	}
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
