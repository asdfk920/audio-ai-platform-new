package logic

import (
	"strconv"
	"strings"
	"sync"
	"time"

	shadowv2 "github.com/jacklau/audio-ai-platform/services/device/internal/device/shadowv2"
	"github.com/zeromicro/go-zero/core/logx"
)

type DeviceSubscription struct {
	UserID   int64
	DeviceSN string
}

var (
	deviceSubscriptionMap = make(map[string]map[int64]*wsUserConn) // deviceSN -> userID -> wsUserConn
	subscriptionMu        sync.RWMutex
)

func RegisterDeviceSubscription(userID int64, deviceSN string, dc *wsUserConn) {
	deviceSN = strings.TrimSpace(deviceSN)
	if deviceSN == "" || userID <= 0 || dc == nil {
		return
	}

	subscriptionMu.Lock()
	defer subscriptionMu.Unlock()

	if _, ok := deviceSubscriptionMap[deviceSN]; !ok {
		deviceSubscriptionMap[deviceSN] = make(map[int64]*wsUserConn)
	}
	deviceSubscriptionMap[deviceSN][userID] = dc

	logx.Infof("[DeviceSub] 用户 %d 订阅设备 %s 成功", userID, deviceSN)
}

func RegisterDeviceSubscriptions(userID int64, deviceSNs []string, dc *wsUserConn) []string {
	if len(deviceSNs) == 0 || userID <= 0 || dc == nil {
		return nil
	}

	subscriptionMu.Lock()
	defer subscriptionMu.Unlock()

	success := make([]string, 0, len(deviceSNs))
	for _, sn := range deviceSNs {
		sn = strings.TrimSpace(sn)
		if sn == "" {
			continue
		}
		if _, ok := deviceSubscriptionMap[sn]; !ok {
			deviceSubscriptionMap[sn] = make(map[int64]*wsUserConn)
		}
		deviceSubscriptionMap[sn][userID] = dc
		success = append(success, sn)
	}

	logx.Infof("[DeviceSub] 用户 %d 批量订阅 %d 个设备成功: %v", userID, len(success), success)
	return success
}

func UnregisterDeviceSubscription(userID int64, deviceSN string) {
	deviceSN = strings.TrimSpace(deviceSN)
	if deviceSN == "" || userID <= 0 {
		return
	}

	subscriptionMu.Lock()
	defer subscriptionMu.Unlock()

	if users, ok := deviceSubscriptionMap[deviceSN]; ok {
		delete(users, userID)
		if len(users) == 0 {
			delete(deviceSubscriptionMap, deviceSN)
		}
		logx.Infof("[DeviceSub] 用户 %d 取消订阅设备 %s", userID, deviceSN)
	}
}

func UnregisterDeviceSubscriptions(userID int64, deviceSNs []string) []string {
	if len(deviceSNs) == 0 || userID <= 0 {
		return nil
	}

	subscriptionMu.Lock()
	defer subscriptionMu.Unlock()

	success := make([]string, 0, len(deviceSNs))
	for _, sn := range deviceSNs {
		sn = strings.TrimSpace(sn)
		if sn == "" {
			continue
		}
		if users, ok := deviceSubscriptionMap[sn]; ok {
			delete(users, userID)
			if len(users) == 0 {
				delete(deviceSubscriptionMap, sn)
			}
			success = append(success, sn)
		}
	}

	logx.Infof("[DeviceSub] 用户 %d 批量取消订阅 %d 个设备: %v", userID, len(success), success)
	return success
}

func UnregisterAllDeviceSubscriptions(userID int64) {
	if userID <= 0 {
		return
	}

	subscriptionMu.Lock()
	defer subscriptionMu.Unlock()

	for deviceSN, users := range deviceSubscriptionMap {
		if _, ok := users[userID]; ok {
			delete(users, userID)
			if len(users) == 0 {
				delete(deviceSubscriptionMap, deviceSN)
			}
		}
	}

	logx.Infof("[DeviceSub] 用户 %d 取消所有设备订阅", userID)
}

func GetDeviceSubscribers(deviceSN string) []int64 {
	deviceSN = strings.TrimSpace(deviceSN)
	if deviceSN == "" {
		return nil
	}

	subscriptionMu.RLock()
	defer subscriptionMu.RUnlock()

	users, ok := deviceSubscriptionMap[deviceSN]
	if !ok || len(users) == 0 {
		return nil
	}

	subscribers := make([]int64, 0, len(users))
	for userID := range users {
		subscribers = append(subscribers, userID)
	}

	return subscribers
}

func NotifyDeviceStatusChange(deviceSN string, data interface{}) {
	deviceSN = strings.TrimSpace(deviceSN)
	if deviceSN == "" {
		return
	}

	subscribers := GetDeviceSubscribers(deviceSN)
	if len(subscribers) == 0 {
		logx.Debugf("[DeviceSub] 设备 %s 无订阅者，跳过推送", deviceSN)
		return
	}

	logx.Infof("[DeviceSub] 推送设备 %s 状态变更给 %d 个订阅者", deviceSN, len(subscribers))

	for _, userID := range subscribers {
		NotifyUserWsJSON(userID, data)
	}
}

func BuildDeviceStatusChangeMessage(deviceSN string, reported map[string]interface{}, version int64) map[string]interface{} {
	return map[string]interface{}{
		"type":        "status_change",
		"device_sn":   deviceSN,
		"reported":    reported,
		"version":     version,
		"update_time": time.Now().UnixMilli(),
	}
}

func BuildDeviceShadowPushMessage(shadow *shadowv2.DeviceShadow) map[string]interface{} {
	if shadow == nil {
		return nil
	}
	return map[string]interface{}{
		"type":        "status_change",
		"device_sn":   shadow.DeviceSN,
		"reported":    shadow.Reported,
		"version":     shadow.Version,
		"update_time": shadow.UpdateTime,
		"status":      string(shadow.Status),
	}
}

func (dc *wsUserConn) WriteDeviceStatusChange(deviceSN string, shadow interface{}) error {
	if s, ok := shadow.(*shadowv2.DeviceShadow); ok {
		msg := BuildDeviceShadowPushMessage(s)
		return dc.WriteJSON(msg)
	}
	msg := map[string]interface{}{
		"type":        "status_change",
		"device_sn":   deviceSN,
		"data":        shadow,
		"update_time": time.Now().UnixMilli(),
	}
	return dc.WriteJSON(msg)
}

func GetUserSubscriptionCount(userID int64) int {
	if userID <= 0 {
		return 0
	}

	subscriptionMu.RLock()
	defer subscriptionMu.RUnlock()

	count := 0
	for _, users := range deviceSubscriptionMap {
		if _, ok := users[userID]; ok {
			count++
		}
	}

	return count
}

func LogSubscriptionStats() {
	subscriptionMu.RLock()
	defer subscriptionMu.RUnlock()

	totalDevices := len(deviceSubscriptionMap)
	totalSubscriptions := 0
	for _, users := range deviceSubscriptionMap {
		totalSubscriptions += len(users)
	}

	logx.Infof("[DeviceSub] 统计: %d 个设备被订阅, 总计 %d 个活跃订阅", totalDevices, totalSubscriptions)
}

func FormatUserID(userID int64) string {
	return strconv.FormatInt(userID, 10)
}
