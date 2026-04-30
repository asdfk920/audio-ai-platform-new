package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go-admin/app/admin/device/shadow"
	"go-admin/common/mqttadmin"
	"gorm.io/gorm"
)

// DeviceShadowView 设备影子（管理端聚合 Redis + PG）
type DeviceShadowView struct {
	DeviceID        int64           `json:"device_id"`
	Sn              string          `json:"sn"`
	Online          bool            `json:"online"`
	LastOnlineTime  *time.Time      `json:"last_online_time,omitempty"`
	FirmwareVersion string          `json:"firmware_version,omitempty"`
	Battery         *int32          `json:"battery,omitempty"`
	PowerStatus     string          `json:"power_status,omitempty"`
	Volume          *int32          `json:"volume,omitempty"`
	Calibration     json.RawMessage `json:"calibration,omitempty"`
	Network         json.RawMessage `json:"network,omitempty"`
	Reported        json.RawMessage `json:"reported"`
	Desired         json.RawMessage `json:"desired"`
	Delta           json.RawMessage `json:"delta"`
	RedisPresent    bool            `json:"redis_present"`
	LastReportTime  *time.Time      `json:"last_report_time,omitempty"`
}

type deviceShadowRow struct {
	DeviceID       int64           `gorm:"column:device_id"`
	Sn             string          `gorm:"column:sn"`
	Reported       json.RawMessage `gorm:"column:reported;type:jsonb"`
	Desired        json.RawMessage `gorm:"column:desired;type:jsonb"`
	LastReportTime *time.Time      `gorm:"column:last_report_time"`
}

// GetDeviceShadow 聚合 PG device_shadow + Redis device:shadow:{SN}
func (e *PlatformDeviceService) GetDeviceShadow(sn string) (*DeviceShadowView, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	sn = strings.TrimSpace(sn)
	if sn == "" {
		return nil, ErrPlatformDeviceInvalid
	}
	snNorm := strings.ToUpper(sn)

	var dev struct {
		ID              int64  `gorm:"column:id"`
		Sn              string `gorm:"column:sn"`
		FirmwareVersion string `gorm:"column:firmware_version"`
		OnlineStatus    int16  `gorm:"column:online_status"`
	}
	if err := e.Orm.Table("device").Select("id, sn, firmware_version, online_status").Where("sn = ? AND deleted_at IS NULL", sn).Take(&dev).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPlatformDeviceNotFound
		}
		return nil, err
	}

	var row deviceShadowRow
	_ = e.Orm.Table("device_shadow").Where("device_id = ?", dev.ID).Take(&row).Error

	out := &DeviceShadowView{
		DeviceID:        dev.ID,
		Sn:              dev.Sn,
		FirmwareVersion: strings.TrimSpace(dev.FirmwareVersion),
		LastReportTime:  row.LastReportTime,
		Online:          dev.OnlineStatus == 1,
	}

	reportedMap := map[string]interface{}{}
	if len(row.Reported) > 0 {
		_ = json.Unmarshal(row.Reported, &reportedMap)
	}

	rdb := shadow.Client()
	if rdb != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		sk := shadow.ShadowKey(snNorm)
		h, err := rdb.HGetAll(ctx, sk).Result()
		if err == nil && len(h) > 0 {
			out.RedisPresent = true
			mergeRedisIntoReported(h, reportedMap)
			onlineVal, _ := rdb.Get(ctx, shadow.OnlineKey(snNorm)).Result()
			out.Online = strings.TrimSpace(h[shadow.FOnline]) == "1" || onlineVal == "1"
			if ms := firstNonZero(msParse(h[shadow.FUpdatedMs]), msParse(h[shadow.FLastActiveMs])); ms > 0 {
				t := time.UnixMilli(ms)
				out.LastOnlineTime = &t
			}
			if rj := strings.TrimSpace(h[shadow.FReportedJSON]); rj != "" {
				var extra map[string]interface{}
				if json.Unmarshal([]byte(rj), &extra) == nil {
					for k, v := range extra {
						reportedMap[k] = v
					}
				}
			}
		}
	}

	applyScalarShortcuts(reportedMap, out)

	b, _ := json.Marshal(reportedMap)
	out.Reported = b

	desiredMap := map[string]interface{}{}
	if len(row.Desired) > 0 {
		_ = json.Unmarshal(row.Desired, &desiredMap)
	}
	db, _ := json.Marshal(desiredMap)
	out.Desired = db

	deltaMap := computeJSONDelta(desiredMap, reportedMap)
	dj, _ := json.Marshal(deltaMap)
	out.Delta = dj

	return out, nil
}

func mergeRedisIntoReported(h map[string]string, reported map[string]interface{}) {
	setInt := func(key, field string) {
		if v := strings.TrimSpace(h[field]); v != "" {
			if n, err := strconv.ParseInt(v, 10, 32); err == nil {
				reported[key] = int32(n)
			}
		}
	}
	if v := strings.TrimSpace(h[shadow.FFirmwareVersion]); v != "" {
		reported["firmware_version"] = v
	}
	setInt("battery", shadow.FBattery)
	setInt("volume", shadow.FVolume)
	if v := strings.TrimSpace(h[shadow.FRunState]); v != "" {
		reported["run_state"] = v
		reported["power_status"] = v
	}
	if v := strings.TrimSpace(h[shadow.FIP]); v != "" {
		net := map[string]interface{}{"ip": v}
		b, _ := json.Marshal(net)
		reported["network"] = json.RawMessage(b)
	}
	if v := strings.TrimSpace(h[shadow.FMac]); v != "" {
		reported["mac"] = v
	}
}

func applyScalarShortcuts(m map[string]interface{}, out *DeviceShadowView) {
	if v, ok := m["firmware_version"].(string); ok && v != "" {
		out.FirmwareVersion = v
	}
	if v, ok := m["battery"].(int32); ok {
		b := v
		out.Battery = &b
	}
	if v, ok := m["battery"].(float64); ok {
		b := int32(v)
		out.Battery = &b
	}
	if v, ok := m["volume"].(int32); ok {
		b := v
		out.Volume = &b
	}
	if v, ok := m["volume"].(float64); ok {
		b := int32(v)
		out.Volume = &b
	}
	if v, ok := m["power_status"].(string); ok {
		out.PowerStatus = v
	} else if v, ok := m["run_state"].(string); ok {
		out.PowerStatus = v
	}
	if raw, ok := m["calibration"].(json.RawMessage); ok && len(raw) > 0 {
		out.Calibration = raw
	}
	if raw, ok := m["network"].(json.RawMessage); ok && len(raw) > 0 {
		out.Network = raw
	} else if sub, ok := m["network"].(map[string]interface{}); ok {
		b, _ := json.Marshal(sub)
		out.Network = b
	}
}

func msParse(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

func firstNonZero(a, b int64) int64 {
	if a != 0 {
		return a
	}
	return b
}

func computeJSONDelta(desired, reported map[string]interface{}) map[string]interface{} {
	if len(desired) == 0 {
		return map[string]interface{}{}
	}
	delta := map[string]interface{}{}
	for k, dv := range desired {
		rv, ok := reported[k]
		if !ok {
			delta[k] = dv
			continue
		}
		db, _ := json.Marshal(dv)
		rb, _ := json.Marshal(rv)
		if !bytes.Equal(db, rb) {
			delta[k] = dv
		}
	}
	return delta
}

// PutDeviceShadowDesiredIn 更新期望状态
type PutDeviceShadowDesiredIn struct {
	Sn       string
	Desired  json.RawMessage
	Merge    bool // true 时与库中 desired 合并
	Operator string
}

// PutDeviceShadowDesiredOut 更新结果
type PutDeviceShadowDesiredOut struct {
	DeviceID   int64           `json:"device_id"`
	Sn         string          `json:"sn"`
	Desired    json.RawMessage `json:"desired"`
	Delta      json.RawMessage `json:"delta"`
	PushedMQTT bool            `json:"pushed_mqtt"`
	Online     bool            `json:"online"`
}

// PutDeviceShadowDesired 写入 desired、计算 delta、写 Redis、在线则 MQTT 下发 delta
func (e *PlatformDeviceService) PutDeviceShadowDesired(in *PutDeviceShadowDesiredIn) (*PutDeviceShadowDesiredOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	if in == nil || len(in.Desired) == 0 {
		return nil, ErrPlatformDeviceInvalid
	}
	sn := strings.TrimSpace(in.Sn)
	if sn == "" {
		return nil, ErrPlatformDeviceInvalid
	}
	var newDesired map[string]interface{}
	if err := json.Unmarshal(in.Desired, &newDesired); err != nil {
		return nil, fmt.Errorf("desired 不是合法 JSON: %w", err)
	}

	var dev struct {
		ID     int64  `gorm:"column:id"`
		Sn     string `gorm:"column:sn"`
		Status int16  `gorm:"column:status"`
	}
	if err := e.Orm.Table("device").Select("id, sn, status").Where("sn = ? AND deleted_at IS NULL", sn).Take(&dev).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPlatformDeviceNotFound
		}
		return nil, err
	}
	if dev.Status != 1 {
		return nil, fmt.Errorf("设备未处于可用状态，无法写入影子期望")
	}

	var row deviceShadowRow
	_ = e.Orm.Table("device_shadow").Where("device_id = ?", dev.ID).Take(&row).Error
	prevDesired := map[string]interface{}{}
	if len(row.Desired) > 0 {
		_ = json.Unmarshal(row.Desired, &prevDesired)
	}
	if in.Merge {
		for k, v := range newDesired {
			prevDesired[k] = v
		}
	} else {
		prevDesired = newDesired
	}
	desiredBytes, err := json.Marshal(prevDesired)
	if err != nil {
		return nil, err
	}

	// 以当前 reported 为基准算 delta（与 Get 一致：先取 PG 再叠 Redis）
	view, _ := e.GetDeviceShadow(sn)
	reportedMap := map[string]interface{}{}
	if view != nil && len(view.Reported) > 0 {
		_ = json.Unmarshal(view.Reported, &reportedMap)
	}
	deltaMap := computeJSONDelta(prevDesired, reportedMap)
	deltaBytes, _ := json.Marshal(deltaMap)

	now := time.Now()
	if err := e.upsertDeviceShadowDesired(dev.ID, dev.Sn, desiredBytes, now); err != nil {
		return nil, err
	}
	op := strings.TrimSpace(in.Operator)
	if op == "" {
		op = "admin"
	}
	_ = e.Orm.Exec(`INSERT INTO device_event_log (device_id, sn, event_type, content, operator) VALUES (?,?,?,?,?)`,
		dev.ID, dev.Sn, "admin_shadow_desired", truncateEvent(fmt.Sprintf("update desired merge=%v", in.Merge)), op).Error

	online := false
	rdb := shadow.Client()
	if rdb != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		sk := shadow.ShadowKey(dev.Sn)
		onlineVal, _ := rdb.Get(ctx, shadow.OnlineKey(strings.ToUpper(dev.Sn))).Result()
		online = onlineVal == "1"
		pipe := rdb.Pipeline()
		pipe.HSet(ctx, sk, shadow.FDesiredJSON, string(desiredBytes))
		pipe.HSet(ctx, sk, shadow.FDeltaJSON, string(deltaBytes))
		pipe.Expire(ctx, sk, shadow.DefaultShadowTTL())
		_, _ = pipe.Exec(ctx)
	}

	pushed := false
	if online && len(deltaMap) > 0 {
		if cli := mqttadmin.Client(); cli != nil {
			payload := map[string]interface{}{
				"type":       "device_shadow_delta",
				"device_sn":  dev.Sn,
				"device_id":  dev.ID,
				"delta":      deltaMap,
				"desired":    prevDesired,
				"timestamp":  now.UnixMilli(),
				"source":     "admin",
				"operator":   strings.TrimSpace(in.Operator),
			}
			b, _ := json.Marshal(payload)
			topic := fmt.Sprintf("device/%s/shadow/delta", dev.Sn)
			if err := cli.Publish(topic, 1, false, b); err == nil {
				pushed = true
			}
		}
	}

	return &PutDeviceShadowDesiredOut{
		DeviceID:   dev.ID,
		Sn:         dev.Sn,
		Desired:    desiredBytes,
		Delta:      deltaBytes,
		PushedMQTT: pushed,
		Online:     online,
	}, nil
}

func (e *PlatformDeviceService) upsertDeviceShadowDesired(deviceID int64, sn string, desired json.RawMessage, now time.Time) error {
	var n int64
	if err := e.Orm.Table("device_shadow").Where("device_id = ?", deviceID).Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		return e.Orm.Exec(`INSERT INTO device_shadow (device_id, sn, desired, created_at, updated_at) VALUES (?,?,?::jsonb,?,?)`,
			deviceID, sn, string(desired), now, now).Error
	}
	return e.Orm.Exec(`UPDATE device_shadow SET desired = ?::jsonb, updated_at = ? WHERE device_id = ?`,
		string(desired), now, deviceID).Error
}
