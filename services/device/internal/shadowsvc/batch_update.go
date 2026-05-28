package shadowsvc

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	redisshadow "github.com/jacklau/audio-ai-platform/services/device/internal/device/shadow"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
)

// BatchUpdateItemInput 批量更新单项
type BatchUpdateItemInput struct {
	SN                  string
	ClientExpectVersion *int64 // 非 nil 时须与 Redis 当前 version 一致（乐观锁）
	ReportedPatch       map[string]interface{}
	DesiredPatch        map[string]interface{}
	MetadataPatch       map[string]interface{}
}

// BatchUpdateItemOutput 批量更新成功后的单项结果
type BatchUpdateItemOutput struct {
	SN      string `json:"sn"`
	Version int64  `json:"version"`
}

// BatchUpdateOutput 批量更新输出
type BatchUpdateOutput struct {
	Updated int                     `json:"updated"`
	Items   []BatchUpdateItemOutput `json:"items"`
}

type batchPrepared struct {
	DeviceID     int64
	SN           string
	ExpectVer    int64
	NewVersion   int64
	Reported     map[string]interface{}
	Desired      map[string]interface{}
	Metadata     map[string]interface{}
	Delta        map[string]interface{}
	RollbackHash map[string]string
}

// BatchUpdateShadows 批量更新设备影子：Redis Lua 原子 CAS + PostgreSQL 单事务；任一步失败则整批失败并回滚 Redis
func (s *Service) BatchUpdateShadows(ctx context.Context, items []BatchUpdateItemInput) (*BatchUpdateOutput, error) {
	if s.svcCtx == nil || s.svcCtx.Redis == nil {
		return nil, errorx.NewCodeError(errorx.CodeInternalError, "Redis 未配置")
	}
	if s.svcCtx.DB == nil {
		return nil, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	if len(items) == 0 {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "items 不能为空")
	}
	if len(items) > redisshadow.MaxBatchShadowSize {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, fmt.Sprintf("单次最多 %d 台设备", redisshadow.MaxBatchShadowSize))
	}

	prepared, err := s.prepareBatch(ctx, items)
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

func (s *Service) prepareBatch(ctx context.Context, items []BatchUpdateItemInput) ([]batchPrepared, error) {
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

	nowMs := time.Now().UnixMilli()
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

		if it.ReportedPatch != nil && len(it.ReportedPatch) > 0 {
			reported = mergeMaps(reported, it.ReportedPatch)
		}
		if it.DesiredPatch != nil && len(it.DesiredPatch) > 0 {
			desired = mergeMaps(desired, it.DesiredPatch)
		}
		if it.MetadataPatch != nil && len(it.MetadataPatch) > 0 {
			metadata = mergeMetadataSection(metadata, it.MetadataPatch, nowMs)
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

func (s *Service) buildCASItems(prepared []batchPrepared) ([]redisshadow.BatchCASItem, error) {
	out := make([]redisshadow.BatchCASItem, len(prepared))
	for i, p := range prepared {
		rj, _ := json.Marshal(p.Reported)
		dj, _ := json.Marshal(p.Desired)
		dlt, _ := json.Marshal(p.Delta)
		meta, _ := json.Marshal(p.Metadata)
		out[i] = redisshadow.BatchCASItem{
			SN:            p.SN,
			ExpectVersion: p.ExpectVer,
			ReportedJSON:  string(rj),
			DesiredJSON:   string(dj),
			DeltaJSON:     string(dlt),
			MetadataJSON:  string(meta),
		}
	}
	return out, nil
}

func (s *Service) persistBatchInTx(ctx context.Context, prepared []batchPrepared) error {
	tx, err := s.svcCtx.DB.BeginTx(ctx, nil)
	if err != nil {
		return errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now()
	for _, p := range prepared {
		if err := s.upsertShadowInTx(ctx, tx, p.DeviceID, p.SN, p.Reported, p.Desired, p.Metadata, p.NewVersion, &now); err != nil {
			logx.Errorf("batch shadow db upsert sn=%s err=%v", p.SN, err)
			return errorx.NewDefaultError(errorx.CodeDatabaseError)
		}
	}
	if err := tx.Commit(); err != nil {
		logx.Errorf("batch shadow db commit err=%v", err)
		return errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	return nil
}

func (s *Service) upsertShadowInTx(ctx context.Context, tx *sql.Tx, deviceID int64, sn string, reported, desired, metadata map[string]interface{}, version int64, lastReportTime *time.Time) error {
	reportedBytes, _ := json.Marshal(reported)
	desiredBytes, _ := json.Marshal(desired)
	metadataBytes, _ := json.Marshal(metadata)
	var reportArg interface{}
	if lastReportTime != nil {
		reportArg = *lastReportTime
	}
	_, err := tx.ExecContext(ctx, `
INSERT INTO public.device_shadow (device_id, sn, reported, desired, metadata, version, last_report_time)
VALUES ($1, $2, $3::jsonb, $4::jsonb, $5::jsonb, $6, $7)
ON CONFLICT (device_id) DO UPDATE
SET sn = EXCLUDED.sn,
    reported = EXCLUDED.reported,
    desired = EXCLUDED.desired,
    metadata = EXCLUDED.metadata,
    version = EXCLUDED.version,
    last_report_time = COALESCE(EXCLUDED.last_report_time, public.device_shadow.last_report_time),
    updated_at = CURRENT_TIMESTAMP
`, deviceID, strings.ToUpper(strings.TrimSpace(sn)), string(reportedBytes), string(desiredBytes), string(metadataBytes), version, reportArg)
	return err
}

func (s *Service) rollbackRedisBatch(ctx context.Context, prepared []batchPrepared) {
	pipe := s.svcCtx.Redis.Pipeline()
	for _, p := range prepared {
		if len(p.RollbackHash) == 0 {
			continue
		}
		fields := make(map[string]interface{}, len(p.RollbackHash))
		for k, v := range p.RollbackHash {
			fields[k] = v
		}
		pipe.HSet(ctx, redisshadow.ShadowKey(p.SN), fields)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		logx.Errorf("batch shadow redis rollback failed: %v", err)
	}
}

func batchShadowTTL(svcCtx *svc.ServiceContext) time.Duration {
	if svcCtx == nil {
		return 24 * time.Hour
	}
	sec := svcCtx.Config.DeviceShadow.SeedTTLSeconds
	if sec <= 0 {
		sec = svcCtx.Config.DeviceShadow.HeartbeatTTLSeconds
	}
	if sec <= 0 {
		sec = 86400
	}
	return time.Duration(sec) * time.Second
}
