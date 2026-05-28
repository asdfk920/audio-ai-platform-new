package shadowv2

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
)

type RedisShadowStore struct {
	rdb        *redis.Client
	defaultTTL time.Duration
}

func NewRedisShadowStore(rdb *redis.Client, ttlSeconds int) *RedisShadowStore {
	ttl := 24 * time.Hour
	if ttlSeconds > 0 {
		ttl = time.Duration(ttlSeconds) * time.Second
	}

	return &RedisShadowStore{
		rdb:        rdb,
		defaultTTL: ttl,
	}
}

func (s *RedisShadowStore) InitShadow(ctx context.Context, deviceSN string) (*DeviceShadow, error) {
	key := BuildShadowKey(deviceSN)
	now := time.Now().Unix()

	reportedJSON, _ := json.Marshal(map[string]interface{}{})
	desiredJSON, _ := json.Marshal(map[string]interface{}{})

	script := redis.NewScript(LuaScriptInit)
	result, err := script.Run(ctx, s.rdb, []string{key},
		string(reportedJSON),
		string(desiredJSON),
		DefaultVersion,
		now,
		string(StatusOnline),
	).Result()

	if err != nil {
		logx.Errorf("shadowv2: InitShadow failed for device %s: %v", deviceSN, err)
		return nil, fmt.Errorf("init shadow failed: %w", err)
	}

	resMap := convertResultToMap(result)

	if errMsg, exists := resMap["err"]; exists && errMsg != nil {
		return nil, fmt.Errorf("%v", errMsg)
	}

	shadow := &DeviceShadow{
		DeviceSN:   deviceSN,
		Reported:   make(map[string]interface{}),
		Desired:    make(map[string]interface{}),
		Version:    DefaultVersion,
		UpdateTime: now,
		Status:     StatusOnline,
	}

	s.rdb.Expire(ctx, key, s.defaultTTL)

	logx.Infof("shadowv2: Shadow initialized for device %s", deviceSN)
	return shadow, nil
}

func (s *RedisShadowStore) UpdateReported(ctx context.Context, deviceSN string, reported map[string]interface{}) (*DeviceShadow, error) {
	key := BuildShadowKey(deviceSN)
	now := time.Now().Unix()

	reportedJSON, _ := json.Marshal(reported)

	script := redis.NewScript(LuaScriptUpdateReported)
	result, err := script.Run(ctx, s.rdb, []string{key},
		string(reportedJSON),
		now,
	).Result()

	if err != nil {
		logx.Errorf("shadowv2: UpdateReported failed for device %s: %v", deviceSN, err)
		return nil, fmt.Errorf("update reported failed: %w", err)
	}

	resMap := convertResultToMap(result)

	if errMsg, exists := resMap["err"]; exists && errMsg != nil {
		return nil, fmt.Errorf("shadow not found")
	}

	var finalReported map[string]interface{}
	if reportedStr, ok := resMap["reported"].(string); ok {
		json.Unmarshal([]byte(reportedStr), &finalReported)
	} else {
		finalReported = reported
	}

	version := parseInt64FromInterface(resMap["version"])
	statusStr, _ := resMap["status"].(string)
	status := DeviceStatus(statusStr)
	if status == "" {
		status = StatusOffline
	}

	shadow := &DeviceShadow{
		DeviceSN:   deviceSN,
		Reported:   finalReported,
		Version:    version,
		UpdateTime: now,
		Status:     status,
	}

	s.rdb.Expire(ctx, key, s.defaultTTL)

	logx.Infof("shadowv2: Reported updated for device %s, version=%d", deviceSN, version)
	return shadow, nil
}

func (s *RedisShadowStore) UpdateDesired(ctx context.Context, deviceSN string, desired map[string]interface{}) (*DeviceShadow, error) {
	key := BuildShadowKey(deviceSN)
	now := time.Now().Unix()

	desiredJSON, _ := json.Marshal(desired)

	script := redis.NewScript(LuaScriptUpdateDesired)
	result, err := script.Run(ctx, s.rdb, []string{key},
		string(desiredJSON),
		now,
	).Result()

	if err != nil {
		logx.Errorf("shadowv2: UpdateDesired failed for device %s: %v", deviceSN, err)
		return nil, fmt.Errorf("update desired failed: %w", err)
	}

	resMap := convertResultToMap(result)

	if errMsg, exists := resMap["err"]; exists && errMsg != nil {
		return nil, fmt.Errorf("shadow not found")
	}

	var finalDesired map[string]interface{}
	if desiredStr, ok := resMap["desired"].(string); ok {
		json.Unmarshal([]byte(desiredStr), &finalDesired)
	} else {
		finalDesired = desired
	}

	version := parseInt64FromInterface(resMap["version"])
	statusStr, _ := resMap["status"].(string)
	status := DeviceStatus(statusStr)
	if status == "" {
		status = StatusOffline
	}

	shadow := &DeviceShadow{
		DeviceSN:   deviceSN,
		Desired:    finalDesired,
		Version:    version,
		UpdateTime: now,
		Status:     status,
	}

	s.rdb.Expire(ctx, key, s.defaultTTL)

	logx.Infof("shadowv2: Desired updated for device %s, version=%d", deviceSN, version)
	return shadow, nil
}

func (s *RedisShadowStore) GetShadow(ctx context.Context, deviceSN string) (*DeviceShadow, error) {
	deviceSN = strings.TrimSpace(deviceSN)
	if deviceSN == "" {
		return nil, fmt.Errorf("shadow not found")
	}
	// v1 影子键使用大写 SN；订阅/查询可能传入原始大小写，两种都尝试
	candidates := []string{deviceSN}
	upper := strings.ToUpper(deviceSN)
	if upper != deviceSN {
		candidates = append(candidates, upper)
	}
	var lastErr error
	for _, sn := range candidates {
		shadow, err := s.getShadowBySN(ctx, sn)
		if err == nil {
			return shadow, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("shadow not found")
}

func (s *RedisShadowStore) getShadowBySN(ctx context.Context, deviceSN string) (*DeviceShadow, error) {
	key := BuildShadowKey(deviceSN)
	data, err := s.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		logx.Errorf("shadowv2: GetShadow failed for device %s: %v", deviceSN, err)
		return nil, fmt.Errorf("query shadow failed: %w", err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("shadow not found")
	}
	return FromMap(deviceSN, data), nil
}

func (s *RedisShadowStore) GetReported(ctx context.Context, deviceSN string) (map[string]interface{}, int64, error) {
	shadow, err := s.GetShadow(ctx, deviceSN)
	if err != nil {
		return nil, 0, err
	}

	return shadow.Reported, shadow.Version, nil
}

func (s *RedisShadowStore) GetVersion(ctx context.Context, deviceSN string) (int64, error) {
	key := BuildShadowKey(deviceSN)
	fields := GetShadowFields()

	versionStr, err := s.rdb.HGet(ctx, key, fields.Version).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, fmt.Errorf("shadow not found")
		}
		return 0, err
	}

	version := parseInt64(versionStr)
	return version, nil
}

func (s *RedisShadowStore) UpdateStatus(ctx context.Context, deviceSN string, status DeviceStatus) (*DeviceShadow, error) {
	key := BuildShadowKey(deviceSN)
	now := time.Now().Unix()

	script := redis.NewScript(LuaScriptUpdateStatus)
	result, err := script.Run(ctx, s.rdb, []string{key},
		string(status),
		now,
	).Result()

	if err != nil {
		logx.Errorf("shadowv2: UpdateStatus failed for device %s: %v", deviceSN, err)
		return nil, fmt.Errorf("update status failed: %w", err)
	}

	resMap := convertResultToMap(result)

	if errMsg, exists := resMap["err"]; exists && errMsg != nil {
		return nil, fmt.Errorf("shadow not found")
	}

	version := parseInt64FromInterface(resMap["version"])

	shadow := &DeviceShadow{
		DeviceSN:   deviceSN,
		Version:    version,
		UpdateTime: now,
		Status:     status,
	}

	s.rdb.Expire(ctx, key, s.defaultTTL)

	logx.Infof("shadowv2: Status updated for device %s to %s, version=%d", deviceSN, status, version)
	return shadow, nil
}

func (s *RedisShadowStore) CASUpdate(ctx context.Context, req *CASUpdateReq) (*CASUpdateResp, error) {
	key := BuildShadowKey(req.DeviceSN)
	now := time.Now().Unix()

	reportedJSON := "{}"
	if req.Reported != nil && len(req.Reported) > 0 {
		tmp, _ := json.Marshal(req.Reported)
		reportedJSON = string(tmp)
	}

	desiredJSON := "{}"
	if req.Desired != nil && len(req.Desired) > 0 {
		tmp, _ := json.Marshal(req.Desired)
		desiredJSON = string(tmp)
	}

	statusStr := ""
	if req.Status != nil {
		statusStr = string(*req.Status)
	}

	script := redis.NewScript(LuaScriptCASUpdate)
	result, err := script.Run(ctx, s.rdb, []string{key},
		req.ExpectVersion,
		reportedJSON,
		desiredJSON,
		statusStr,
		now,
	).Result()

	if err != nil {
		logx.Errorf("shadowv2: CASUpdate failed for device %s: %v", req.DeviceSN, err)
		return nil, fmt.Errorf("CAS update failed: %w", err)
	}

	resMap := convertResultToMap(result)
	resp := &CASUpdateResp{}

	if conflict, ok := resMap["conflict"].(bool); ok && conflict {
		currentVer := parseInt64FromInterface(resMap["current_version"])
		resp.Success = false
		resp.Message = "version conflict, please retry"
		resp.Error = fmt.Sprintf("current_version: %d, expect_version: %d", currentVer, req.ExpectVersion)
		return resp, nil
	}

	if errMsg, ok := resMap["err"].(string); ok && errMsg != "" {
		resp.Success = false
		resp.Message = errMsg
		resp.Error = errMsg
		return resp, nil
	}

	version := parseInt64FromInterface(resMap["version"])
	status := StatusOffline
	if statusVal, ok := resMap["status"].(string); ok && statusVal != "" {
		status = DeviceStatus(statusVal)
	}

	var reported, desired map[string]interface{}
	if reportedStr, ok := resMap["reported"].(string); ok {
		json.Unmarshal([]byte(reportedStr), &reported)
	}
	if desiredStr, ok := resMap["desired"].(string); ok {
		json.Unmarshal([]byte(desiredStr), &desired)
	}

	shadow, getErr := s.GetShadow(ctx, req.DeviceSN)
	if getErr != nil {
		shadow = &DeviceShadow{
			DeviceSN:   req.DeviceSN,
			Reported:   reported,
			Desired:    desired,
			Version:    version,
			UpdateTime: now,
			Status:     status,
		}
	} else {
		if reported != nil {
			shadow.Reported = reported
		}
		if desired != nil {
			shadow.Desired = desired
		}
		shadow.Version = version
		shadow.Status = status
	}

	resp.Success = true
	resp.Message = "CAS update success"
	resp.Shadow = shadow

	s.rdb.Expire(ctx, key, s.defaultTTL)

	logx.Infof("shadowv2: CAS update success for device %s, new version=%d", req.DeviceSN, version)
	return resp, nil
}

func (s *RedisShadowStore) DeleteShadow(ctx context.Context, deviceSN string) error {
	key := BuildShadowKey(deviceSN)
	err := s.rdb.Del(ctx, key).Err()
	if err != nil {
		logx.Errorf("shadowv2: DeleteShadow failed for device %s: %v", deviceSN, err)
		return fmt.Errorf("delete shadow failed: %w", err)
	}

	onlineKey := "device:online:" + deviceSN
	s.rdb.Del(ctx, onlineKey)

	logx.Infof("shadowv2: Shadow deleted for device %s", deviceSN)
	return nil
}

func (s *RedisShadowStore) ShadowExists(ctx context.Context, deviceSN string) (bool, error) {
	key := BuildShadowKey(deviceSN)
	exists, err := s.rdb.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

func convertResultToMap(result interface{}) map[string]interface{} {
	resMap := make(map[string]interface{})

	switch v := result.(type) {
	case []interface{}:
		for i := 0; i < len(v); i += 2 {
			if i+1 < len(v) {
				key, ok1 := v[i].(string)
				val := v[i+1]
				if ok1 {
					resMap[key] = val
				}
			}
		}
	case map[string]interface{}:
		resMap = v
	}

	return resMap
}

func parseInt64FromInterface(v interface{}) int64 {
	switch val := v.(type) {
	case int64:
		return val
	case float64:
		return int64(val)
	case string:
		return parseInt64(val)
	case json.Number:
		n, _ := val.Int64()
		return n
	case nil:
		return 0
	default:
		return 0
	}
}

// CountByStatus 统计指定状态的设备数量
func (s *RedisShadowStore) CountByStatus(ctx context.Context, status DeviceStatus) (int64, error) {
	pattern := BuildShadowKey("*")
	var cursor uint64
	var count int64

	for {
		keys, nextCursor, err := s.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return 0, fmt.Errorf("scan shadows failed: %w", err)
		}

		cursor = nextCursor
		if len(keys) > 0 {
			for _, key := range keys {
				statusStr, err := s.rdb.HGet(ctx, key, GetShadowFields().Status).Result()
				if err == nil && DeviceStatus(statusStr) == status {
					count++
				}
			}
		}

		if cursor == 0 {
			break
		}
	}

	return count, nil
}

// CountTotal 统计设备影子总数
func (s *RedisShadowStore) CountTotal(ctx context.Context) (int64, error) {
	pattern := BuildShadowKey("*")
	var cursor uint64
	var count int64

	for {
		keys, nextCursor, err := s.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return 0, fmt.Errorf("scan shadows failed: %w", err)
		}

		count += int64(len(keys))
		cursor = nextCursor

		if cursor == 0 {
			break
		}
	}

	return count, nil
}
