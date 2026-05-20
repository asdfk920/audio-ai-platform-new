package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/device/internal/commandsvc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/deviceauthsvc"
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

// wsDeviceConn 单设备一行：Gorilla WebSocket 同一连接只允许一个写入者并发；心跳 Ping 与 SendCmdToDevice 必须串行，否则易出现异常断连。
type wsDeviceConn struct {
	mu   sync.Mutex
	conn *websocket.Conn
}

func (dc *wsDeviceConn) WriteJSON(v interface{}) error {
	if dc == nil || dc.conn == nil {
		return fmt.Errorf("连接为空")
	}
	dc.mu.Lock()
	defer dc.mu.Unlock()
	return dc.conn.WriteJSON(v)
}

func (dc *wsDeviceConn) WritePing(payload []byte) error {
	if dc == nil || dc.conn == nil {
		return fmt.Errorf("连接为空")
	}
	dc.mu.Lock()
	defer dc.mu.Unlock()
	return dc.conn.WriteMessage(websocket.PingMessage, payload)
}

// 全局：保存所有设备长连接（按 deviceId 字符串键）
var (
	deviceConnMap = make(map[string]*wsDeviceConn)
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

// DeviceWs 设备长连接入口：升级 WebSocket **之前**先校验设备 JWT；通过后升级，再完成首包签名认证，最后登记在线并收发消息。
func (l *DeviceWsLogic) DeviceWs(w http.ResponseWriter, r *http.Request) {
	logx.Infof("====================================")
	logx.Infof("[WS] 📡 收到新的WebSocket连接请求")
	logx.Infof("[WS] 客户端地址: %s", r.RemoteAddr)
	logx.Infof("[WS] User-Agent: %s", r.UserAgent())
	logx.Infof("====================================")

	// 0. 握手前校验：须为已注册设备且 JWT 有效（未建立 WS 即拒绝，避免无效升级）
	token := deviceauthsvc.ExtractDeviceWSBearerToken(r)
	if token == "" {
		l.writeWSHandshakeJSON(w, http.StatusUnauthorized, errorx.CodeTokenInvalid,
			"缺少设备访问凭证：请在请求头添加 Authorization: Bearer <token>，或使用 ?token= / ?access_token=")
		return
	}
	authSvc := deviceauthsvc.New(l.svcCtx)
	if _, err := authSvc.VerifyDeviceToken(l.ctx, token); err != nil {
		logx.Errorf("[WS] ❌ 握手前设备 JWT 校验失败: %v", err)
		l.writeWSHandshakeFromError(w, err)
		return
	}
	logx.Infof("[WS] ✅ 握手前设备 JWT 校验通过")

	// 1. 升级 HTTP → WebSocket
	conn, err := upGrader.Upgrade(w, r, nil)
	if err != nil {
		logx.Errorf("[WS] ❌ WebSocket升级失败: %v", err)
		return
	}

	logx.Infof("[WS] ✅ WebSocket连接建立成功")
	authWindow := websocketAuthHandshakeTimeoutFromConfig(l.svcCtx.Config)
	logx.Infof("[WS] 🔐 开始身份认证（须在 %v 内用文本帧发送完整 auth JSON 首包）...", authWindow)

	// 2. 立即进行身份认证（设备须在配置窗口内发送首包 auth JSON）
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

	dc := &wsDeviceConn{conn: conn}

	// ② 必须先登记连接：SendCmdToDevice / commandsvc.Dispatch 依赖 deviceConnMap
	lock.Lock()
	deviceConnMap[deviceId] = dc
	lock.Unlock()

	// ③ HTTP 在离线时写入的 pending 指令：上线后统一经 WebSocket 信封补发
	go l.afterWSAuthFlushPendingInstructions(authResp.DeviceID)

	logx.Infof("[WS] 📍 设备已加入在线列表: device_id=%s (当前在线设备数: %d)", deviceId, len(deviceConnMap))

	defer func() {
		lock.Lock()
		delete(deviceConnMap, deviceId)
		lock.Unlock()

		logx.Infof("[WS] 🔴 设备离线: device_id=%s (剩余在线设备数: %d)", deviceId, len(deviceConnMap))

		l.handleDeviceDisconnect(deviceId, authResp.DeviceID)
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

	go l.startHeartbeat(dc, deviceId, pingInterval)

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

// startHeartbeat 经 wsDeviceConn 写 Ping，与 SendCmdToDevice 共用写锁，避免 Gorilla/WebSocket 并发写导致异常断连。
func (l *DeviceWsLogic) startHeartbeat(dc *wsDeviceConn, deviceId string, interval time.Duration) {
	logx.Infof("[WS] 💓 心跳保活协程已启动: device_id=%s, interval=%v", deviceId, interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	pingCount := 0
	for range ticker.C {
		pingCount++
		err := dc.WritePing([]byte("ping"))
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
	dc, ok := deviceConnMap[deviceId]
	lock.RUnlock()

	if !ok {
		logx.Errorf("设备 %s 未在线或未建立 WebSocket 连接", deviceId)
		return fmt.Errorf("设备 WebSocket 未连接: %s", deviceId)
	}

	if err := dc.WriteJSON(data); err != nil {
		return fmt.Errorf("WebSocket 写入失败(device_id=%s): %w", deviceId, err)
	}
	return nil
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

// afterWSAuthFlushPendingInstructions WebSocket 认证成功后，将库中 pending(status=1) 经统一通道推到设备。
func (l *DeviceWsLogic) afterWSAuthFlushPendingInstructions(deviceID int64) {
	if l.svcCtx == nil || l.svcCtx.DB == nil {
		return
	}
	time.Sleep(150 * time.Millisecond)

	dev, err := l.svcCtx.DeviceRepo.FindById(l.ctx, deviceID)
	if err != nil || dev == nil {
		logx.Errorf("[WS] Flush pending: 无法读取设备 id=%d: %v", deviceID, err)
		return
	}
	sn := strings.TrimSpace(dev.Sn)
	if sn == "" {
		logx.Errorf("[WS] Flush pending: SN 为空 device_id=%d", deviceID)
		return
	}
	cmdsvc := commandsvc.New(l.svcCtx)
	n, err := cmdsvc.DispatchPendingInstructions(l.ctx, deviceID, sn)
	if err != nil {
		logx.Errorf("[WS] Flush pending: DispatchPendingInstructions 失败 device_id=%d: %v", deviceID, err)
		return
	}
	if n > 0 {
		logx.Infof("[WS] ✅ 上线补发 pending 指令经 WebSocket device_id=%d sn=%s count=%d", deviceID, sn, n)
	}
}

func (l *DeviceWsLogic) handleDeviceDisconnect(deviceId string, deviceID int64) {
	l.updateDeviceOfflineStatus(deviceID)
}

func (l *DeviceWsLogic) updateDeviceOfflineStatus(deviceID int64) {
	if l.svcCtx == nil || l.svcCtx.DB == nil {
		return
	}
	_, err := l.svcCtx.DB.ExecContext(l.ctx, `
		UPDATE public.device
		SET online_status = 0,
		    updated_at = NOW()
		WHERE id = $1
		  AND deleted_at IS NULL
	`, deviceID)
	if err != nil {
		logx.Errorf("[WS] 更新设备离线状态失败: device_id=%d error=%v", deviceID, err)
		return
	}
	logx.Infof("[WS] 设备已更新为离线(DB): device_id=%d", deviceID)
}

// writeWSHandshakeJSON 在未 Upgrade WebSocket 时返回 JSON（HTTP 4xx），便于网关/App 感知「未获准建连」。
func (l *DeviceWsLogic) writeWSHandshakeJSON(w http.ResponseWriter, httpStatus, businessCode int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(errorx.Error(businessCode, msg))
}

func (l *DeviceWsLogic) writeWSHandshakeFromError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	code := errorx.CodeOf(err)
	msg := ""
	var ce *errorx.CodeError
	if errors.As(err, &ce) {
		msg = strings.TrimSpace(ce.GetMsg())
	}
	if msg == "" {
		msg = "设备访问凭证无效或设备不可用"
	}
	st := errorx.HTTPStatusForCode(code)
	if st < 400 {
		st = http.StatusUnauthorized
	}
	l.writeWSHandshakeJSON(w, st, code, msg)
}

// truncateStringForLog 截断字符串用于日志显示（避免日志过长）
func truncateStringForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
