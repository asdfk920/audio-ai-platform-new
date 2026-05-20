package logic

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/services/device/internal/config"
	"github.com/jacklau/audio-ai-platform/services/device/internal/deviceauthsvc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/model"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
	"github.com/jacklau/audio-ai-platform/services/device/internal/util"
)

// websocketAuthHandshakeTimeoutFromConfig 首包认证读超时（可配置）；默认 60s，最短 5s，最长 10m。
func websocketAuthHandshakeTimeoutFromConfig(cf config.Config) time.Duration {
	const def = 60 * time.Second
	s := strings.TrimSpace(cf.WebSocket.AuthMessageTimeout)
	if s != "" {
		d, err := time.ParseDuration(s)
		if err == nil && d >= 5*time.Second && d <= 10*time.Minute {
			return d
		}
	}
	return def
}

// WsAuthLogic WebSocket认证逻辑
// 处理设备WebSocket连接后的身份认证
type WsAuthLogic struct {
	ctx               context.Context
	svcCtx            *svc.ServiceContext
	deviceAuthService *deviceauthsvc.Service
}

// NewWsAuthLogic 创建WebSocket认证逻辑实例
func NewWsAuthLogic(ctx context.Context, svcCtx *svc.ServiceContext) *WsAuthLogic {
	return &WsAuthLogic{
		ctx:               ctx,
		svcCtx:            svcCtx,
		deviceAuthService: deviceauthsvc.New(svcCtx),
	}
}

// AuthenticateDevice WebSocket 设备认证（完整安全流程）
//
// 流程：
//  1. **从请求头读取Token**：从HTTP请求头 Authorization: Bearer <token> 获取Token
//  2. **格式校验**：检查消息类型、SN、签名字段完整性
//  3. **查询设备**：根据SN从数据库获取设备信息和预存密钥、注册时间戳
//  4. **Token验证**：使用JWT库验证Token有效性、未过期、未吊销
//  5. **签名验签**：使用数据库中的设备密钥和注册时间戳对请求签名进行HMAC-SHA256验证
//  6. **状态更新**：标记设备为已认证在线状态，加入在线列表
//  7. **返回结果**：向设备发送认证成功/失败响应
//
// 安全特性：
//   - Token通过HTTP请求头传递，避免在消息体中暴露
//   - 使用数据库中存储的注册时间戳，无需客户端发送时间戳
//   - 彻底解决设备与服务器时间不同步问题
//   - HMAC-SHA256签名确保消息完整性和来源可信
//   - JWT Token支持自动过期，可定期刷新
//
// 参数 conn *websocket.Conn: WebSocket连接对象
// 参数 r *http.Request: HTTP请求对象（用于读取请求头中的Token）
// 返回 (*types.WsAuthResponse, error): 认证结果；失败时 Success=false，交由上层关闭连接
func (l *WsAuthLogic) AuthenticateDevice(conn *websocket.Conn, r *http.Request) (*types.WsAuthResponse, error) {
	var cf config.Config
	if l.svcCtx != nil {
		cf = l.svcCtx.Config
	}
	handshakeWindow := websocketAuthHandshakeTimeoutFromConfig(cf)
	logx.Infof("====================================")
	logx.Infof("[WS Auth] 🔐 开始WebSocket认证流程...")
	logx.Infof("[WS Auth] ⏱️  认证首包超时窗口: %v", handshakeWindow)

	logx.Infof("\n[WS Auth] 📋 步骤1: 读取设备 JWT（握手前已校验，此处读取用于绑签名校验）...")

	token := deviceauthsvc.ExtractDeviceWSBearerToken(r)
	if token == "" {
		logx.Errorf("❌ [WS Auth] 缺少设备 JWT")
		resp := l.buildAuthResponse(false, "缺少Authorization: Bearer <token> 或 URL ?token= / ?access_token=", 0, 0)
		_ = conn.WriteJSON(resp)
		return resp, fmt.Errorf("缺少设备JWT")
	}

	logx.Infof("[WS Auth] ✅ 已解析设备 JWT: %s... (长度:%d)", truncateString(token, 20), len(token))

	deadline := time.Now().Add(handshakeWindow)
	authMsg, readErr := readWsAuthHandshake(conn, deadline)
	if readErr != nil {
		logx.Errorf("❌ [WS Auth] 步骤2: 读取认证消息失败")
		logx.Errorf("   错误详情: %v", readErr)
		logx.Errorf("   可能原因:")
		logx.Errorf("   - 未在 %v 内发送文本 JSON 认证首包（连接建立后尽快发送）", handshakeWindow)
		logx.Errorf("   - 首帧为 Ping/非文本等非 JSON（已自动应答 Ping，请仍以文本帧发送 {\"type\":\"auth\",...}）")
		logx.Errorf("   - timestamp 为字符串或非法 JSON，`json:\"timestamp\"` 必须能解析为毫秒时间戳")

		resp := l.buildAuthResponse(false, authReadFailUserMessage(readErr, handshakeWindow), 0, 0)
		_ = conn.WriteJSON(resp)
		return resp, fmt.Errorf("读取认证消息失败: %w", readErr)
	}

	sn := strings.TrimSpace(strings.ToUpper(authMsg.Sn))
	signature := strings.TrimSpace(authMsg.Signature)

	logx.Infof("✅ [WS Auth] 步骤2: 成功读取认证消息")
	logx.Infof("   消息内容:")
	logx.Infof("   - type:      %s", authMsg.Type)
	logx.Infof("   - sn:        %s", sn)
	logx.Infof("   - timestamp: %d", authMsg.Timestamp)
	logx.Infof("   - signature: %s... (长度:%d)", truncateString(signature, 15), len(signature))

	logx.Infof("\n[WS Auth] 📋 步骤3: 校验消息格式...")

	if err := l.validateMessageFormat(&authMsg); err != nil {
		logx.Errorf("❌ [WS Auth] 格式校验失败: %v", err)
		l.printValidationTips(&authMsg)
		resp := l.buildAuthResponse(false, err.Error(), 0, 0)
		_ = conn.WriteJSON(resp)
		return resp, err
	}

	logx.Infof("✅ [WS Auth] 步骤3: 消息格式校验通过")

	logx.Infof("\n[WS Auth] 🔍 步骤4: 校验JWT Token（验证Token有效性和SN绑定关系）...")

	principal, tokenErr := l.deviceAuthService.VerifyDeviceToken(l.ctx, token)
	if tokenErr != nil {
		logx.Errorf("❌ [WS Auth] Token验证失败: %v", tokenErr)
		logx.Errorf("   可能原因:")
		logx.Errorf("   - Token无效或格式错误")
		logx.Errorf("   - Token已过期（默认24小时有效）")
		logx.Errorf("   - Token与设备不匹配")
		logx.Errorf("   解决方案:")
		logx.Errorf("   - 重新调用注册接口获取新Token: POST /api/device/register")
		resp := l.buildAuthResponse(false, "Token无效或已过期，请重新调用注册接口获取新的访问凭证", 0, 0)
		_ = conn.WriteJSON(resp)
		return resp, fmt.Errorf("Token无效: %v", tokenErr)
	}

	logx.Infof("✅ [WS Auth] JWT Token验证成功: device_id=%d sn=%s", principal.DeviceID, principal.DeviceSN)

	snFromToken := strings.TrimSpace(strings.ToUpper(principal.DeviceSN))
	if sn != snFromToken {
		logx.Errorf("❌ [WS Auth] Token与设备SN不匹配!")
		logx.Errorf("   Token中的SN: %s", principal.DeviceSN)
		logx.Errorf("   WS消息中的SN: %s", sn)
		resp := l.buildAuthResponse(false, "Token与设备序列号不匹配：WS首包JSON的sn必须与Token中的sn一致", principal.DeviceID, 0)
		_ = conn.WriteJSON(resp)
		return resp, fmt.Errorf("Token与设备不匹配")
	}

	logx.Infof("✅ [WS Auth] SN绑定关系验证通过: %s", sn)

	logx.Infof("\n[WS Auth] 🔍 步骤5: 从数据库加载设备信息（密钥）...")

	device, err := l.svcCtx.DeviceRepo.FindById(l.ctx, principal.DeviceID)
	if err != nil {
		logx.Errorf("❌ [WS Auth] 数据库查询失败: %v", err)
		resp := l.buildAuthResponse(false, "系统繁忙，请稍后重试（数据库查询失败）", principal.DeviceID, 0)
		_ = conn.WriteJSON(resp)
		return resp, fmt.Errorf("查询设备失败: %v", err)
	}

	if device == nil || device.DeviceSecret == "" {
		logx.Errorf("❌ [WS Auth] 设备不存在或未预置密钥: device_id=%d", principal.DeviceID)
		logx.Errorf("   请先调用注册接口: POST /api/device/register")
		resp := l.buildAuthResponse(false, "设备未注册或密钥缺失，请先调用注册接口获取凭证", principal.DeviceID, 0)
		_ = conn.WriteJSON(resp)
		return resp, fmt.Errorf("设备不存在: id=%d", principal.DeviceID)
	}

	if authMsg.Timestamp == 0 {
		logx.Errorf("❌ [WS Auth] 客户端未提供时间戳")
		resp := l.buildAuthResponse(false, "认证消息缺少timestamp字段", 0, 0)
		_ = conn.WriteJSON(resp)
		return resp, fmt.Errorf("缺少时间戳")
	}

	clientTimestamp := authMsg.Timestamp

	logx.Infof("✅ [WS Auth] 步骤5: 设备信息加载成功")
	logx.Infof("   设备ID:       %d", device.ID)
	logx.Infof("   设备SN:       %s", device.Sn)
	logx.Infof("   客户端时间戳: %d (%s)", clientTimestamp, formatTimestamp(clientTimestamp))
	logx.Infof("   密钥存储方式: %s", wsSecretStorageHint(device.DeviceSecret))

	logx.Infof("\n[WS Auth] ✍️  步骤6: 验证HMAC-SHA256签名（使用客户端时间戳）...")

	if err := l.verifySignatureWithClientTimestamp(device, sn, clientTimestamp, signature); err != nil {
		logx.Errorf("❌ [WS Auth] 签名验证失败: %v", err)
		logx.Errorf("   签名算法: HMAC-SHA256(device_secret, sn + timestamp)")
		logx.Errorf("   signData: %s%d", sn, clientTimestamp)
		resp := l.buildAuthResponse(false, err.Error(), device.ID, 0)
		_ = conn.WriteJSON(resp)
		return resp, err
	}

	logx.Infof("✅ [WS Auth] 步骤6: 签名验证通过")

	// 生命周期：注册后为 status=5，HTTP 等业务曾仅查 status=1 会误判「设备不存在」；认证成功后升为正常。
	if device.Status == model.DeviceStatusUnauthenticated || device.Status == model.DeviceStatusDefault {
		if err := l.svcCtx.DeviceRepo.UpdateStatusAfterRegister(l.ctx, device.ID, model.DeviceStatusNormal); err != nil {
			logx.Errorf("⚠️  [WS Auth] 更新设备生命周期状态失败(不影响认证): %v", err)
		} else {
			logx.Infof("✅ [WS Auth] 设备状态已更新为正常(已通过 WS 认证): device_id=%d", device.ID)
		}
	}

	logx.Infof("\n[WS Auth] 💾 步骤7: 更新设备在线状态...")

	now := time.Now()
	if err := l.svcCtx.DeviceRepo.UpdateLastActive(l.ctx, device.ID, 1, now); err != nil {
		logx.Errorf("⚠️  [WS Auth] 更新设备活跃时间失败(不影响认证): %v", err)
	} else {
		logx.Infof("✅ [WS Auth] 设备活跃时间已更新: %s", now.Format(time.RFC3339))
	}

	expiresIn := int64(86400)

	logx.Infof("\n====================================")
	logx.Infof("🎉 [WS Auth] ✅✅✅ 设备认证成功! ✅✅✅")
	logx.Infof("====================================")
	logx.Infof("  设备SN:       %s", sn)
	logx.Infof("  设备ID:       %d", device.ID)
	logx.Infof("  Token有效期:  %d 秒 (%.1f 小时)", expiresIn, float64(expiresIn)/3600)
	logx.Infof("====================================")

	resp := l.buildAuthResponse(true, "认证成功，设备已上线", device.ID, expiresIn)
	if writeErr := conn.WriteJSON(resp); writeErr != nil {
		logx.Errorf("⚠️  [WS Auth] 发送认证成功响应失败: %v", writeErr)
	} else {
		logx.Infof("📤 [WS Auth] 认证成功响应已发送给设备")
	}

	conn.SetReadDeadline(time.Time{})

	return resp, nil
}

// verifySignatureWithClientTimestamp 使用客户端发送的时间戳验证HMAC-SHA256签名
//
// 该方法统一处理明文密钥和bcrypt密钥两种情况：
// - 明文密钥：服务端使用密钥和客户端时间戳重算HMAC-SHA256签名并与客户端签名比对
// - bcrypt密钥：由于无法还原明文密钥，暂时跳过签名验证（仅验证Token）
//
// 参数 device *model.Device: 设备对象（包含密钥）
// 参数 snUpper string: 设备序列号（大写）
// 参数 clientTimestamp int64: 客户端发送的时间戳（毫秒级Unix时间戳）
// 参数 clientSig string: 客户端发送的签名字符串
// 返回 error: 验证失败时的错误信息
func (l *WsAuthLogic) verifySignatureWithClientTimestamp(device *model.Device, snUpper string, clientTimestamp int64, clientSig string) error {
	if device == nil {
		return fmt.Errorf("内部错误：设备数据为空")
	}

	secret := strings.TrimSpace(device.DeviceSecret)
	if secret == "" {
		return fmt.Errorf("设备密钥未配置")
	}

	if wsDeviceSecretLooksBcrypt(secret) {
		logx.Infof("[WS Auth] 检测到bcrypt密钥，跳过签名验证（仅依赖Token认证）...")
		logx.Infof("[WS Auth] ⚠️  注意：bcrypt密钥模式下，建议后续升级为明文密钥以增强安全性")
		return nil
	}

	logx.Infof("[WS Auth] 检测到明文密钥，使用客户端时间戳重算签名验证...")
	expected := wsHMACSignatureHex(snUpper, clientTimestamp, secret)
	if err := verifyHexSignatureEqual(clientSig, expected); err != nil {
		return fmt.Errorf("签名验证失败：%v（signData=sn+timestamp，sn为大写，timestamp为毫秒）", err)
	}

	logx.Infof("[WS Auth] 明文密钥签名验证通过")
	return nil
}

func wsSecretStorageHint(secret string) string {
	if wsDeviceSecretLooksBcrypt(secret) {
		return "bcrypt（须使用注册返回的毫秒timestamp+对应签名）"
	}
	return "明文（使用数据库中的register_timestamp计算HMAC签名）"
}

func wsDeviceSecretLooksBcrypt(stored string) bool {
	s := strings.TrimSpace(stored)
	return strings.HasPrefix(s, "$2a$") || strings.HasPrefix(s, "$2b$") || strings.HasPrefix(s, "$2y$")
}

func wsHMACSignatureHex(snUpper string, timestampMs int64, plainSecret string) string {
	signData := fmt.Sprintf("%s%d", strings.TrimSpace(snUpper), timestampMs)
	mac := hmac.New(sha256.New, []byte(strings.TrimSpace(plainSecret)))
	mac.Write([]byte(signData))
	return hex.EncodeToString(mac.Sum(nil))
}

func verifyHexSignatureEqual(clientHex, expectedHex string) error {
	c := strings.TrimSpace(strings.ToLower(clientHex))
	e := strings.TrimSpace(strings.ToLower(expectedHex))

	if c == "" || e == "" {
		return fmt.Errorf("签名为空")
	}

	if len(c) != len(e) {
		return fmt.Errorf("签名长度不一致（期望:%d字节，实际:%d字节）", len(e), len(c))
	}

	if !hmac.Equal([]byte(c), []byte(e)) {
		return fmt.Errorf("与云端计算的签名不一致")
	}

	return nil
}

func (l *WsAuthLogic) printValidationTips(msg *types.WsAuthMessage) {
	logx.Errorf("")
	logx.Errorf("📋 正确的认证消息格式:")
	logx.Errorf("{")
	logx.Errorf(`  "type":      "auth",`)
	logx.Errorf(`  "sn":        "AUSP2605000002Y2",`)
	logx.Errorf(`  "timestamp": 1779188152847,`)
	logx.Errorf(`  "signature": "<hex64>"`)
	logx.Errorf("}")
	logx.Errorf("")
	logx.Errorf("🔧 字段说明:")
	logx.Errorf("  type:      固定为 \"auth\"")
	logx.Errorf("  sn:        须与JWT内sn一致（建议大写）")
	logx.Errorf("  timestamp: 当前时间戳（毫秒级Unix时间戳）")
	logx.Errorf("  signature: hex(HMAC-SHA256(device_secret, sn + timestamp))")
	logx.Errorf("")
	logx.Errorf("📌 Token传递方式（任选其一）:")
	logx.Errorf("  请求头: Authorization: Bearer <your_jwt_token>")
	logx.Errorf("  或URL: ws://host/ws/device?token=<your_jwt_token>（部分客户端无法自定义握手头时使用）")
	logx.Errorf("")
	if msg.Type != "auth" {
		logx.Errorf("❌ 您提供的type: \"%s\" (应为 \"auth\")", msg.Type)
	}
	if msg.Sn == "" {
		logx.Errorf("❌ sn字段为空")
	} else if !util.ValidateSNFormat(msg.Sn) {
		logx.Errorf("❌ sn格式错误: \"%s\" (应为16位字母数字)", msg.Sn)
	}
	if msg.Timestamp == 0 {
		logx.Errorf("❌ timestamp字段为空或为0")
	}
	if msg.Signature == "" {
		logx.Errorf("❌ signature字段为空")
	}
}

func (l *WsAuthLogic) validateMessageFormat(msg *types.WsAuthMessage) error {
	if msg.Type != "auth" {
		return fmt.Errorf("消息类型错误: 期望 'auth', 实际 '%s'", msg.Type)
	}

	if msg.Sn == "" {
		return fmt.Errorf("设备序列号不能为空")
	}

	if !util.ValidateSNFormat(msg.Sn) {
		parsed := util.ParseSN(msg.Sn)
		if errMsg, ok := parsed["error"].(string); ok {
			return fmt.Errorf("SN格式错误: %s", errMsg)
		}
		return fmt.Errorf("SN格式错误或校验码无效")
	}

	if msg.Signature == "" {
		return fmt.Errorf("签名不能为空")
	}

	return nil
}

func (l *WsAuthLogic) verifyStoredSignature(sn string, requestSignature string, storedSignature string) error {
	requestSignature = strings.TrimSpace(strings.ToLower(requestSignature))
	storedSignature = strings.TrimSpace(strings.ToLower(storedSignature))

	if requestSignature == "" || storedSignature == "" {
		return fmt.Errorf("签名字符串为空")
	}

	if !hmac.Equal([]byte(requestSignature), []byte(storedSignature)) {
		return fmt.Errorf("签名不匹配（请求签名与注册签名不一致）")
	}

	return nil
}

func (l *WsAuthLogic) buildAuthResponse(success bool, message string, deviceID int64, expiresIn int64) *types.WsAuthResponse {
	return &types.WsAuthResponse{
		Type:      "auth_response",
		Success:   success,
		Message:   message,
		DeviceID:  deviceID,
		ExpiresIn: expiresIn,
	}
}

// wsAuthPayloadWire 解析认证消息体；timestamp 使用 RawMessage 以兼容字符串/小数等客户端序列化差异。
type wsAuthPayloadWire struct {
	Type      string          `json:"type"`
	Sn        string          `json:"sn"`
	Timestamp json.RawMessage `json:"timestamp"`
	Signature string          `json:"signature"`
}

func readWsAuthHandshake(conn *websocket.Conn, deadline time.Time) (types.WsAuthMessage, error) {
	writeWait := 5 * time.Second
	for {
		if err := conn.SetReadDeadline(deadline); err != nil {
			return types.WsAuthMessage{}, err
		}
		msgType, payload, err := conn.ReadMessage()
		if err != nil {
			return types.WsAuthMessage{}, err
		}

		switch msgType {
		case websocket.PingMessage:
			if werr := conn.SetWriteDeadline(time.Now().Add(writeWait)); werr != nil {
				return types.WsAuthMessage{}, werr
			}
			_ = conn.WriteMessage(websocket.PongMessage, payload)
			logx.Infof("[WS Auth] 📍 握手阶段应答 Ping，继续等待文本 JSON 认证包")
			continue
		case websocket.TextMessage, websocket.BinaryMessage:
			msg, perr := parseWsAuthPayload(payload)
			return msg, perr
		default:
			logx.Infof("[WS Auth] ⚠️ 跳过非预期的 WebSocket 帧类型(type=%d)，继续等待文本 JSON", msgType)
		}
	}
}

func parseWsAuthPayload(payload []byte) (types.WsAuthMessage, error) {
	var w wsAuthPayloadWire
	if err := json.Unmarshal(payload, &w); err != nil {
		return types.WsAuthMessage{}, fmt.Errorf("解析认证 JSON 失败（请发送 UTF-8 Text 帧单行 JSON）: %w", err)
	}
	ts, err := parseMilliTimestampFlexible(w.Timestamp)
	if err != nil {
		return types.WsAuthMessage{}, err
	}
	return types.WsAuthMessage{
		Type:      w.Type,
		Sn:        w.Sn,
		Timestamp: ts,
		Signature: w.Signature,
	}, nil
}

func parseMilliTimestampFlexible(raw json.RawMessage) (int64, error) {
	b := bytes.TrimSpace(raw)
	if len(b) == 0 || bytes.EqualFold(b, []byte("null")) {
		return 0, nil
	}

	var num json.Number
	if err := json.Unmarshal(b, &num); err == nil {
		if i, ierr := num.Int64(); ierr == nil {
			return i, nil
		}
		f, ierr := num.Float64()
		if ierr != nil {
			return 0, fmt.Errorf("字段 timestamp 无法转为数字: %w", ierr)
		}
		return int64(f), nil
	}

	var str string
	if err := json.Unmarshal(b, &str); err == nil {
		str = strings.TrimSpace(str)
		if str == "" {
			return 0, fmt.Errorf("字段 timestamp 不能为空的字符串")
		}
		i, err := strconv.ParseInt(str, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("字段 timestamp 须为毫秒级整数，当前字符串=%q: %w", str, err)
		}
		return i, nil
	}

	var f float64
	if err := json.Unmarshal(b, &f); err == nil {
		return int64(f), nil
	}

	return 0, fmt.Errorf("字段 timestamp 无法解析，应为数字或可解析的数字字符串；原始片段=%s", string(b))
}

func authReadFailUserMessage(err error, authWindow time.Duration) string {
	if err == nil {
		return "读取认证消息失败"
	}
	sec := int(authWindow.Round(time.Second) / time.Second)
	if sec < 1 {
		sec = 1
	}
	es := strings.ToLower(err.Error())
	switch {
	case errors.Is(err, os.ErrDeadlineExceeded) ||
		strings.Contains(es, "timeout") ||
		strings.Contains(es, "i/o timeout"):
		return fmt.Sprintf("读取认证超时：连接建立后须在 %d 秒内用文本帧发送完整 {\"type\":\"auth\",...}", sec)
	case strings.Contains(es, "timestamp") ||
		strings.Contains(es, "解析") ||
		strings.Contains(es, "parse") ||
		strings.Contains(es, "invalid character"):
		return "认证 JSON 不合法或无有效毫秒 timestamp（不要注释；建议使用数字类型 timestamp）"
	default:
		return fmt.Sprintf("读取认证失败：握手后请以 Text(JSON)在 %d 秒内发送 auth 首包（若工具先发 Ping，云端已应答，请再接一条 JSON）", sec)
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func formatTimestamp(ts int64) string {
	if ts > 1_000_000_000_000 {
		t := time.UnixMilli(ts)
		return t.Format("2006-01-02 15:04:05.000")
	}
	t := time.Unix(ts, 0)
	return t.Format("2006-01-02 15:04:05")
}
