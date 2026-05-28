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
	"github.com/jacklau/audio-ai-platform/services/device/internal/device/shadow"
	"github.com/jacklau/audio-ai-platform/services/device/internal/deviceauthsvc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/model"
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

// DeviceWs 设备长连接入口（简化版）
//
// 流程：
//  1. 握手前校验：提取JWT Token + 验证Token有效性 + 检查设备状态
//     - Token无效/过期 → 拒绝连接（401）
//     - 设备未激活 → 拒绝连接（403）
//     - 设备已禁用 → 拒绝连接（403）
//  2. 升级 HTTP → WebSocket
//  3. 直接建立长连接（无需首包签名认证）
//  4. 更新在线状态、加入连接池、启动心跳
//  5. 进入消息监听循环（可立即收发消息）
func (l *DeviceWsLogic) DeviceWs(w http.ResponseWriter, r *http.Request) {
	logx.Infof("====================================")
	logx.Infof("[WS] 📡 收到新的WebSocket连接请求")
	logx.Infof("[WS] 客户端地址: %s", r.RemoteAddr)
	logx.Infof("[WS] User-Agent: %s", r.UserAgent())
	logx.Infof("====================================")

	// ══════════════════════════════════════════════
	// 步骤0: 握手前校验 - JWT Token验证 + 设备状态检查
	// ══════════════════════════════════════════════
	token := deviceauthsvc.ExtractDeviceWSBearerToken(r)
	if token == "" {
		l.writeWSHandshakeJSON(w, http.StatusUnauthorized, errorx.CodeTokenInvalid,
			"缺少设备访问凭证：请在请求头添加 Authorization: Bearer <token>，或使用 ?token= / ?access_token=")
		return
	}

	authSvc := deviceauthsvc.New(l.svcCtx)

	principal, err := authSvc.VerifyDeviceToken(l.ctx, token)
	if err != nil {
		logx.Errorf("[WS] ❌ 握手前设备 JWT 校验失败: %v", err)
		l.writeWSHandshakeFromError(w, err)
		return
	}
	logx.Infof("[WS] ✅ JWT Token校验通过: device_id=%d sn=%s", principal.DeviceID, principal.DeviceSN)

	deviceID := principal.DeviceID
	deviceSN := principal.DeviceSN
	snUpper := strings.ToUpper(strings.TrimSpace(deviceSN))

	if statusErr := l.checkDeviceStatusForWS(deviceID); statusErr != nil {
		logx.Errorf("[WS] ❌ 设备状态检查失败: device_id=%d error=%v", deviceID, statusErr)
		l.writeWSHandshakeJSON(w, http.StatusForbidden, errorx.CodeDeviceNotFound, statusErr.Error())
		return
	}
	logx.Infof("[WS] ✅ 设备状态正常: device_id=%d (已激活且未禁用)", deviceID)

	// ══════════════════════════════════════════════
	// 步骤1: 升级 HTTP → WebSocket
	// ══════════════════════════════════════════════
	conn, err := upGrader.Upgrade(w, r, nil)
	if err != nil {
		logx.Errorf("[WS] ❌ WebSocket升级失败: %v", err)
		return
	}

	logx.Infof("✅ ✅ ✅ [WS] WebSocket连接建立成功!")
	logx.Infof("   设备ID:   %d", deviceID)
	logx.Infof("   设备SN:   %s", deviceSN)
	logx.Infof("   客户端IP: %s", r.RemoteAddr)

	defer conn.Close()

	deviceIdStr := fmt.Sprintf("%d", deviceID)

	// ══════════════════════════════════════════════
	// 步骤2: 初始化连接管理
	// ══════════════════════════════════════════════

	// ① 更新设备在线状态为"在线"
	l.updateDeviceOnlineStatus(deviceID)

	dc := &wsDeviceConn{conn: conn}

	// ② 加入连接池（供SendCmdToDevice等外部函数使用）
	lock.Lock()
	deviceConnMap[deviceIdStr] = dc
	if snUpper != "" {
		deviceConnMap[snUpper] = dc // 冗余键：与其它按 SN 寻址的逻辑兼容（同一条连接）
	}
	lock.Unlock()

	// ③ 补发离线期间的pending指令
	go l.afterWSAuthFlushPendingInstructions(deviceID)

	// ④ 补发离线期间的诊断指令（异步，不阻塞主流程）
	go l.flushCachedDiagnosisCommands(deviceID, snUpper)

	logx.Infof("[WS] 📍 设备已加入在线列表: device_id=%s sn_key=%s (当前在线设备数: %d)", deviceIdStr, snUpper, len(deviceConnMap))

	// ④ 注册断开清理逻辑
	defer func() {
		lock.Lock()
		delete(deviceConnMap, deviceIdStr)
		if snUpper != "" {
			delete(deviceConnMap, snUpper)
		}
		lock.Unlock()

		logx.Infof("[WS] 🔴 设备离线: device_id=%s (剩余在线设备数: %d)", deviceIdStr, len(deviceConnMap))

		l.handleDeviceDisconnect(deviceIdStr, deviceID)
	}()

	// ══════════════════════════════════════════════
	// 步骤3: 配置心跳机制
	// ══════════════════════════════════════════════
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
		logx.Debugf("[WS] 💓 收到设备 %s 的Pong响应", deviceIdStr)
		conn.SetReadDeadline(time.Now().Add(pongTimeout))
		return nil
	})

	go l.startHeartbeat(dc, deviceIdStr, pingInterval)

	logx.Infof("[WS] 🔄 进入消息监听循环...")

	// ══════════════════════════════════════════════
	// 步骤4: 进入消息监听循环（双向通信）
	// ══════════════════════════════════════════════
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logx.Errorf("[WS] ❌ 设备 %s 连接异常断开: %v", deviceIdStr, err)
			} else {
				logx.Infof("[WS] ℹ️  设备 %s 正常断开连接: %v", deviceIdStr, err)
			}
			break
		}

		logx.Debugf("[WS] 📥 收到设备 %s 消息 (类型:%d, 长度:%d): %s",
			deviceIdStr, messageType, len(message), truncateStringForLog(string(message), 100))

		l.handleDeviceMessage(deviceIdStr, message)
	}

	logx.Infof("[WS] 👋 设备 %s 连接处理结束", deviceIdStr)
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
		// 传递 l (DeviceWsLogic) 给处理函数，使其能访问数据库和连接池
		HandleDeviceRebootResponse(l, deviceId, response)
	case "diagnosis", "collect_log":
		l.handleDiagnosisResponse(deviceId, response)
	default:
		logx.Infof("[CMD_RESPONSE] 未知指令反馈: device_id=%s, cmd=%s", deviceId, cmd)
	}
}

// handleDiagnosisResponse 处理设备诊断指令响应
// 设备执行完日志收集等诊断指令后回传结果
func (l *DeviceWsLogic) handleDiagnosisResponse(deviceId string, response map[string]interface{}) {
	traceID, _ := response["trace_id"].(string)
	status, _ := response["status"].(string)
	code, _ := response["code"].(float64)

	logx.Infof("[DIAGNOSIS_RESPONSE] 收到诊断指令响应: device_id=%s, trace_id=%s, status=%s, code=%v",
		deviceId, traceID, status, int(code))

	if status == "success" {
		logx.Infof("[DIAGNOSIS_RESPONSE] ✅ 诊断指令执行成功: trace_id=%s, device_id=%s", traceID, deviceId)
	} else {
		errorMsg, _ := response["error_msg"].(string)
		logx.Errorf("[DIAGNOSIS_RESPONSE] ❌ 诊断指令执行失败: trace_id=%s, device_id=%s, error=%s",
			traceID, deviceId, errorMsg)
	}

	logContent, hasLogContent := response["log_content"].(string)
	if hasLogContent && logContent != "" {
		logSize, _ := response["log_size"].(float64)
		logMD5, _ := response["log_md5"].(string)
		logx.Infof("[DIAGNOSIS_RESPONSE] 日志内容接收: trace_id=%s, size=%d bytes, md5=%s",
			traceID, int(logSize), logMD5)
	}
}

// flushCachedDiagnosisCommands 设备上线后自动下发缓存的诊断指令
// 在WebSocket连接成功并认证后异步调用
// 参数 deviceID int64: 设备ID
// 参数 deviceSN string: 设备序列号（大写）
func (l *DeviceWsLogic) flushCachedDiagnosisCommands(deviceID int64, deviceSN string) {
	if deviceSN == "" {
		logx.Slowf("[DiagnosisCache] 设备SN为空，跳过缓存指令补发: device_id=%d", deviceID)
		return
	}

	time.Sleep(300 * time.Millisecond)

	cache := GetDiagnosisCache(l.svcCtx)
	cachedCommands, err := cache.FlushCachedCommands(deviceSN)
	if err != nil {
		logx.Errorf("[DiagnosisCache] 获取缓存指令失败: device_sn=%s, err=%v", deviceSN, err)
		return
	}

	if len(cachedCommands) == 0 {
		logx.Debugf("[DiagnosisCache] 无待补发的缓存指令: device_sn=%s", deviceSN)
		return
	}

	deviceIDStr := fmt.Sprintf("%d", deviceID)
	successCount := 0
	failCount := 0

	for i, cmd := range cachedCommands {
		logx.Infof("[DiagnosisCache] 📤 补发缓存指令 [%d/%d]: trace_id=%s, device_sn=%s, cmd_action=%s",
			i+1, len(cachedCommands), cmd.TraceID, deviceSN, cmd.CmdAction)

		if l.svcCtx.WsDeliver != nil {
			result, err := l.svcCtx.WsDeliver(deviceIDStr, deviceSN, cmd)
			if err != nil {
				logx.Errorf("[DiagnosisCache] ❌ 补发失败: trace_id=%s, err=%v", cmd.TraceID, err)
				failCount++
				continue
			}
			if result.LocalWriteOk || result.RelayPublishOk {
				successCount++
				logx.Infof("[DiagnosisCache] ✅ 补发成功: trace_id=%s, local=%v, relay=%v",
					cmd.TraceID, result.LocalWriteOk, result.RelayPublishOk)
			} else {
				failCount++
				logx.Slowf("[DiagnosisCache] ⚠️ 补发未成功（设备可能已离线）: trace_id=%s", cmd.TraceID)
			}
		} else if l.svcCtx.WsPushJSON != nil {
			err := l.svcCtx.WsPushJSON(deviceIDStr, cmd)
			if err != nil {
				logx.Errorf("[DiagnosisCache] ❌ WsPushJSON补发失败: trace_id=%s, err=%v", cmd.TraceID, err)
				failCount++
				continue
			}
			successCount++
			logx.Infof("[DiagnosisCache] ✅ WsPushJSON补发成功: trace_id=%s", cmd.TraceID)
		} else {
			logx.Errorf("[DiagnosisCache] 无可用的推送通道，无法补发")
			failCount++
		}

		time.Sleep(100 * time.Millisecond)
	}

	logx.Infof("[DiagnosisCache] 📊 缓存指令补发完成: device_sn=%s, total=%d, success=%d, fail=%d",
		deviceSN, len(cachedCommands), successCount, failCount)
}

// SendCmdToDevice 给外部调用：发送指令到设备（兼容旧注入，内部走 DeliverDeviceWs）
func SendCmdToDevice(deviceId string, data interface{}) error {
	r, err := DeliverDeviceWs(deviceId, "", data)
	if err != nil {
		logx.Errorf("设备 %s 未在线或未建立 WebSocket 连接: %v", deviceId, err)
		return err
	}
	if r.LocalWriteOk || r.RelayPublishOk {
		return nil
	}
	return fmt.Errorf("设备 WebSocket 未连接: %s", deviceId)
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
		SET online_status = $2,
		    updated_at = NOW()
		WHERE id = $1
		  AND deleted_at IS NULL
	`

	result, err := l.svcCtx.DB.ExecContext(l.ctx, query, deviceID, DeviceOnlineStatusOnline)
	if err != nil {
		logx.Errorf("[WS] 更新设备在线状态失败: device_id=%d, error=%v", deviceID, err)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		logx.Infof("[WS] ✅ DB 设备在线状态已更新: device_id=%d", deviceID)
	} else {
		logx.Infof("[WS] ⚠️ DB 在线状态写入影响行数为 0 device_id=%d（仍将刷新 Redis 心跳，避免误判在线 TTL）", deviceID)
	}
	// commandsvc.isDeviceOnline 依赖 Redis TTL；不因「本次 UPDATE 影响 0 行」而跳过写入，否则会话已建立但投递仍走不可靠路径
	l.syncDeviceOnlineStatusToRedis(deviceID)
}

// syncDeviceOnlineStatusToRedis 同步设备在线状态到Redis
// 用于让commandsvc的isDeviceOnline()能正确识别设备在线状态，从而实时推送指令
func (l *DeviceWsLogic) syncDeviceOnlineStatusToRedis(deviceID int64) {
	if l.svcCtx == nil || l.svcCtx.Redis == nil {
		logx.Infof("[WS] ℹ️  Redis未配置，跳过在线状态同步")
		return
	}

	device, err := l.svcCtx.DeviceRepo.FindById(l.ctx, deviceID)
	if err != nil {
		logx.Errorf("[WS] ❌ 查询设备信息失败（无法同步Redis）: device_id=%d, error=%v", deviceID, err)
		return
	}

	if device == nil || device.Sn == "" {
		logx.Errorf("[WS] ❌ 设备不存在或SN为空（无法同步Redis）: device_id=%d", deviceID)
		return
	}

	sn := strings.ToUpper(strings.TrimSpace(device.Sn))
	onlineKey := shadow.OnlineKey(sn)

	err = l.svcCtx.Redis.Set(l.ctx, onlineKey, "1", 30*time.Minute).Err()
	if err != nil {
		logx.Errorf("[WS] ❌ 设置Redis在线状态失败: sn=%s, key=%s, error=%v", sn, onlineKey, err)
		return
	}

	logx.Infof("[WS] ✅ Redis在线状态已同步: sn=%s, key=%s, TTL=30分钟", sn, onlineKey)
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
	n, _, err := cmdsvc.DispatchPendingInstructions(l.ctx, deviceID, sn)
	if err != nil {
		logx.Errorf("[WS] Flush pending: DispatchPendingInstructions 失败 device_id=%d: %v", deviceID, err)
		return
	}
	if n > 0 {
		logx.Infof("[WS] ✅ 上线补发 pending 指令经 WebSocket device_id=%d sn=%s count=%d", deviceID, sn, n)
	}
}

// checkDeviceStatusForWS WebSocket握手前检查设备状态
//
// 检查项：
//   - 设备是否存在
//   - 设备是否已激活（status=1，正常）
//   - 设备是否被禁用（status=2）或停用（status=3）
//   - 设备是否未注册（status=4）
//
// 返回 error: 状态异常时返回具体原因；正常时返回nil
func (l *DeviceWsLogic) checkDeviceStatusForWS(deviceID int64) error {
	if l.svcCtx == nil || l.svcCtx.DeviceRepo == nil {
		return fmt.Errorf("服务上下文未初始化")
	}

	device, err := l.svcCtx.DeviceRepo.FindById(l.ctx, deviceID)
	if err != nil {
		return fmt.Errorf("查询设备信息失败: %w", err)
	}

	if device == nil || device.ID == 0 {
		return fmt.Errorf("设备不存在（device_id=%d），请先注册", deviceID)
	}

	switch device.Status {
	case model.DeviceStatusNormal:
		logx.Infof("[WS] ✅ 设备状态检查通过: id=%d sn=%s status=正常(1)", device.ID, device.Sn)
		return nil

	case model.DeviceStatusUnregistered:
		return fmt.Errorf("设备未激活（sn=%s），请先调用 /api/device/register 接口激活设备", device.Sn)

	case model.DeviceStatusDisabled:
		return fmt.Errorf("设备已被管理员禁用（sn=%s），无法建立连接，请联系客服", device.Sn)

	case model.DeviceStatusInactive:
		return fmt.Errorf("设备已停用或报废（sn=%s），无法使用", device.Sn)

	default:
		logx.Infof("[WS] ⚠️ 设备状态未知: id=%d status=%d (允许连接)", device.ID, device.Status)
		return nil
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
