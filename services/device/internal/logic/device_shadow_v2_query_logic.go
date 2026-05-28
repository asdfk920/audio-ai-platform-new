package logic

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	shadowv2 "github.com/jacklau/audio-ai-platform/services/device/internal/device/shadowv2"
	"github.com/jacklau/audio-ai-platform/services/device/internal/shadowsvc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
)

type DeviceShadowV2QueryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeviceShadowV2QueryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeviceShadowV2QueryLogic {
	return &DeviceShadowV2QueryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Query 查询设备影子：仅从 PostgreSQL device_shadow 表读取 reported
func (l *DeviceShadowV2QueryLogic) Query(deviceSN string) (*types.DeviceShadowV2QueryResp, error) {
	deviceSN = strings.ToUpper(strings.TrimSpace(deviceSN))
	if deviceSN == "" {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "device_sn 不能为空")
	}

	view, err := shadowsvc.New(l.svcCtx).GetShadowViewBySNFromDB(l.ctx, deviceSN)
	if err != nil {
		return nil, err
	}
	return viewToQueryResp(view), nil
}

func viewToQueryResp(view *shadowsvc.View) *types.DeviceShadowV2QueryResp {
	if view == nil {
		return &types.DeviceShadowV2QueryResp{Message: "shadow not found"}
	}

	reported := decodeJSONMap(view.Reported)

	status := string(shadowv2.StatusOffline)
	if view.Online {
		status = string(shadowv2.StatusOnline)
	}

	var updateTime int64
	if view.LastReportTime != nil {
		updateTime = view.LastReportTime.UnixMilli()
	}

	return &types.DeviceShadowV2QueryResp{
		DeviceSN:   view.DeviceSN,
		Reported:   reported,
		Version:    view.Version,
		UpdateTime: updateTime,
		Status:     status,
	}
}

func decodeJSONMap(raw json.RawMessage) map[string]interface{} {
	out := map[string]interface{}{}
	if len(raw) == 0 {
		return out
	}
	_ = json.Unmarshal(raw, &out)
	if out == nil {
		return map[string]interface{}{}
	}
	return out
}
