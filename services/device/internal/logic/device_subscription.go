package logic

import (
	"strconv"
	"strings"
	"sync"

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

func BuildDeviceStatusChangeMessage(shadow interface{}) map[string]interface{} {
	return map[string]interface{}{
		"cmd":  "device_status_change",
		"data": shadow,
	}
}

func (dc *wsUserConn) WriteDeviceStatusChange(deviceSN string, shadow interface{}) error {
	msg := BuildDeviceStatusChangeMessage(shadow)
	msg["device_sn"] = deviceSN
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
