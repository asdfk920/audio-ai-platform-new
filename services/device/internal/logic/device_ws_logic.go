package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
)

type DeviceWsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeviceWsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeviceWsLogic {
	return &DeviceWsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// 全局：保存所有设备长连接
var (
	deviceConnMap = make(map[string]*websocket.Conn)
	lock          sync.RWMutex
)

// 升级为 websocket
var upGrader = &websocket.Upgrader{
	ReadBufferSize:  10240,
	WriteBufferSize: 10240,
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许跨域
	},
}

// 设备连接请求结构体
type DeviceConnectReq struct {
	DeviceId string `json:"deviceId"`
}

// DeviceWs 设备长连接入口（带安全认证）
func (l *DeviceWsLogic) DeviceWs(w http.ResponseWriter, r *http.Request) {
	// 1. 升级 HTTP → WebSocket
	conn, err := upGrader.Upgrade(w, r, nil)
	if err != nil {
		logx.Error("ws upgrade error:", err)
		return
	}

	// 2. 立即进行身份认证（设备必须在10秒内发送认证消息）
	authLogic := NewWsAuthLogic(l.ctx, l.svcCtx)
	authResp, authErr := authLogic.AuthenticateDevice(conn)

	if authErr != nil || !authResp.Success {
		logx.Errorf("设备认证失败: %v", authErr)

		if authErr == nil {
			_ = conn.WriteJSON(map[string]interface{}{
				"type":    "error",
				"message": "认证失败，连接将被关闭",
			})
		}

		time.Sleep(1 * time.Second)
		conn.Close()
		return
	}

	deviceId := fmt.Sprintf("%d", authResp.DeviceID)

	defer conn.Close()

	logx.Infof("[WS] 设备认证成功并上线: device_id=%d, sn=%s", authResp.DeviceID, deviceId)

	lock.Lock()
	deviceConnMap[deviceId] = conn
	lock.Unlock()

	defer func() {
		lock.Lock()
		delete(deviceConnMap, deviceId)
		lock.Unlock()
		logx.Infof("[WS] 设备离线: device_id=%d", authResp.DeviceID)
	}()

	pingInterval := time.Duration(54) * time.Second
	if l.svcCtx.Config.WebSocket.PingInterval != "" {
		if d, err := time.ParseDuration(l.svcCtx.Config.WebSocket.PingInterval); err == nil {
			pingInterval = d
		}
	}

	pongTimeout := time.Duration(60) * time.Second
	if l.svcCtx.Config.WebSocket.PongTimeout != "" {
		if d, err := time.ParseDuration(l.svcCtx.Config.WebSocket.PongTimeout); err == nil {
			pongTimeout = d
		}
	}

	conn.SetReadDeadline(time.Now().Add(pongTimeout))
	conn.SetPongHandler(func(appData string) error {
		conn.SetReadDeadline(time.Now().Add(pongTimeout))
		return nil
	})

	go l.startHeartbeat(conn, deviceId, pingInterval)

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			logx.Errorf("[WS] 设备 %s 连接断开: %v", deviceId, err)
			break
		}

		l.handleDeviceMessage(deviceId, message)
	}
}

// startHeartbeat 启动心跳保活协程
func (l *DeviceWsLogic) startHeartbeat(conn *websocket.Conn, deviceId string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		err := conn.WriteMessage(websocket.PingMessage, []byte("ping"))
		if err != nil {
			logx.Errorf("设备 %s 心跳发送失败: %v", deviceId, err)
			break
		}
		logx.Debugf("设备 %s 心跳发送成功", deviceId)
	}
}

// handleDeviceMessage 处理设备消息
func (l *DeviceWsLogic) handleDeviceMessage(deviceId string, message []byte) {
	var msg map[string]interface{}
	if err := json.Unmarshal(message, &msg); err != nil {
		logx.Errorf("设备 %s 消息解析失败: %v", deviceId, err)
		return
	}

	msgType, _ := msg["type"].(string)
	switch msgType {
	case "status":
		logx.Infof("设备 %s 上报状态: %s", deviceId, string(message))
	case "heartbeat":
		logx.Debugf("设备 %s 心跳响应", deviceId)
	default:
		logx.Infof("设备 %s 消息: type=%s, data=%s", deviceId, msgType, string(message))
	}
}

// SendCmdToDevice 给外部调用：发送指令到设备
func SendCmdToDevice(deviceId string, data interface{}) error {
	lock.RLock()
	conn, ok := deviceConnMap[deviceId]
	lock.RUnlock()

	if !ok {
		logx.Error("设备 %s 未在线或未建立 WebSocket 连接", deviceId)
		return nil
	}

	return conn.WriteJSON(data)
}

// GetOnlineDevices 获取当前在线设备列表
func GetOnlineDevices() []string {
	lock.RLock()
	defer lock.RUnlock()

	devices := make([]string, 0, len(deviceConnMap))
	for id := range deviceConnMap {
		devices = append(devices, id)
	}
	return devices
}

// IsDeviceOnline 检查设备是否在线
func IsDeviceOnline(deviceId string) bool {
	lock.RLock()
	defer lock.RUnlock()

	_, ok := deviceConnMap[deviceId]
	return ok
}
