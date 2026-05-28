package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	redisshadow "github.com/jacklau/audio-ai-platform/services/device/internal/device/shadow"
	"github.com/jacklau/audio-ai-platform/services/device/internal/shadowsvc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
)

type DeviceShadowBatchReportLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	shadow *shadowsvc.Service
}

func NewDeviceShadowBatchReportLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeviceShadowBatchReportLogic {
	return &DeviceShadowBatchReportLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		shadow: shadowsvc.New(svcCtx),
	}
}

// ParseAndValidateBatchReportBody 解析请求体并校验（禁止 desired、reported 产品模型）
func ParseAndValidateBatchReportBody(raw []byte) (*types.DeviceShadowBatchReportReq, error) {
	var envelope struct {
		Items []json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "请求体不是合法 JSON")
	}
	if len(envelope.Items) == 0 {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "items 不能为空")
	}
	if len(envelope.Items) > redisshadow.MaxBatchReportSize {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam,
			fmt.Sprintf("单次最多上报 %d 台设备", redisshadow.MaxBatchReportSize))
	}

	req := &types.DeviceShadowBatchReportReq{Items: make([]types.DeviceShadowBatchReportItem, len(envelope.Items))}
	seen := make(map[string]struct{}, len(envelope.Items))

	for i, itemRaw := range envelope.Items {
		var keys map[string]json.RawMessage
		if err := json.Unmarshal(itemRaw, &keys); err != nil {
			return nil, errorx.NewCodeError(errorx.CodeInvalidParam, fmt.Sprintf("items[%d] 格式错误", i))
		}
		if _, has := keys["desired"]; has {
			return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "设备端禁止携带 desired 字段")
		}

		var item types.DeviceShadowBatchReportItem
		if err := json.Unmarshal(itemRaw, &item); err != nil {
			return nil, errorx.NewCodeError(errorx.CodeInvalidParam, fmt.Sprintf("items[%d] 格式错误", i))
		}
		sn := strings.ToUpper(strings.TrimSpace(item.SN))
		if sn == "" {
			return nil, errorx.NewCodeError(errorx.CodeInvalidParam, fmt.Sprintf("items[%d].sn 必填", i))
		}
		if _, dup := seen[sn]; dup {
			return nil, errorx.NewCodeError(errorx.CodeInvalidParam, fmt.Sprintf("重复 SN: %s", sn))
		}
		seen[sn] = struct{}{}
		item.SN = sn

		if err := redisshadow.ValidateDeviceReported(item.Reported); err != nil {
			return nil, errorx.NewCodeError(errorx.CodeInvalidParam, fmt.Sprintf("items[%d]: %s", i, err.Error()))
		}
		req.Items[i] = item
	}
	return req, nil
}

func (l *DeviceShadowBatchReportLogic) BatchReport(req *types.DeviceShadowBatchReportReq) (*types.DeviceShadowBatchReportResp, error) {
	if req == nil || len(req.Items) == 0 {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "items 不能为空")
	}

	inputs := make([]shadowsvc.BatchReportItemInput, len(req.Items))
	for i, it := range req.Items {
		inputs[i] = shadowsvc.BatchReportItemInput{
			SN:                  it.SN,
			ClientExpectVersion: it.ExpectVersion,
			ReportedPatch:       it.Reported,
		}
	}

	out, err := l.shadow.BatchReportShadows(l.ctx, inputs)
	if err != nil {
		return nil, err
	}

	resp := &types.DeviceShadowBatchReportResp{
		SuccessCount: out.SuccessCount,
		FailCount:    out.FailCount,
	}
	for _, item := range out.Items {
		resp.Items = append(resp.Items, types.DeviceShadowBatchReportItemResult{
			SN:         item.SN,
			NewVersion: item.NewVersion,
		})
		NotifyDeviceStatusChange(item.SN, BuildDeviceStatusChangeMessage(item.SN, map[string]interface{}{
			"sn":          item.SN,
			"new_version": item.NewVersion,
		}, item.NewVersion))
	}
	return resp, nil
}
