package shadow

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const MaxBatchShadowSize = 100

// BatchCASItem 单设备批量更新项（调用方已完成 merge，传入全量 JSON）
type BatchCASItem struct {
	SN            string
	ExpectVersion int64
	ReportedJSON  string
	DesiredJSON   string
	DeltaJSON     string
	MetadataJSON  string
}

// BatchCASResult 批量 CAS 结果
type BatchCASResult struct {
	Updated int
}

// BatchCASError 批量失败（整批未写入 Redis）
type BatchCASError struct {
	Code            string
	Index           int // 1-based，与 Lua 返回一致
	Key             string
	CurrentVersion  int64
	ExpectVersion   int64
	Message         string
}

func (e *BatchCASError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Code
}

// RunBatchCAS 原子批量写入影子 Hash；version 不匹配或 key 不存在时整批失败
func RunBatchCAS(ctx context.Context, rdb *redis.Client, ttl time.Duration, items []BatchCASItem) (*BatchCASResult, error) {
	if rdb == nil {
		return nil, fmt.Errorf("redis client nil")
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("items empty")
	}
	if len(items) > MaxBatchShadowSize {
		return nil, fmt.Errorf("batch size exceeds %d", MaxBatchShadowSize)
	}

	keys := make([]string, len(items))
	args := make([]interface{}, 0, len(items)*6)
	nowMs := strconv.FormatInt(time.Now().UnixMilli(), 10)

	for i, it := range items {
		sn := strings.ToUpper(strings.TrimSpace(it.SN))
		if sn == "" {
			return nil, fmt.Errorf("item[%d]: sn empty", i)
		}
		keys[i] = ShadowKey(sn)
		rj := strings.TrimSpace(it.ReportedJSON)
		if rj == "" {
			rj = "{}"
		}
		dj := strings.TrimSpace(it.DesiredJSON)
		if dj == "" {
			dj = "{}"
		}
		dlt := strings.TrimSpace(it.DeltaJSON)
		if dlt == "" {
			dlt = "{}"
		}
		meta := strings.TrimSpace(it.MetadataJSON)
		if meta == "" {
			meta = "{}"
		}
		args = append(args,
			strconv.FormatInt(it.ExpectVersion, 10),
			rj, dj, dlt, meta, nowMs,
		)
	}

	script := redis.NewScript(LuaScriptBatchCAS)
	raw, err := script.Run(ctx, rdb, keys, args...).Result()
	if err != nil {
		return nil, fmt.Errorf("batch cas redis: %w", err)
	}

	if batchErr := parseBatchLuaError(raw); batchErr != nil {
		return nil, batchErr
	}

	if ttl > 0 {
		pipe := rdb.Pipeline()
		for _, k := range keys {
			pipe.Expire(ctx, k, ttl)
		}
		_, _ = pipe.Exec(ctx)
	}

	return &BatchCASResult{Updated: len(items)}, nil
}

func parseBatchLuaError(raw interface{}) error {
	m, ok := raw.([]interface{})
	if !ok || len(m) == 0 {
		return nil
	}
	mp := luaSliceToMap(m)
	if errCode, _ := mp["err"].(string); errCode == "" {
		return nil
	}
	be := &BatchCASError{
		Code:    fmt.Sprint(mp["err"]),
		Message: fmt.Sprint(mp["err"]),
	}
	if idx, ok := mp["index"].(int64); ok {
		be.Index = int(idx)
	}
	if k, ok := mp["key"].(string); ok {
		be.Key = k
	}
	if cv, ok := mp["current_version"].(int64); ok {
		be.CurrentVersion = cv
	}
	if ev, ok := mp["expect_version"].(int64); ok {
		be.ExpectVersion = ev
	}
	switch be.Code {
	case "version_conflict":
		be.Message = fmt.Sprintf("设备影子版本冲突(index=%d): 当前=%d 期望=%d", be.Index, be.CurrentVersion, be.ExpectVersion)
	case "shadow_not_found":
		be.Message = fmt.Sprintf("设备影子不存在(index=%d)", be.Index)
	default:
		be.Message = be.Code
	}
	return be
}

func luaSliceToMap(slice []interface{}) map[string]interface{} {
	out := make(map[string]interface{})
	for i := 0; i+1 < len(slice); i += 2 {
		k, _ := slice[i].(string)
		if k == "" {
			continue
		}
		out[k] = slice[i+1]
	}
	return out
}

// SnapshotFromHash 从 HGETALL 结果解析版本与 JSON 字段（用于批量预读）
func SnapshotFromHash(sn string, h map[string]string) (expectVersion int64, reported, desired, metadata map[string]interface{}, err error) {
	expectVersion = atoi64(h[FVersion])
	reported = map[string]interface{}{}
	desired = map[string]interface{}{}
	metadata = map[string]interface{}{}
	if rj := strings.TrimSpace(h[FReportedJSON]); rj != "" {
		_ = json.Unmarshal([]byte(rj), &reported)
	}
	if dj := strings.TrimSpace(h[FDesiredJSON]); dj != "" {
		_ = json.Unmarshal([]byte(dj), &desired)
	}
	if mj := strings.TrimSpace(h[FMetadataJSON]); mj != "" {
		_ = json.Unmarshal([]byte(mj), &metadata)
	}
	if reported == nil {
		reported = map[string]interface{}{}
	}
	if desired == nil {
		desired = map[string]interface{}{}
	}
	if metadata == nil {
		metadata = map[string]interface{}{}
	}
	return expectVersion, reported, desired, metadata, nil
}
