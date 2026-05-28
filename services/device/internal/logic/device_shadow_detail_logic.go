package logic

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	shadowv2 "github.com/jacklau/audio-ai-platform/services/device/internal/device/shadowv2"
	"github.com/jacklau/audio-ai-platform/services/device/internal/middleware/jwt"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
)

// DeviceShadowDetailLogic 设备影子详情查询逻辑（用于设备详情页展示）
type DeviceShadowDetailLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewDeviceShadowDetailLogic 创建设备影子详情查询逻辑实例
func NewDeviceShadowDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeviceShadowDetailLogic {
	return &DeviceShadowDetailLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// DeviceShadowDetailResp 设备影子详情响应（优化用于前端展示）
type DeviceShadowDetailResp struct {
	DeviceSN        string                   `json:"device_sn"`         // 设备序列号
	Found           bool                     `json:"found"`             // 是否找到影子
	BasicInfo       ShadowBasicInfo          `json:"basic_info"`        // 基本信息
	ReportedAttrs   []ShadowAttribute        `json:"reported_attrs"`    // 上报属性列表（格式化）
	DesiredAttrs    []ShadowAttribute        `json:"desired_attrs"`     // 期望属性列表（格式化）
	RawReported     map[string]interface{}   `json:"raw_reported"`      // 原始 reported 数据
	RawDesired      map[string]interface{}   `json:"raw_desired"`       // 原始 desired 数据
	VersionInfo     VersionInfo              `json:"version_info"`      // 版本信息
	TimelineInfo    TimelineInfo             `json:"timeline_info"`      // 时间线信息
	StatusHistory   []StatusChangeEvent      `json:"status_history"`    // 状态变更历史（最近10条）
	Metadata        map[string]interface{}   `json:"metadata"`          // 元数据
}

// ShadowBasicInfo 影子基本信息
type ShadowBasicInfo struct {
	DeviceSN       string `json:"device_sn"`        // 设备序列号
	Status         string `json:"status"`            // 状态：online/offline/abnormal
	StatusText     string `json:"status_text"`       // 状态文本：在线/离线/异常
	IsOnline       bool   `json:"is_online"`         // 是否在线
	FirmwareVersion string `json:"firmware_version"`  // 固件版本（从 metadata 提取）
	IPAddress      string `json:"ip_address"`        // IP 地址（从 metadata 提取）
	BatteryLevel   int    `json:"battery_level"`     // 电量（从 reported 提取）
	RunState       string `json:"run_state"`         // 运行状态（从 reported 提取）
}

// ShadowAttribute 影子属性（格式化展示）
type ShadowAttribute struct {
	Key          string      `json:"key"`           // 属性名
	Label        string      `json:"label"`         // 显示标签
	Value        interface{} `json:"value"`         // 值
	ValueText    string      `json:"value_text"`    // 值的文本显示
	Type         string      `json:"type"`          // 值类型：string/number/boolean/json/object
	UpdateTime   int64       `json:"update_time"`    // 更新时间戳
	UpdateTimeFormatted string `json:"update_time_formatted"` // 格式化的更新时间
	Changed      bool        `json:"changed"`        // 是否与 desired 不同（仅对 reported）
	Category     string      `json:"category"`      // 分类：system/user/custom
}

// VersionInfo 版本信息
type VersionInfo struct {
	CurrentVersion int64 `json:"current_version"` // 当前版本号
	UpdateCount    int64 `json:"update_count"`    // 更新次数（估算）
	LastUpdated    int64 `json:"last_updated"`    // 最后更新时间
}

// TimelineInfo 时间线信息
type TimelineInfo struct {
	CreatedAt      int64 `json:"created_at"`       // 创建时间
	LastUpdatedAt  int64 `json:"last_updated_at"`  // 最后更新时间
	Duration       int64 `json:"duration_seconds"`  // 存在时长（秒）
	ActiveRatio    float64 `json:"active_ratio"`    // 活跃度比例（0-1）
}

// StatusChangeEvent 状态变更事件
type StatusChangeEvent struct {
	FromStatus string `json:"from_status"` // 原状态
	ToStatus   string `json:"to_status"`   // 新状态
	ChangeTime int64  `json:"change_time"`  // 变更时间
	Reason     string `json:"reason"`       // 变更原因
}

// GetDeviceShadowDetail 获取设备影子详情（用于详情页展示）
func (l *DeviceShadowDetailLogic) GetDeviceShadowDetail(deviceSN string) (*DeviceShadowDetailResp, error) {
	// 1. 鉴权：获取当前用户ID
	userID, ok := jwt.GetUserIdFromContext(l.ctx)
	if !ok || userID <= 0 {
		return nil, fmt.Errorf("请先登录")
	}

	// 2. 参数校验
	deviceSN = strings.TrimSpace(deviceSN)
	if deviceSN == "" {
		return nil, fmt.Errorf("设备序列号不能为空")
	}

	// 3. 查询 V2 影子数据
	store := l.svcCtx.GetRedisShadowStore()
	if store == nil {
		return nil, fmt.Errorf("影子服务未初始化")
	}

	shadow, err := store.GetShadow(l.ctx, deviceSN)
	if err != nil {
		if err.Error() == "shadow not found" {
			// 返回空结果，但标记为未找到
			return &DeviceShadowDetailResp{
				DeviceSN: deviceSN,
				Found:    false,
			}, nil
		}
		return nil, fmt.Errorf("查询影子失败: %v", err)
	}

	// 4. 构建优化的响应数据
	resp := l.buildDetailResponse(shadow)

	logx.Infof("[ShadowDetail] 用户 %d 查询设备 %s 影子详情成功", userID, deviceSN)

	return resp, nil
}

// buildDetailResponse 构建详情响应
func (l *DeviceShadowDetailLogic) buildDetailResponse(shadow *shadowv2.DeviceShadow) *DeviceShadowDetailResp {
	resp := &DeviceShadowDetailResp{
		DeviceSN:      shadow.DeviceSN,
		Found:         true,
		RawReported:   shadow.Reported,
		RawDesired:    shadow.Desired,
		Metadata:      map[string]interface{}{},
	}

	// 1. 构建基本信息
	resp.BasicInfo = l.buildBasicInfo(shadow)

	// 2. 格式化属性列表
	resp.ReportedAttrs = l.formatAttributes(shadow.Reported, "reported")
	resp.DesiredAttrs = l.formatAttributes(shadow.Desired, "desired")

	// 标记与 desired 不同的属性
	l.markChangedAttributes(resp.ReportedAttrs, shadow.Desired)

	// 3. 版本信息
	resp.VersionInfo = VersionInfo{
		CurrentVersion: shadow.Version,
		UpdateCount:    shadow.Version - 1, // 初始版本为1，所以更新次数=version-1
		LastUpdated:    shadow.UpdateTime,
	}

	// 4. 时间线信息
	now := time.Now().Unix()
	resp.TimelineInfo = TimelineInfo{
		LastUpdatedAt: shadow.UpdateTime,
		Duration:      now - shadow.UpdateTime,
	}

	// 5. 状态历史（简化版，实际应从事件日志中读取）
	resp.StatusHistory = []StatusChangeEvent{
		{
			ToStatus:   string(shadow.Status),
			ChangeTime: shadow.UpdateTime,
		},
	}

	return resp
}

// buildBasicInfo 构建基本信息
func (l *DeviceShadowDetailLogic) buildBasicInfo(shadow *shadowv2.DeviceShadow) ShadowBasicInfo {
	info := ShadowBasicInfo{
		DeviceSN:   shadow.DeviceSN,
		Status:     string(shadow.Status),
		StatusText: getStatusChineseText(shadow.Status),
		IsOnline:   shadow.Status == shadowv2.StatusOnline,
	}

	// 从 reported 中提取常用字段
	if shadow.Reported != nil {
		if v, ok := shadow.Reported["battery"]; ok {
			switch val := v.(type) {
			case float64:
				info.BatteryLevel = int(val)
			case int64:
				info.BatteryLevel = int(val)
			case int:
				info.BatteryLevel = val
			}
		}

		if v, ok := shadow.Reported["run_state"]; ok {
			if s, ok := v.(string); ok {
				info.RunState = s
			}
		}

		if v, ok := shadow.Reported["power"]; ok {
			if s, ok := v.(string); ok && info.RunState == "" {
				if s == "on" || s == "playing" {
					info.RunState = "运行中"
				} else {
					info.RunState = "待机"
				}
			}
		}
	}

	if shadow.Reported != nil {
		if v, ok := shadow.Reported["firmware_version"]; ok {
			if s, ok := v.(string); ok {
				info.FirmwareVersion = s
			}
		}
		if v, ok := shadow.Reported["firmware"]; ok && info.FirmwareVersion == "" {
			if s, ok := v.(string); ok {
				info.FirmwareVersion = s
			}
		}
		if v, ok := shadow.Reported["ip"]; ok {
			if s, ok := v.(string); ok {
				info.IPAddress = s
			}
		} else if v, ok := shadow.Reported["ip_address"]; ok {
			if s, ok := v.(string); ok {
				info.IPAddress = s
			}
		}
	}

	return info
}

// formatAttributes 格式化属性列表
func (l *DeviceShadowDetailLogic) formatAttributes(attrs map[string]interface{}, attrType string) []ShadowAttribute {
	if attrs == nil || len(attrs) == 0 {
		return []ShadowAttribute{}
	}

	result := make([]ShadowAttribute, 0, len(attrs))

	for key, value := range attrs {
		attr := ShadowAttribute{
			Key:        key,
			Label:      formatAttributeLabel(key),
			Value:      value,
			ValueText:  formatAttributeValue(value),
			Type:       getAttributeType(value),
			Category:   categorizeAttribute(key),
		}

		result = append(result, attr)
	}

	return result
}

// markChangedAttributes 标记与 desired 不同的属性
func (l *DeviceShadowDetailLogic) markChangedAttributes(reportedAttrs []ShadowAttribute, desired map[string]interface{}) {
	if desired == nil {
		return
	}

	for i := range reportedAttrs {
		key := reportedAttrs[i].Key
		if desiredValue, exists := desired[key]; exists {
			// 简单比较值是否不同
			reportedAttrs[i].Changed = !isEqual(reportedAttrs[i].Value, desiredValue)
		}
	}
}

// isEqual 比较两个值是否相等
func isEqual(a, b interface{}) bool {
	switch va := a.(type) {
	case string:
		if vb, ok := b.(string); ok {
			return va == vb
		}
	case float64:
		if vb, ok := b.(float64); ok {
			return va == vb
		}
	case int64:
		switch vb := b.(type) {
		case int64:
			return va == vb
		case float64:
			return float64(va) == vb
		}
	case bool:
		if vb, ok := b.(bool); ok {
			return va == vb
		}
	case nil:
		return b == nil
	}
	return false
}

// getStatusChineseText 获取状态的中文文本
func getStatusChineseText(status shadowv2.DeviceStatus) string {
	switch status {
	case shadowv2.StatusOnline:
		return "在线"
	case shadowv2.StatusOffline:
		return "离线"
	case shadowv2.StatusAbnormal:
		return "异常"
	default:
		return "未知"
	}
}

// formatAttributeLabel 格式化属性标签
func formatAttributeLabel(key string) string {
	labels := map[string]string{
		"power":        "电源状态",
		"volume":       "音量",
		"play_state":   "播放状态",
		"current_song": "当前歌曲",
		"mode":         "模式",
		"brightness":   "亮度",
		"temperature":  "温度",
		"humidity":     "湿度",
		"battery":      "电量",
		"wifi_rssi":    "WiFi信号强度",
		"memory_usage": "内存使用率",
		"cpu_usage":    "CPU使用率",
		"storage_used": "存储使用量",
		"firmware":     "固件版本",
		"ip":           "IP地址",
		"mac":          "MAC地址",
	}

	if label, ok := labels[key]; ok {
		return label
	}

	// 将下划线转为空格，首字母大写
	return strings.Title(strings.ReplaceAll(key, "_", " "))
}

// formatAttributeValue 格式化属性值的显示文本
func formatAttributeValue(value interface{}) string {
	if value == nil {
		return "-"
	}

	switch v := value.(type) {
	case string:
		if v == "" {
			return "-"
		}
		return v
	case float64:
		if v == float64(int(v)) {
			return fmt.Sprintf("%.0f", v)
		}
		return fmt.Sprintf("%.2f", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case int:
		return fmt.Sprintf("%d", v)
	case bool:
		if v {
			return "是"
		}
		return "否"
	default:
		return fmt.Sprintf("%v", value)
	}
}

// getAttributeType 获取属性类型
func getAttributeType(value interface{}) string {
	if value == nil {
		return "null"
	}

	switch value.(type) {
	case string:
		return "string"
	case float64, int64, int:
		return "number"
	case bool:
		return "boolean"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	default:
		return "unknown"
	}
}

// categorizeAttribute 属性分类
func categorizeAttribute(key string) string {
	systemAttrs := map[string]bool{
		"power": true, "battery": true, "firmware": true,
		"memory_usage": true, "cpu_usage": true, "storage_used": true,
		"temperature": true, "wifi_rssi": true, "ip": true, "mac": true,
	}

	mediaAttrs := map[string]bool{
		"volume": true, "play_state": true, "current_song": true,
		"mode": true, "brightness": true, "seek_position": true,
	}

	if systemAttrs[key] {
		return "system"
	}
	if mediaAttrs[key] {
		return "media"
	}
	return "custom"
}
