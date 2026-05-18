package logic

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/crypto/bcrypt"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/device/internal/deviceauthsvc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/model"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
	"github.com/jacklau/audio-ai-platform/services/device/internal/util"
)

// DeviceRegisterLogic 设备注册逻辑
// 处理设备首次注册的业务逻辑（基于SN+时间戳+签名的安全验证机制）
type DeviceRegisterLogic struct {
	ctx               context.Context
	svcCtx            *svc.ServiceContext
	deviceAuthService *deviceauthsvc.Service
}

// NewDeviceRegisterLogic 创建设备注册逻辑实例
func NewDeviceRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeviceRegisterLogic {
	return &DeviceRegisterLogic{
		ctx:               ctx,
		svcCtx:            svcCtx,
		deviceAuthService: deviceauthsvc.New(svcCtx),
	}
}

// DeviceRegister 设备安全注册（简化版：只需 SN + 设备密钥）
//
// 完整安全注册流程：
//
//  1. **参数校验**：校验SN格式、设备密钥非空
//  2. **自动生成**：服务端自动生成当前时间戳
//  3. **查询预存密钥**：根据SN从数据库查询生产阶段预存的设备密钥
//  4. **密钥比对**：比对请求中的密钥与数据库中存储的密钥是否一致
//  5. **状态检查**：判断设备是否已注册，防止重复注册
//  6. **生成凭证**：为设备生成JWT访问凭证（Token）
//  7. **更新状态**：标记设备为已注册状态，记录注册时间
//  8. **返回结果**：返回Token、有效期等认证信息给设备端保存
//
// 安全特性：
//   - 服务端自动管理时间戳，防止客户端篡改
//   - 密钥比对确保设备身份合法性
//   - JWT Token支持自动过期，可定期刷新
//   - 数据库唯一索引保证SN全局唯一，防止重复注册
//
// 参数 req *types.DeviceRegisterReq: 设备注册请求（包含sn、device_secret）
// 返回 *types.DeviceRegisterResp: 设备注册响应（包含token、expires_in等）
// 返回 error: 注册失败时的错误信息
func (l *DeviceRegisterLogic) DeviceRegister(req *types.DeviceRegisterReq) (*types.DeviceRegisterResp, error) {
	sn := strings.TrimSpace(strings.ToUpper(req.Sn))
	deviceSecret := strings.TrimSpace(req.DeviceSecret)

	logx.Infof("收到设备注册请求: sn=%s (显示格式:%s)", sn, util.FormatSNDisplay(sn))

	if err := l.validateRequestParams(sn, deviceSecret); err != nil {
		return nil, err
	}

	device, err := l.svcCtx.DeviceRepo.FindBySn(l.ctx, sn)
	if err != nil {
		logx.Errorf("查询设备失败: sn=%s, err=%v", sn, err)
		return nil, errorx.NewDefaultError(errorx.CodeSystemError)
	}

	if device == nil || device.DeviceSecret == "" {
		logx.Slow("设备不存在或未预置密钥: sn=%s", sn)
		return nil, errorx.NewDefaultError(errorx.CodeDeviceNotFound)
	}

	if !verifyDeviceSecretAgainstStored(deviceSecret, device.DeviceSecret) {
		logx.Slow("设备密钥不匹配: sn=%s", sn)
		return nil, errorx.NewDefaultError(errorx.CodeDeviceSecretInvalid)
	}

	if device.Status == 1 {
		logx.Infof("设备已注册，重新生成凭证: sn=%s, device_id=%d", sn, device.ID)
		return l.generateNewCredential(device, deviceSecret)
	}

	token, expiresIn, err := l.deviceAuthService.IssueDeviceToken(&deviceauthsvc.Principal{
		DeviceID:        device.ID,
		DeviceSN:        device.Sn,
		ProductKey:      device.ProductKey,
		Mac:             device.Mac,
		FirmwareVersion: device.FirmwareVersion,
		IP:              device.Ip,
		Status:          device.Status,
	})
	if err != nil {
		logx.Errorf("生成访问凭证失败: sn=%s, err=%v", sn, err)
		return nil, errorx.NewDefaultError(errorx.CodeSystemError)
	}

	now := time.Now()
	timestamp := now.UnixMilli()

	signature := l.generateRegisterSignature(sn, timestamp, deviceSecret)

	if err := l.svcCtx.DeviceRepo.UpdateRegisterSignature(l.ctx, device.ID, signature); err != nil {
		logx.Errorf("保存注册签名失败: sn=%s, err=%v", sn, err)
	}

	if err := l.svcCtx.DeviceRepo.UpdateStatusAfterRegister(l.ctx, device.ID, 1); err != nil {
		logx.Errorf("更新设备注册状态失败: sn=%s, err=%v", sn, err)
	}

	snDisplay := util.FormatSNDisplay(sn)
	logx.Infof("设备注册成功: sn=%s (显示格式:%s), device_id=%d, token_expires=%ds, signature=%s",
		sn, snDisplay, device.ID, expiresIn, signature[:16]+"...")

	return &types.DeviceRegisterResp{
		Sn:                sn,
		Token:             token,
		ExpiresIn:         expiresIn,
		DeviceID:          device.ID,
		RegisterTime:      now.Format(time.RFC3339),
		RegisterTimestamp: timestamp,
		Signature:         signature,
	}, nil
}

// validateRequestParams 校验请求参数的完整性和格式
func (l *DeviceRegisterLogic) validateRequestParams(sn string, deviceSecret string) error {
	if err := validateSnFormat(sn); err != nil {
		return errorx.NewCodeError(errorx.CodeDeviceSnInvalid, err.Error())
	}

	if deviceSecret == "" {
		return errorx.NewCodeError(errorx.CodeInvalidParam, "设备密钥不能为空")
	}

	return nil
}

// verifyDeviceSecretAgainstStored 与 deviceauthsvc 一致：库存 bcrypt 哈希时用 CompareHashAndPassword；否则按明文常量时间比对（兼容旧数据）。
func verifyDeviceSecretAgainstStored(requestPlaintext, stored string) bool {
	requestPlaintext = strings.TrimSpace(requestPlaintext)
	stored = strings.TrimSpace(stored)
	if requestPlaintext == "" || stored == "" {
		return false
	}
	if isBcryptSecret(stored) {
		return bcrypt.CompareHashAndPassword([]byte(stored), []byte(requestPlaintext)) == nil
	}
	if len(requestPlaintext) != len(stored) {
		return false
	}
	result := 0
	for i := 0; i < len(requestPlaintext); i++ {
		result |= int(requestPlaintext[i]) ^ int(stored[i])
	}
	return result == 0
}

func isBcryptSecret(s string) bool {
	return strings.HasPrefix(s, "$2a$") || strings.HasPrefix(s, "$2b$") || strings.HasPrefix(s, "$2y$")
}

// generateRegisterSignature 生成设备注册签名
// 签名算法：HMAC-SHA256(device_secret, sn + timestamp)
// 该签名将存储在数据库中，用于后续的 WebSocket 认证验证
//
// 参数 sn string: 设备序列号
// 参数 timestamp int64: 注册时间戳（毫秒级）
// 参数 deviceSecret string: 设备密钥（生产预烧录）
// 返回 string: 64位十六进制签名字符串
func (l *DeviceRegisterLogic) generateRegisterSignature(sn string, timestamp int64, deviceSecret string) string {
	signData := fmt.Sprintf("%s%d", sn, timestamp)

	mac := hmac.New(sha256.New, []byte(strings.TrimSpace(deviceSecret)))
	mac.Write([]byte(signData))
	signature := hex.EncodeToString(mac.Sum(nil))

	return signature
}

// generateNewCredential 为已注册设备重新生成凭证（包含签名更新）
// plainSecret 为本次请求中的设备密钥明文；HMAC 必须与设备侧约定一致（不能用库里的 bcrypt 哈希当 key）。
func (l *DeviceRegisterLogic) generateNewCredential(device *model.Device, plainSecret string) (*types.DeviceRegisterResp, error) {
	token, expiresIn, err := l.deviceAuthService.IssueDeviceToken(&deviceauthsvc.Principal{
		DeviceID:        device.ID,
		DeviceSN:        device.Sn,
		ProductKey:      device.ProductKey,
		Mac:             device.Mac,
		FirmwareVersion: device.FirmwareVersion,
		IP:              device.Ip,
		Status:          device.Status,
	})
	if err != nil {
		return nil, errorx.NewDefaultError(errorx.CodeSystemError)
	}

	now := time.Now()
	timestamp := now.UnixMilli()

	signature := l.generateRegisterSignature(device.Sn, timestamp, plainSecret)

	if err := l.svcCtx.DeviceRepo.UpdateRegisterSignature(l.ctx, device.ID, signature); err != nil {
		logx.Errorf("更新注册签名失败(重注册): sn=%s, err=%v", device.Sn, err)
	}

	snDisplay := util.FormatSNDisplay(device.Sn)
	logx.Infof("设备重新注册成功: sn=%s (显示格式:%s), device_id=%d, token_expires=%ds",
		device.Sn, snDisplay, device.ID, expiresIn)

	return &types.DeviceRegisterResp{
		Sn:                device.Sn,
		Token:             token,
		ExpiresIn:         expiresIn,
		DeviceID:          device.ID,
		RegisterTime:      now.Format(time.RFC3339),
		RegisterTimestamp: timestamp,
		Signature:         signature,
	}, nil
}

// validateSnFormat 校验设备序列号格式
func validateSnFormat(sn string) error {
	if sn == "" {
		return fmt.Errorf("设备序列号不能为空")
	}

	if !util.ValidateSNFormat(sn) {
		parsed := util.ParseSN(sn)
		if errMsg, ok := parsed["error"].(string); ok {
			return fmt.Errorf("%s", errMsg)
		}
		return fmt.Errorf("SN格式错误或校验码无效")
	}

	return nil
}
