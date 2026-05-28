package logic

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
)

// DeliverDeviceWs 投递设备 WebSocket 消息：优先本机连接，否则 Redis relay（多副本）
func DeliverDeviceWs(deviceKey, sn string, data interface{}) (svc.WsDeliverResult, error) {
	keys := wsDeliveryKeys(deviceKey, sn)
	for _, k := range keys {
		lock.RLock()
		dc, ok := deviceConnMap[k]
		lock.RUnlock()
		if !ok {
			continue
		}
		if err := dc.WriteJSON(data); err != nil {
			return svc.WsDeliverResult{}, fmt.Errorf("WebSocket 写入失败(key=%s): %w", k, err)
		}
		return svc.WsDeliverResult{LocalWriteOk: true}, nil
	}

	if wsRelayRedis != nil {
		pctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		var lastErr error
		for _, k := range keys {
			if err := relayPublishWsPush(pctx, k, data); err != nil {
				lastErr = err
				continue
			}
			return svc.WsDeliverResult{RelayPublishOk: true}, nil
		}
		if lastErr != nil {
			return svc.WsDeliverResult{}, lastErr
		}
	}

	return svc.WsDeliverResult{}, fmt.Errorf("设备 WebSocket 未连接: %s", strings.Join(keys, ","))
}

func wsDeliveryKeys(deviceKey, sn string) []string {
	seen := make(map[string]struct{}, 2)
	var out []string
	add := func(k string) {
		k = strings.TrimSpace(k)
		if k == "" {
			return
		}
		if _, ok := seen[k]; ok {
			return
		}
		seen[k] = struct{}{}
		out = append(out, k)
	}
	add(deviceKey)
	add(sn)
	return out
}
