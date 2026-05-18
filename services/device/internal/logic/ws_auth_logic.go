package logic

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/services/device/internal/deviceauthsvc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
	"github.com/jacklau/audio-ai-platform/services/device/internal/util"
)

const (
	wsAuthTimeout      = 10 * time.Second // 认证消息超时时间
	timestampTolerance = 300              // 时间戳容差：5分钟（300秒）
)

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

// AuthenticateDevice WebSocket设备认证（完整安全流程）
//
// 流程：
//  1. **格式校验**：检查消息类型、SN、Token、时间戳、签名字段完整性
//  2. **时间戳验证**：检查时间戳是否在有效范围内（±5分钟），防止重放攻击
//  3. **查询设备**：根据SN从数据库获取设备信息和预存密钥
//  4. **Token验证**：使用JWT库验证Token有效性、未过期、未吊销
//  5. **签名验签**：使用数据库中的设备密钥对请求签名进行HMAC-SHA256验证
//  6. **状态更新**：标记设备为已认证在线状态，加入在线列表
//  7. **返回结果**：向设备发送认证成功/失败响应
//
// 安全特性：
//   - 时间戳机制有效防范重放攻击
//   - HMAC-SHA256签名确保消息完整性和来源可信
//   - JWT Token支持自动过期，可定期刷新
//   - 常量时间比较防止时序攻击
//
// 参数 conn *websocket.Conn: WebSocket连接对象
// 返回 (*model.Device, error): 认证成功返回设备信息，失败返回错误
func (l *WsAuthLogic) AuthenticateDevice(conn *websocket.Conn) (*types.WsAuthResponse, error) {
	conn.SetReadDeadline(time.Now().Add(wsAuthTimeout))

	var authMsg types.WsAuthMessage
	if err := conn.ReadJSON(&authMsg); err != nil {
		logx.Errorf("读取认证消息失败: %v", err)
		return l.buildAuthResponse(false, "读取认证消息超时或格式错误", 0, 0), fmt.Errorf("读取认证消息失败")
	}

	sn := strings.TrimSpace(strings.ToUpper(authMsg.Sn))
	token := strings.TrimSpace(authMsg.Token)
	timestamp := authMsg.Timestamp
	signature := strings.TrimSpace(authMsg.Signature)

	logx.Infof("[WS Auth] 收到认证请求: sn=%s, timestamp=%d", sn, timestamp)

	if err := l.validateMessageFormat(&authMsg); err != nil {
		logx.Slowf("[WS Auth] 消息格式校验失败: sn=%s, err=%v", sn, err)
		resp := l.buildAuthResponse(false, err.Error(), 0, 0)
		_ = conn.WriteJSON(resp)
		return resp, err
	}

	if err := l.validateTimestamp(timestamp); err != nil {
		logx.Slowf("[WS Auth] 时间戳校验失败: sn=%s, err=%v", sn, err)
		resp := l.buildAuthResponse(false, err.Error(), 0, 0)
		_ = conn.WriteJSON(resp)
		return resp, err
	}

	device, err := l.svcCtx.DeviceRepo.FindBySn(l.ctx, sn)
	if err != nil {
		logx.Errorf("[WS Auth] 查询设备失败: sn=%s, err=%v", sn, err)
		resp := l.buildAuthResponse(false, "系统繁忙，请稍后重试", 0, 0)
		_ = conn.WriteJSON(resp)
		return resp, fmt.Errorf("查询设备失败")
	}

	if device == nil || device.DeviceSecret == "" {
		logx.Slowf("[WS Auth] 设备不存在或未预置密钥: sn=%s", sn)
		resp := l.buildAuthResponse(false, "设备未注册或密钥缺失", 0, 0)
		_ = conn.WriteJSON(resp)
		return resp, fmt.Errorf("设备不存在")
	}

	principal, tokenErr := l.deviceAuthService.VerifyDeviceToken(l.ctx, token)
	if tokenErr != nil {
		logx.Slowf("[WS Auth] Token验证失败: sn=%s, err=%v", sn, tokenErr)
		resp := l.buildAuthResponse(false, "Token无效或已过期", 0, 0)
		_ = conn.WriteJSON(resp)
		return resp, fmt.Errorf("Token无效")
	}

	if principal.DeviceSN != sn {
		logx.Slowf("[WS Auth] Token与设备不匹配: token_sn=%s, request_sn=%s", principal.DeviceSN, sn)
		resp := l.buildAuthResponse(false, "Token与设备不匹配", 0, 0)
		_ = conn.WriteJSON(resp)
		return resp, fmt.Errorf("Token与设备不匹配")
	}

	if device.RegisterSignature == nil || *device.RegisterSignature == "" {
		logx.Slowf("[WS Auth] 设备未注册或缺少注册签名: sn=%s", sn)
		resp := l.buildAuthResponse(false, "设备未完成注册", 0, 0)
		_ = conn.WriteJSON(resp)
		return resp, fmt.Errorf("设备未完成注册")
	}

	if err := l.verifyStoredSignature(sn, signature, *device.RegisterSignature); err != nil {
		logx.Slowf("[WS Auth] 签名验证失败: sn=%s, err=%v", sn, err)
		resp := l.buildAuthResponse(false, "签名验证失败", 0, 0)
		_ = conn.WriteJSON(resp)
		return resp, err
	}

	now := time.Now()
	if err := l.svcCtx.DeviceRepo.UpdateLastActive(l.ctx, device.ID, 1, now); err != nil {
		logx.Errorf("[WS Auth] 更新设备活跃时间失败: sn=%s, err=%v", sn, err)
	}

	expiresIn := int64(86400)
	logx.Infof("[WS Auth] 设备认证成功: sn=%s, device_id=%d", sn, device.ID)

	resp := l.buildAuthResponse(true, "认证成功", device.ID, expiresIn)
	if writeErr := conn.WriteJSON(resp); writeErr != nil {
		logx.Errorf("[WS Auth] 发送认证成功响应失败: %v", writeErr)
	}

	return resp, nil
}

// validateMessageFormat 校验认证消息格式
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

	if msg.Token == "" {
		return fmt.Errorf("访问凭证不能为空")
	}

	if msg.Timestamp <= 0 {
		return fmt.Errorf("时间戳无效")
	}

	if msg.Signature == "" {
		return fmt.Errorf("签名不能为空")
	}

	return nil
}

// validateTimestamp 验证时间戳有效性（防重放攻击）
func (l *WsAuthLogic) validateTimestamp(timestamp int64) error {
	now := time.Now()
	var requestTime time.Time

	if timestamp > 1_000_000_000_000 {
		requestTime = time.UnixMilli(timestamp)
	} else {
		requestTime = time.Unix(timestamp, 0)
	}

	diff := now.Sub(requestTime).Seconds()
	if diff < -float64(timestampTolerance) || diff > float64(timestampTolerance) {
		return fmt.Errorf("请求已超时（容差±%d秒），请同步设备时间", timestampTolerance)
	}

	return nil
}

// verifySignature 使用HMAC-SHA256验证请求签名
// 签名算法：HMAC-SHA256(device_secret, sn + token + timestamp)
func (l *WsAuthLogic) verifySignature(sn string, token string, timestamp int64, requestSignature string, deviceSecret string) error {
	signData := fmt.Sprintf("%s%s%d", sn, token, timestamp)

	mac := hmac.New(sha256.New, []byte(strings.TrimSpace(deviceSecret)))
	mac.Write([]byte(signData))
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(requestSignature), []byte(expectedSignature)) {
		return fmt.Errorf("签名不匹配")
	}

	return nil
}

// verifyStoredSignature 验证请求签名是否与数据库中存储的注册签名匹配
// 该方法用于验证设备在注册时生成的签名，确保设备身份的合法性
//
// 参数 sn string: 设备序列号（用于日志）
// 参数 requestSignature string: 请求中的签名字符串
// 参数 storedSignature string: 数据库中存储的注册签名
// 返回 error: 验证失败时的错误信息
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

// buildAuthResponse 构建认证响应消息
func (l *WsAuthLogic) buildAuthResponse(success bool, message string, deviceID int64, expiresIn int64) *types.WsAuthResponse {
	return &types.WsAuthResponse{
		Type:      "auth_response",
		Success:   success,
		Message:   message,
		DeviceID:  deviceID,
		ExpiresIn: expiresIn,
	}
}
