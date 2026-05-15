package logic

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetDeviceDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetDeviceDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDeviceDetailLogic {
	return &GetDeviceDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetDeviceDetailLogic) GetDeviceDetail(req *types.DeviceDetailReq) (resp *types.DeviceDetailResp, err error) {
	userId := l.getUserIdFromCtx()
	if userId == 0 {
		l.Logger.Errorf("GetDeviceDetail: 获取用户ID失败")
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "请重新登录")
	}

	sn := strings.TrimSpace(req.Sn)
	if sn == "" {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "设备序列号不能为空")
	}

	bindRow, err := l.svcCtx.DeviceBind.FindActiveBindBySN(l.ctx, sn)
	if err != nil {
		l.Logger.Errorf("GetDeviceDetail: 查询设备绑定关系失败, userId=%d, sn=%s, err=%v", userId, sn, err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "查询失败")
	}

	if bindRow == nil {
		l.Logger.Errorf("GetDeviceDetail: 未找到活跃绑定记录, userId=%d, sn=%s", userId, sn)

		deviceBindStatus, checkErr := l.svcCtx.DeviceBind.CheckDeviceBindStatus(l.ctx, sn)
		if checkErr != nil {
			l.Logger.Errorf("GetDeviceDetail: 检查设备绑定状态失败, sn=%s, err=%v", sn, checkErr)
		} else if deviceBindStatus != nil {
			l.Logger.Infof("GetDeviceDetail: 设备表绑定状态, sn=%s, bind_status=%d, bound_user_id=%v",
				sn, deviceBindStatus.BindStatus, deviceBindStatus.BoundUserID)
		} else {
			l.Logger.Infof("GetDeviceDetail: 设备不存在于设备表, sn=%s", sn)
		}

		return nil, errorx.NewCodeError(errorx.CodeDeviceNoPermission, "无权限查看该设备")
	}

	if bindRow.UserID != userId {
		l.Logger.Errorf("GetDeviceDetail: 用户无权限查看该设备, userId=%d, bindUserId=%d, sn=%s",
			userId, bindRow.UserID, sn)
		return nil, errorx.NewCodeError(errorx.CodeDeviceNoPermission, "无权限查看该设备")
	}

	device, err := l.queryDeviceBySN(sn)
	if err != nil {
		if err == sql.ErrNoRows {
			l.Logger.Errorf("GetDeviceDetail: 设备不存在, sn=%s", sn)
			return nil, errorx.NewCodeError(errorx.CodeDeviceNotFound, "设备不存在")
		}
		l.Logger.Errorf("GetDeviceDetail: 查询设备详情失败, sn=%s, err=%v", sn, err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "查询失败")
	}

	return &types.DeviceDetailResp{
		Sn:              device.SN,
		DeviceType:      device.ProductKey,
		OnlineStatus:    device.OnlineStatus,
		FirmwareVersion: device.FirmwareVersion,
		HardwareVersion: device.HardwareVersion,
		Manufacturer:    "",
		CreateTime:      "",
		UpdateTime:      "",
	}, nil
}

func (l *GetDeviceDetailLogic) getUserIdFromCtx() int64 {
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

func (l *GetDeviceDetailLogic) queryDeviceBySN(sn string) (*DeviceDetailRow, error) {
	var row DeviceDetailRow
	err := l.svcCtx.DB.QueryRowContext(l.ctx, `
		SELECT sn, product_key, firmware_version, hardware_version, online_status
		FROM public.device
		WHERE sn = $1
		LIMIT 1`, sn).Scan(
		&row.SN, &row.ProductKey, &row.FirmwareVersion, &row.HardwareVersion, &row.OnlineStatus,
	)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

type DeviceDetailRow struct {
	SN              string
	ProductKey      string
	FirmwareVersion string
	HardwareVersion string
	OnlineStatus    int16
}
