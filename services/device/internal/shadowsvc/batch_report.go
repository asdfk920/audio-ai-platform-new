package shadowsvc

import (
	"context"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	redisshadow "github.com/jacklau/audio-ai-platform/services/device/internal/device/shadow"
)

// BatchReportItemInput 设备端批量上报单项（仅 reported）
type BatchReportItemInput struct {
	SN                  string
	ClientExpectVersion *int64 // nil 表示不校验客户端版本，直接以服务端当前版本 CAS
	ReportedPatch       map[string]interface{}
}

// BatchReportItemOutput 批量上报成功单项
type BatchReportItemOutput struct {
	SN         string `json:"sn"`
	NewVersion int64  `json:"new_version"`
}

// BatchReportOutput 批量上报输出
type BatchReportOutput struct {
	SuccessCount int                     `json:"success_count"`
	FailCount    int                     `json:"fail_count"`
	Items        []BatchReportItemOutput `json:"items"`
}

// BatchReportShadows 设备端批量上报 reported（不写 desired；原子 CAS + DB 事务）
func (s *Service) BatchReportShadows(ctx context.Context, items []BatchReportItemInput) (*BatchReportOutput, error) {
	if s.svcCtx == nil || s.svcCtx.Redis == nil {
		return nil, errorx.NewCodeError(errorx.CodeInternalError, "Redis 未配置")
	}
	if s.svcCtx.DB == nil {
		return nil, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	if len(items) == 0 {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "items 不能为空")
	}
	if len(items) > redisshadow.MaxBatchReportSize {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam,
			fmt.Sprintf("单次最多上报 %d 台设备", redisshadow.MaxBatchReportSize))
	}

	inputs := make([]BatchUpdateItemInput, len(items))
	for i, it := range items {
		inputs[i] = BatchUpdateItemInput{
			SN:                  it.SN,
			ClientExpectVersion: it.ClientExpectVersion,
			ReportedPatch:       it.ReportedPatch,
		}
	}

	out, err := s.batchUpdateReportedOnly(ctx, inputs)
	if err != nil {
		return nil, err
	}

	resp := &BatchReportOutput{
		SuccessCount: len(out.Items),
		FailCount:    0,
	}
	for _, item := range out.Items {
		resp.Items = append(resp.Items, BatchReportItemOutput{
			SN:         item.SN,
			NewVersion: item.Version,
		})
	}
	return resp, nil
}

// batchUpdateReportedOnly 内部：仅合并 reported，desired 保持库中原值
func (s *Service) batchUpdateReportedOnly(ctx context.Context, items []BatchUpdateItemInput) (*BatchUpdateOutput, error) {
	prepared, err := s.prepareBatchReport(ctx, items)
	if err != nil {
		return nil, err
	}

	casItems, err := s.buildCASItems(prepared)
	if err != nil {
		return nil, err
	}

	ttl := batchShadowTTL(s.svcCtx)
	if _, err := redisshadow.RunBatchCAS(ctx, s.svcCtx.Redis, ttl, casItems); err != nil {
		if be, ok := err.(*redisshadow.BatchCASError); ok {
			return nil, errorx.NewCodeError(errorx.CodeDeviceShadowVersionConflict, be.Error())
		}
		return nil, errorx.NewDefaultError(errorx.CodeRedisError)
	}

	if err := s.persistBatchInTx(ctx, prepared); err != nil {
		s.rollbackRedisBatch(ctx, prepared)
		return nil, err
	}

	out := &BatchUpdateOutput{Updated: len(prepared)}
	for _, p := range prepared {
		out.Items = append(out.Items, BatchUpdateItemOutput{SN: p.SN, Version: p.NewVersion})
	}
	return out, nil
}

func (s *Service) prepareBatchReport(ctx context.Context, items []BatchUpdateItemInput) ([]batchPrepared, error) {
	seen := make(map[string]struct{}, len(items))
	pipe := s.svcCtx.Redis.Pipeline()
	cmds := make([]*redis.MapStringStringCmd, len(items))
	normItems := make([]BatchUpdateItemInput, len(items))

	for i, it := range items {
		sn := strings.ToUpper(strings.TrimSpace(it.SN))
		if sn == "" {
			return nil, errorx.NewCodeError(errorx.CodeInvalidParam, fmt.Sprintf("items[%d].sn 必填", i))
		}
		if _, dup := seen[sn]; dup {
			return nil, errorx.NewCodeError(errorx.CodeInvalidParam, fmt.Sprintf("重复 SN: %s", sn))
		}
		seen[sn] = struct{}{}
		normItems[i] = it
		normItems[i].SN = sn
		cmds[i] = pipe.HGetAll(ctx, redisshadow.ShadowKey(sn))
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, errorx.NewDefaultError(errorx.CodeRedisError)
	}

	prepared := make([]batchPrepared, 0, len(items))

	for i, it := range normItems {
		h, err := cmds[i].Result()
		if err != nil {
			return nil, errorx.NewDefaultError(errorx.CodeRedisError)
		}
		if len(h) == 0 {
			return nil, errorx.NewCodeError(errorx.CodeNotFound, fmt.Sprintf("设备 %s 影子不存在，请先注册/初始化", it.SN))
		}

		curVer, reported, desired, metadata, _ := redisshadow.SnapshotFromHash(it.SN, h)
		if it.ClientExpectVersion != nil && *it.ClientExpectVersion != curVer {
			return nil, errorx.NewCodeError(errorx.CodeDeviceShadowVersionConflict,
				fmt.Sprintf("设备 %s 版本冲突: 当前=%d 提交=%d", it.SN, curVer, *it.ClientExpectVersion))
		}

		rollback := make(map[string]string, len(h))
		for k, v := range h {
			rollback[k] = v
		}

		if len(it.ReportedPatch) > 0 {
			reported = mergeMaps(reported, it.ReportedPatch)
		}

		dev, err := s.svcCtx.DeviceRepo.FindBySn(ctx, it.SN)
		if err != nil || dev == nil {
			return nil, errorx.NewCodeError(errorx.CodeNotFound, fmt.Sprintf("设备 %s 不存在", it.SN))
		}

		prepared = append(prepared, batchPrepared{
			DeviceID:     dev.ID,
			SN:           it.SN,
			ExpectVer:    curVer,
			NewVersion:   curVer + 1,
			Reported:     reported,
			Desired:      desired,
			Metadata:     metadata,
			Delta:        ComputeJSONDelta(desired, reported),
			RollbackHash: rollback,
		})
	}
	return prepared, nil
}
