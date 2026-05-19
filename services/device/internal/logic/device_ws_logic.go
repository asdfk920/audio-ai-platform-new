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
	logx.Infof("====================================")
	logx.Infof("[WS] 📡 收到新的WebSocket连接请求")
	logx.Infof("[WS] 客户端地址: %s", r.RemoteAddr)
	logx.Infof("[WS] User-Agent: %s", r.UserAgent())
	logx.Infof("====================================")

	// 1. 升级 HTTP → WebSocket
	conn, err := upGrader.Upgrade(w, r, nil)
	if err != nil {
		logx.Errorf("[WS] ❌ WebSocket升级失败: %v", err)
		return
	}

	logx.Infof("[WS] ✅ WebSocket连接建立成功")
	logx.Infof("[WS] 🔐 开始身份认证流程（10秒超时）...")

	// 2. 立即进行身份认证（设备必须在10秒内发送认证消息）
	authLogic := NewWsAuthLogic(l.ctx, l.svcCtx)
	authResp, authErr := authLogic.AuthenticateDevice(conn, r)

	if authErr != nil || !authResp.Success {
		logx.Errorf("[WS] ❌ 设备认证失败，立即关闭连接")
		logx.Errorf("[WS] 错误原因: %v", authErr)
		// AuthenticateDevice 在失败分支通常已下发 auth_response；此处不再额外等待。
		_ = conn.Close()
		return
	}

	deviceId := fmt.Sprintf("%d", authResp.DeviceID)

	defer conn.Close()

	logx.Infof("")
	logx.Infof("====================================")
	logx.Infof("[WS] ✅✅✅ 设备已成功上线! ✅✅✅")
	logx.Infof("====================================")
	logx.Infof("  设备ID:     %s", deviceId)
	logx.Infof("  连接状态:   已建立长连接")
	logx.Infof("  心跳间隔:   54秒 (自动保活)")
	logx.Infof("====================================")

	// ① 更新设备在线状态为"在线"
	l.updateDeviceOnlineStatus(authResp.DeviceID)

	// ② 检查并自动下发缓存指令（设备上线时）
	go l.deliverCachedInstructions(authResp.DeviceID, deviceId, conn)

	lock.Lock()
	deviceConnMap[deviceId] = conn
	lock.Unlock()

	logx.Infof("[WS] 📍 设备已加入在线列表: device_id=%s (当前在线设备数: %d)", deviceId, len(deviceConnMap))

	defer func() {
		lock.Lock()
		delete(deviceConnMap, deviceId)
		lock.Unlock()
		logx.Infof("[WS] 🔴 设备离线: device_id=%s (剩余在线设备数: %d)", deviceId, len(deviceConnMap))
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

	logx.Infof("[WS] ⚙️  心跳配置: ping_interval=%v, pong_timeout=%v", pingInterval, pongTimeout)

	conn.SetReadDeadline(time.Now().Add(pongTimeout))
	conn.SetPongHandler(func(appData string) error {
		logx.Debugf("[WS] 💓 收到设备 %s 的Pong响应", deviceId)
		conn.SetReadDeadline(time.Now().Add(pongTimeout))
		return nil
	})

	go l.startHeartbeat(conn, deviceId, pingInterval)

	logx.Infof("[WS] 🔄 进入消息监听循环...")

	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logx.Errorf("[WS] ❌ 设备 %s 连接异常断开: %v", deviceId, err)
			} else {
				logx.Infof("[WS] ℹ️  设备 %s 正常断开连接: %v", deviceId, err)
			}
			break
		}

		logx.Debugf("[WS] 📥 收到设备 %s 消息 (类型:%d, 长度:%d): %s",
			deviceId, messageType, len(message), truncateStringForLog(string(message), 100))

		l.handleDeviceMessage(deviceId, message)
	}

	logx.Infof("[WS] 👋 设备 %s 连接处理结束", deviceId)
}

// startHeartbeat 启动心跳保活协程

// startHeartbeat 启动心跳保活协程
func (l *DeviceWsLogic) startHeartbeat(conn *websocket.Conn, deviceId string, interval time.Duration) {
	logx.Infof("[WS] 💓 心跳保活协程已启动: device_id=%s, interval=%v", deviceId, interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	pingCount := 0
	for range ticker.C {
		pingCount++
		err := conn.WriteMessage(websocket.PingMessage, []byte("ping"))
		if err != nil {
			logx.Errorf("[WS] 💔 设备 %s 心跳发送失败(第%d次): %v", deviceId, pingCount, err)
			break
		}
		logx.Debugf("[WS] 💓 设备 %s 心跳发送成功 (第%d次)", deviceId, pingCount)
	}

	logx.Infof("[WS] ⏹️  心跳保活协程已停止: device_id=%s (共发送%d次ping)", deviceId, pingCount)
}

// handleDeviceMessage 处理设备消息
// 支持的消息类型：
//   - status: 设备状态上报
//   - heartbeat: 心跳响应
//   - cmd_response: 指令执行反馈（重启、播放等）
//
// 参数 deviceId string: 设备ID
// 参数 message []byte: 原始消息字节
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
	case "cmd_response":
		l.handleCommandResponse(deviceId, msg)
	default:
		logx.Infof("设备 %s 消息: type=%s, data=%s", deviceId, msgType, string(message))
	}
}

// handleCommandResponse 处理设备指令反馈
// 根据不同的指令类型分发到对应的处理函数
// 参数 deviceId string: 设备ID
// 参数 response map[string]interface{}: 反馈消息体
func (l *DeviceWsLogic) handleCommandResponse(deviceId string, response map[string]interface{}) {
	cmd, _ := response["cmd"].(string)

	logx.Infof("[CMD_RESPONSE] 收到设备反馈: device_id=%s, cmd=%s", deviceId, cmd)

	switch cmd {
	case "reboot":
		HandleDeviceRebootResponse(deviceId, response)
	default:
		logx.Infof("[CMD_RESPONSE] 未知指令反馈: device_id=%s, cmd=%s", deviceId, cmd)
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

// updateDeviceOnlineStatus 更新设备在线状态为"在线"
// 在设备WebSocket认证成功后自动调用
// 参数 deviceID int64: 设备ID（整数）
func (l *DeviceWsLogic) updateDeviceOnlineStatus(deviceID int64) {
	if l.svcCtx == nil || l.svcCtx.DB == nil {
		return
	}

	query := `
		UPDATE public.device
		SET online_status = 1,
		    updated_at = NOW()
		WHERE id = $1
		  AND deleted_at IS NULL
	`

	result, err := l.svcCtx.DB.ExecContext(l.ctx, query, deviceID)
	if err != nil {
		logx.Errorf("[WS] 更新设备在线状态失败: device_id=%d, error=%v", deviceID, err)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		logx.Infof("[WS] ✅ 设备在线状态已更新: device_id=%d, status=在线", deviceID)
	}
}

// deliverCachedInstructions 下发缓存的指令（设备上线时自动调用）
// 从数据库查询该设备的缓存指令，通过WebSocket逐条下发
// 参数 deviceID int64: 设备ID（整数）
// 参数 deviceId string: 设备ID（字符串）
// 参数 conn *websocket.Conn: WebSocket连接
func (l *DeviceWsLogic) deliverCachedInstructions(deviceID int64, deviceId string, conn *websocket.Conn) {
	if l.svcCtx == nil || l.svcCtx.DB == nil {
		return
	}

	time.Sleep(1 * time.Second)

	query := `
		SELECT id, params, created_at
		FROM public.device_instruction
		WHERE device_id = $1
		  AND status = 0
		  AND (expires_at IS NULL OR expires_at >= CURRENT_TIMESTAMP)
		ORDER BY created_at ASC
		LIMIT 10
	`

	rows, err := l.svcCtx.DB.QueryContext(l.ctx, query, deviceID)
	if err != nil {
		logx.Errorf("[CACHE] 查询缓存指令失败: device_id=%s, error=%v", deviceId, err)
		return
	}
	defer rows.Close()

	var cachedCount int
	for rows.Next() {
		var instructionId int64
		var paramsJSON []byte
		var createdAt time.Time

		if err := rows.Scan(&instructionId, &paramsJSON, &createdAt); err != nil {
			logx.Errorf("[CACHE] 扫描缓存指令失败: instruction_id=%d, error=%v", instructionId, err)
			continue
		}

		var params map[string]interface{}
		if err := json.Unmarshal(paramsJSON, &params); err != nil {
			logx.Errorf("[CACHE] 解析指令参数失败: instruction_id=%d, error=%v", instructionId, err)
			continue
		}

		logx.Infof("[CACHE] 准备下发缓存指令: device_id=%s, instruction_id=%d, cmd=%v",
			deviceId, instructionId, params["cmd"])

		cmdMsg := map[string]interface{}{
			"type":       "cmd",
			"cmd":        params["cmd"],
			"request_id": params["request_id"],
			"timestamp":  time.Now().UTC().Format(time.RFC3339),
			"cached":     true,
		}

		if sendErr := conn.WriteJSON(cmdMsg); sendErr != nil {
			logx.Errorf("[CACHE] 下发缓存指令失败: device_id=%s, instruction_id=%d, error=%v",
				deviceId, instructionId, sendErr)
			break
		}

		updateQuery := `
			UPDATE public.device_instruction
			SET status = 1,
			    updated_at = NOW()
			WHERE id = $1
		`
		l.svcCtx.DB.ExecContext(l.ctx, updateQuery, instructionId)

		cachedCount++
		logx.Infof("[CACHE] ✅ 缓存指令已下发: device_id=%s, instruction_id=%d",
			deviceId, instructionId)

		time.Sleep(500 * time.Millisecond)
	}

	if cachedCount > 0 {
		logx.Infof("[CACHE] 📦 设备上线后共补发 %d 条缓存指令: device_id=%s", cachedCount, deviceId)
	}
}

// truncateStringForLog 截断字符串用于日志显示（避免日志过长）
func truncateStringForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
