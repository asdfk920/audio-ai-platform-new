package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
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
				"type":      "device_shadow_delta",
				"device_sn": dev.Sn,
				"device_id": dev.ID,
				"delta":     deltaMap,
				"desired":   prevDesired,
				"timestamp": now.UnixMilli(),
				"source":    "admin",
				"operator":  strings.TrimSpace(in.Operator),
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

// DeviceShadowListFilter 影子列表筛选
type DeviceShadowListFilter struct {
	Sn           string
	SnExact      bool
	OnlineStatus *int16
	HasShadow    *bool // true=仅有 device_shadow 记录的设备
}

// DeviceShadowListItem 影子列表行（PG 为主，详情再拉 Redis）
type DeviceShadowListItem struct {
	DeviceID        int64      `json:"device_id"`
	Sn              string     `json:"sn"`
	FirmwareVersion string     `json:"firmware_version,omitempty"`
	OnlineStatus    int16      `json:"online_status"`
	DisplayOnline   int16      `json:"display_online"`
	Battery         *int32     `json:"battery,omitempty"`
	RunState        string     `json:"run_state,omitempty"`
	HasReported     bool       `json:"has_reported"`
	HasDesired      bool       `json:"has_desired"`
	LastReportTime  *time.Time `json:"last_report_time,omitempty"`
	ShadowUpdatedAt *time.Time `json:"shadow_updated_at,omitempty"`
	LastActiveAt    *time.Time `json:"last_active_at,omitempty"`
}

func (e *PlatformDeviceService) deviceShadowListQuery(f DeviceShadowListFilter) *gorm.DB {
	q := e.Orm.Table("device AS d").
		Joins("LEFT JOIN device_shadow AS ds ON ds.device_id = d.id").
		Where("d.deleted_at IS NULL")
	if s := strings.TrimSpace(f.Sn); s != "" {
		if f.SnExact {
			q = q.Where("d.sn = ?", strings.ToUpper(s))
		} else {
			q = q.Where("d.sn ILIKE ?", "%"+strings.ToUpper(s)+"%")
		}
	}
	if f.OnlineStatus != nil {
		q = q.Where("d.online_status = ?", *f.OnlineStatus)
	}
	if f.HasShadow != nil {
		if *f.HasShadow {
			q = q.Where("ds.device_id IS NOT NULL")
		} else {
			q = q.Where("ds.device_id IS NULL")
		}
	}
	return q
}

// ListDeviceShadows 分页查询设备影子列表
func (e *PlatformDeviceService) ListDeviceShadows(page, pageSize int, f DeviceShadowListFilter) ([]DeviceShadowListItem, int64, error) {
	if e.Orm == nil {
		return nil, 0, fmt.Errorf("orm nil")
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	var total int64
	sub := e.deviceShadowListQuery(f).Select("d.id").Distinct()
	if err := e.Orm.Table("(?) AS _cnt", sub).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	type row struct {
		DeviceID        int64           `gorm:"column:device_id"`
		Sn              string          `gorm:"column:sn"`
		FirmwareVersion string          `gorm:"column:firmware_version"`
		OnlineStatus    int16           `gorm:"column:online_status"`
		LastActiveAt    *time.Time      `gorm:"column:last_active_at"`
		Reported        json.RawMessage `gorm:"column:reported"`
		Desired         json.RawMessage `gorm:"column:desired"`
		LastReportTime  *time.Time      `gorm:"column:last_report_time"`
		ShadowUpdatedAt *time.Time      `gorm:"column:shadow_updated_at"`
	}
	var rows []row
	offset := (page - 1) * pageSize
	listQ := e.deviceShadowListQuery(f)
	selectSQL := `d.id AS device_id, d.sn, d.firmware_version, d.online_status, d.last_active_at,
		ds.reported, ds.desired, ds.last_report_time, ds.updated_at AS shadow_updated_at`
	err := listQ.Select(selectSQL).
		Order("COALESCE(ds.updated_at, d.updated_at) DESC NULLS LAST, d.id DESC").
		Limit(pageSize).Offset(offset).Scan(&rows).Error
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "desired") {
		listQ = e.deviceShadowListQuery(f)
		err = listQ.Select(`d.id AS device_id, d.sn, d.firmware_version, d.online_status, d.last_active_at,
			ds.reported, ds.last_report_time, ds.updated_at AS shadow_updated_at`).
			Order("COALESCE(ds.updated_at, d.updated_at) DESC NULLS LAST, d.id DESC").
			Limit(pageSize).Offset(offset).Scan(&rows).Error
	}
	if err != nil {
		log.Printf("[ShadowList] ❌ 查询失败: page=%d, pageSize=%d, error=%v", page, pageSize, err)
		return nil, 0, err
	}

	log.Printf("[ShadowList] ✅ 查询成功: total=%d, rows_count=%d, page=%d, pageSize=%d",
		total, len(rows), page, pageSize)

	if len(rows) > 0 {
		log.Printf("[ShadowList] 📋 第一条数据: device_id=%d, sn=%s, online_status=%d",
			rows[0].DeviceID, rows[0].Sn, rows[0].OnlineStatus)
	}

	out := make([]DeviceShadowListItem, 0, len(rows))
	for _, r := range rows {
		item := DeviceShadowListItem{
			DeviceID:        r.DeviceID,
			Sn:              r.Sn,
			FirmwareVersion: strings.TrimSpace(r.FirmwareVersion),
			OnlineStatus:    r.OnlineStatus,
			DisplayOnline:   displayOnlineFromLastActive(r.LastActiveAt, r.OnlineStatus),
			HasReported:     jsonObjectNonEmpty(r.Reported),
			HasDesired:      jsonObjectNonEmpty(r.Desired),
			LastReportTime:  r.LastReportTime,
			ShadowUpdatedAt: r.ShadowUpdatedAt,
			LastActiveAt:    r.LastActiveAt,
		}
		if len(r.Reported) > 0 {
			var rep map[string]interface{}
			if json.Unmarshal(r.Reported, &rep) == nil {
				if v, ok := rep["battery"].(float64); ok {
					b := int32(v)
					item.Battery = &b
				}
				if v, ok := rep["run_state"].(string); ok {
					item.RunState = v
				} else if v, ok := rep["power_status"].(string); ok {
					item.RunState = v
				}
			}
		}
		out = append(out, item)
	}

	log.Printf("[ShadowList] 📤 最终返回: out_count=%d, total=%d", len(out), total)
	if len(out) > 0 {
		log.Printf("[ShadowList] 📤 返回的第一条: sn=%s, display_online=%d, has_reported=%v",
			out[0].Sn, out[0].DisplayOnline, out[0].HasReported)
	}

	return out, total, nil
}

func jsonObjectNonEmpty(raw json.RawMessage) bool {
	s := strings.TrimSpace(string(raw))
	return s != "" && s != "{}" && s != "null"
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
