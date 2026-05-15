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

type BindDeviceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewBindDeviceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BindDeviceLogic {
	return &BindDeviceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *BindDeviceLogic) BindDevice(req *types.BindDeviceReq) (resp *types.BindDeviceResp, err error) {
	userId := l.getUserIdFromCtx()
	if userId == 0 {
		l.Logger.Errorf("BindDevice: 获取用户ID失败")
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "请重新登录")
	}

	if err := l.validateParams(req); err != nil {
		return nil, err
	}

	deviceID, err := l.checkDeviceExists(req.Sn)
	if err != nil {
		return nil, err
	}

	if err := l.checkBindStatus(userId, deviceID, req.Sn); err != nil {
		return nil, err
	}

	if err := l.executeBind(userId, deviceID, req.Sn, req.DeviceName); err != nil {
		l.Logger.Errorf("BindDevice: 绑定失败, userId=%d, sn=%s, err=%v", userId, req.Sn, err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "绑定失败")
	}

	l.Logger.Infof("BindDevice: 绑定成功, userId=%d, sn=%s, deviceName=%s", userId, req.Sn, req.DeviceName)

	return &types.BindDeviceResp{
		UserId:     userId,
		DeviceSn:   req.Sn,
		DeviceName: req.DeviceName,
		BindTime:   time.Now().Format("2006-01-02 15:04:05"),
	}, nil
}

func (l *BindDeviceLogic) getUserIdFromCtx() int64 {
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

func (l *BindDeviceLogic) validateParams(req *types.BindDeviceReq) error {
	sn := strings.TrimSpace(req.Sn)
	if sn == "" {
		return errorx.NewCodeError(errorx.CodeInvalidParam, "设备序列号不能为空")
	}

	if !l.isValidSN(sn) {
		return errorx.NewCodeError(errorx.CodeDeviceSnInvalid, "设备序列号格式错误")
	}

	deviceName := strings.TrimSpace(req.DeviceName)
	if deviceName == "" {
		return errorx.NewCodeError(errorx.CodeInvalidParam, "设备名称不能为空")
	}

	req.Sn = sn
	req.DeviceName = deviceName

	return nil
}

func (l *BindDeviceLogic) isValidSN(sn string) bool {
	if len(sn) < 6 || len(sn) > 64 {
		return false
	}
	matched, _ := regexp.MatchString(`^[A-Za-z0-9_-]+$`, sn)
	return matched
}

func (l *BindDeviceLogic) checkDeviceExists(sn string) (int64, error) {
	deviceID, ok, err := l.svcCtx.DeviceBind.FindDeviceIDBySN(l.ctx, sn)
	if err != nil {
		l.Logger.Errorf("BindDevice: 查询设备失败, sn=%s, err=%v", sn, err)
		return 0, errorx.NewCodeError(errorx.CodeDatabaseError, "查询设备失败")
	}
	if !ok {
		return 0, errorx.NewCodeError(errorx.CodeDeviceNotFound, "设备不存在")
	}
	return deviceID, nil
}

func (l *BindDeviceLogic) checkBindStatus(userId, deviceID int64, sn string) error {
	existingBind, err := l.svcCtx.DeviceBind.FindActiveBindByDeviceID(l.ctx, deviceID)
	if err != nil {
		l.Logger.Errorf("BindDevice: 查询绑定状态失败, deviceID=%d, err=%v", deviceID, err)
		return errorx.NewCodeError(errorx.CodeDatabaseError, "查询绑定状态失败")
	}

	if existingBind != nil {
		if existingBind.UserID == userId {
			return errorx.NewCodeError(errorx.CodeDeviceExists, "已绑定该设备")
		}
		return errorx.NewCodeError(errorx.CodeDeviceBoundByOther, "设备已被他人绑定")
	}

	return nil
}

func (l *BindDeviceLogic) executeBind(userId, deviceID int64, sn, deviceName string) error {
	if err := l.svcCtx.DeviceBind.BindDeviceWithTransaction(l.ctx, userId, deviceID, sn, deviceName); err != nil {
		l.Logger.Errorf("BindDevice: 绑定事务失败, userId=%d, deviceID=%d, sn=%s, err=%v", userId, deviceID, sn, err)
		return errorx.NewCodeError(errorx.CodeDatabaseError, "绑定失败：数据库操作异常")
	}

	return nil
}
