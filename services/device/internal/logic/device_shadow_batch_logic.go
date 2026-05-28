package logic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/services/device/internal/shadowsvc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
)

type DeviceShadowBatchLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	shadow *shadowsvc.Service
}

func NewDeviceShadowBatchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeviceShadowBatchLogic {
	return &DeviceShadowBatchLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		shadow: shadowsvc.New(svcCtx),
	}
}

func (l *DeviceShadowBatchLogic) BatchUpdate(req *types.DeviceShadowBatchReq) (*types.DeviceShadowBatchResp, error) {
	if req == nil {
		req = &types.DeviceShadowBatchReq{}
	}

	inputs := make([]shadowsvc.BatchUpdateItemInput, len(req.Items))
	for i, it := range req.Items {
		v := it.ExpectVersion
		expect := &v
		inputs[i] = shadowsvc.BatchUpdateItemInput{
			SN:                  it.SN,
			ClientExpectVersion: expect,
			ReportedPatch:       it.Reported,
			DesiredPatch:        it.Desired,
			MetadataPatch:       it.Metadata,
		}
	}

	out, err := l.shadow.BatchUpdateShadows(l.ctx, inputs)
	if err != nil {
		return nil, err
	}

	resp := &types.DeviceShadowBatchResp{Updated: out.Updated}
	for _, item := range out.Items {
		resp.Items = append(resp.Items, types.DeviceShadowBatchItemResult{
			SN:      item.SN,
			Version: item.Version,
		})
		NotifyDeviceStatusChange(item.SN, BuildDeviceStatusChangeMessage(item.SN, map[string]interface{}{
			"sn":      item.SN,
			"version": item.Version,
		}, item.Version))
	}
	return resp, nil
}
