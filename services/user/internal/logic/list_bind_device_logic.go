package logic

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type ListBindDeviceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListBindDeviceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListBindDeviceLogic {
	return &ListBindDeviceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListBindDeviceLogic) ListBindDevice() (resp *types.DeviceListResp, err error) {
	userId := l.getUserIdFromCtx()
	if userId == 0 {
		l.Logger.Errorf("ListBindDevice: 获取用户ID失败")
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "请重新登录")
	}

	items, err := l.svcCtx.DeviceBind.ListActiveByUserID(l.ctx, userId, "", "", "")
	if err != nil {
		l.Logger.Errorf("ListBindDevice: 查询绑定设备列表失败, userId=%d, err=%v", userId, err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "查询失败")
	}

	list := make([]types.DeviceListItem, 0, len(items))
	for _, item := range items {
		list = append(list, types.DeviceListItem{
			Sn:         item.DeviceSn,
			DeviceName: item.DeviceName,
		})
	}

	return &types.DeviceListResp{
		List: list,
	}, nil
}

func (l *ListBindDeviceLogic) getUserIdFromCtx() int64 {
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
