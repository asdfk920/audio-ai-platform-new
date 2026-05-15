package logic

import (
	"context"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type UnbindDeviceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUnbindDeviceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UnbindDeviceLogic {
	return &UnbindDeviceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UnbindDeviceLogic) UnbindDevice(req *types.UnbindDeviceReq) (resp *types.UnbindDeviceResp, err error) {
	userId := l.getUserIdFromCtx()
	if userId == 0 {
		l.Logger.Errorf("UnbindDevice: 获取用户ID失败")
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "请重新登录")
	}

	if err := l.validateParams(req); err != nil {
		return nil, err
	}

	deviceID, err := l.checkDeviceExists(req.Sn)
	if err != nil {
		return nil, err
	}

	if err := l.checkUserBinding(userId, deviceID, req.Sn); err != nil {
		return nil, err
	}

	if err := l.executeUnbind(userId, deviceID); err != nil {
		l.Logger.Errorf("UnbindDevice: 解绑失败, userId=%d, sn=%s, err=%v", userId, req.Sn, err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "解绑失败")
	}

	l.Logger.Infof("UnbindDevice: 解绑成功, userId=%d, sn=%s", userId, req.Sn)

	return &types.UnbindDeviceResp{
		UserId:     userId,
		DeviceSn:   req.Sn,
		UnbindTime: time.Now().Format("2006-01-02 15:04:05"),
	}, nil
}

func (l *UnbindDeviceLogic) getUserIdFromCtx() int64 {
	v := l.ctx.Value("userId")
	if v == nil {
		return 0
	}
	switch id := v.(type) {
	case json.Number:
		n, err := id.Int64()
		if err != nil {
			return 0
		}
		return n
	case float64:
		return int64(id)
	case int64:
		return id
	case int:
		return int64(id)
	case string:
		n, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			return 0
		}
		return n
	default:
		return 0
	}
}

func (l *UnbindDeviceLogic) validateParams(req *types.UnbindDeviceReq) error {
	sn := strings.TrimSpace(req.Sn)
	if sn == "" {
		return errorx.NewCodeError(errorx.CodeInvalidParam, "设备序列号不能为空")
	}

	if !l.isValidSN(sn) {
		return errorx.NewCodeError(errorx.CodeDeviceSnInvalid, "设备序列号格式错误")
	}

	req.Sn = sn
	return nil
}

func (l *UnbindDeviceLogic) isValidSN(sn string) bool {
	if len(sn) < 6 || len(sn) > 64 {
		return false
	}
	matched, _ := regexp.MatchString(`^[A-Za-z0-9_-]+$`, sn)
	return matched
}

func (l *UnbindDeviceLogic) checkDeviceExists(sn string) (int64, error) {
	deviceID, ok, err := l.svcCtx.DeviceBind.FindDeviceIDBySN(l.ctx, sn)
	if err != nil {
		l.Logger.Errorf("UnbindDevice: 查询设备失败, sn=%s, err=%v", sn, err)
		return 0, errorx.NewCodeError(errorx.CodeDatabaseError, "查询设备失败")
	}
	if !ok {
		return 0, errorx.NewCodeError(errorx.CodeDeviceNotFound, "设备不存在")
	}
	return deviceID, nil
}

func (l *UnbindDeviceLogic) checkUserBinding(userId, deviceID int64, sn string) error {
	bindRow, err := l.svcCtx.DeviceBind.FindActiveBindBySN(l.ctx, sn)
	if err != nil {
		l.Logger.Errorf("UnbindDevice: 查询绑定状态失败, sn=%s, err=%v", sn, err)
		return errorx.NewCodeError(errorx.CodeDatabaseError, "查询绑定状态失败")
	}

	if bindRow == nil || bindRow.UserID != userId {
		return errorx.NewCodeError(errorx.CodeDeviceNotBound, "你未绑定该设备，无法解绑")
	}

	return nil
}

func (l *UnbindDeviceLogic) executeUnbind(userId, deviceID int64) error {
	affected, err := l.svcCtx.DeviceBind.UnbindDeviceWithTransaction(l.ctx, userId, deviceID)
	if err != nil {
		l.Logger.Errorf("UnbindDevice: 解绑事务失败, userId=%d, deviceID=%d, err=%v", userId, deviceID, err)
		return errorx.NewCodeError(errorx.CodeDatabaseError, "解绑失败：数据库操作异常")
	}
	if affected == 0 {
		return errorx.NewCodeError(errorx.CodeDeviceNotBound, "你未绑定该设备，无法解绑")
	}

	return nil
}
