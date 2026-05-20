package logic

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/crypto/bcrypt"

	"github.com/jacklau/audio-ai-platform/services/device/internal/deviceauthsvc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/model"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
)

const (
	deviceRegisterTokenExpireSeconds = 86400
)

// ErrDeviceAlreadyRegistered 设备 SN 已在库（未删除）且密钥正确时再调用注册时使用。
var ErrDeviceAlreadyRegistered = errors.New("该设备已注册，请勿重复注册")

type DeviceRegisterLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeviceRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeviceRegisterLogic {
	return &DeviceRegisterLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// DeviceRegister 设备首次注册：创建设备并返回 token/签名字段；SN 对已存在活跃设备时报 ErrDeviceAlreadyRegistered。
// 仅软删除占位时：校验密钥后可恢复档案并签发凭证（等价于该机首次在云侧生效）。
func (l *DeviceRegisterLogic) DeviceRegister(req *types.DeviceRegisterReq) (*types.DeviceRegisterResp, error) {
	logx.Infof("====================================")
	logx.Infof("[Device Register] 📝 开始设备注册流程...")
	sn := normalizeSN(req.Sn)
	logx.Infof("[Device Register] 📋 设备SN: %s", sn)

	if err := l.validateRegisterRequest(sn, req.DeviceSecret); err != nil {
		return nil, fmt.Errorf("请求参数校验失败: %v", err)
	}

	active, err := l.svcCtx.DeviceRepo.FindBySn(l.ctx, sn)
	if err != nil {
		return nil, fmt.Errorf("查询设备失败: %w", err)
	}
	if active != nil && active.ID > 0 {
		if !verifyDeviceSecretAgainstStored(req.DeviceSecret, active.DeviceSecret) {
			return nil, fmt.Errorf("设备密钥不正确")
		}
		return nil, ErrDeviceAlreadyRegistered
	}

	ghost, err := l.svcCtx.DeviceRepo.FindBySnIncludingDeleted(l.ctx, sn)
	if err != nil {
		return nil, fmt.Errorf("查询设备失败: %w", err)
	}
	if ghost != nil {
		if !verifyDeviceSecretAgainstStored(req.DeviceSecret, ghost.DeviceSecret) {
			return nil, fmt.Errorf("设备密钥不正确")
		}
		if ghost.DeletedAt != nil {
			if err := l.svcCtx.DeviceRepo.ClearDeletedAt(l.ctx, ghost.ID); err != nil {
				return nil, err
			}
			logx.Infof("[Device Register] 已恢复软删除设备: id=%d sn=%s", ghost.ID, sn)
			existing, err := l.svcCtx.DeviceRepo.FindBySn(l.ctx, sn)
			if err != nil {
				return nil, fmt.Errorf("查询设备失败: %w", err)
			}
			if existing == nil {
				return nil, fmt.Errorf("恢复设备后仍无法加载记录，请稍后重试")
			}
			return l.refreshRegistrationForExistingDevice(existing, sn, req.DeviceSecret)
		}
		logx.Errorf("[Device Register] 数据异常 ghost 活跃但 FindBySn 为空 sn=%s id=%d", sn, ghost.ID)
		return nil, ErrDeviceAlreadyRegistered
	}

	logx.Infof("[Device Register] 🔍 新设备，执行首次入库...")

	now := time.Now()
	registerTimestamp := now.UnixMilli()

	hashedSecret, err := bcrypt.GenerateFromPassword([]byte(strings.TrimSpace(req.DeviceSecret)), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("加密设备密钥失败: %v", err)
	}

	signature := calculateHMACSignatureHex(sn, registerTimestamp, req.DeviceSecret)

	productKey := fmt.Sprintf("PK_%s", sn)

	newDevice := &model.Device{
		Sn:                sn,
		Model:             "",
		ProductKey:        productKey,
		DeviceSecret:      string(hashedSecret),
		RegisterSignature: &signature,
		RegisterTimestamp: &registerTimestamp,
		FirmwareVersion:   "",
		HardwareVersion:   "",
		Mac:               "",
		Ip:                "",
		OnlineStatus:      model.DeviceOnlineStatusOffline,
		UsageStatus:       model.DeviceUsageStatusEnabled,
		Status:            model.DeviceStatusNormal,
		CreateBy:          0,
		LastActiveAt:      now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	deviceID, createErr := l.svcCtx.DeviceRepo.CreateWithFullInfo(l.ctx, newDevice)
	if createErr != nil && isPostgresUniqueViolation(createErr) {
		logx.Infof("[Device Register] INSERT 撞唯一键，按已存在设备处理: sn=%s", sn)
		return l.recoverExistingAfterDuplicateKey(sn, req.DeviceSecret)
	}
	if createErr != nil {
		return nil, fmt.Errorf("创建设备记录失败: %w", createErr)
	}

	logx.Infof("[Device Register] ✅ 设备记录创建成功: device_id=%d", deviceID)

	token, tokenErr := l.generateJWTToken(deviceID, sn)
	if tokenErr != nil {
		return nil, fmt.Errorf("生成访问令牌失败: %v", tokenErr)
	}

	logx.Infof("\n====================================")
	logx.Infof("🎉 [Device Register] ✅✅✅ 设备注册成功! ✅✅✅")
	logx.Infof("====================================")

	return &types.DeviceRegisterResp{
		Sn:                sn,
		Token:             token,
		ExpiresIn:         deviceRegisterTokenExpireSeconds,
		DeviceID:          deviceID,
		RegisterTime:      now.Format(time.RFC3339),
		RegisterTimestamp: registerTimestamp,
		Signature:         signature,
	}, nil
}

func (l *DeviceRegisterLogic) recoverExistingAfterDuplicateKey(sn, plainSecret string) (*types.DeviceRegisterResp, error) {
	active, err := l.svcCtx.DeviceRepo.FindBySn(l.ctx, sn)
	if err != nil {
		return nil, fmt.Errorf("创建设备后与唯一约束冲突，但加载记录失败: %w", err)
	}
	if active != nil && active.ID > 0 {
		if !verifyDeviceSecretAgainstStored(plainSecret, active.DeviceSecret) {
			return nil, fmt.Errorf("设备密钥不正确")
		}
		return nil, ErrDeviceAlreadyRegistered
	}
	row, err := l.svcCtx.DeviceRepo.FindBySnIncludingDeleted(l.ctx, sn)
	if err != nil {
		return nil, fmt.Errorf("创建设备后与唯一约束冲突，但无法加载已有记录: %w", err)
	}
	if row == nil {
		return nil, fmt.Errorf("创建设备失败：SN 唯一约束冲突且未找到对应记录")
	}
	if row.DeletedAt != nil {
		if !verifyDeviceSecretAgainstStored(plainSecret, row.DeviceSecret) {
			return nil, fmt.Errorf("设备密钥不正确")
		}
		if err := l.svcCtx.DeviceRepo.ClearDeletedAt(l.ctx, row.ID); err != nil {
			return nil, err
		}
		active2, err := l.svcCtx.DeviceRepo.FindBySn(l.ctx, sn)
		if err != nil {
			return nil, err
		}
		if active2 == nil {
			return nil, fmt.Errorf("处理唯一约束冲突后仍无法加载设备")
		}
		return l.refreshRegistrationForExistingDevice(active2, sn, plainSecret)
	}
	if !verifyDeviceSecretAgainstStored(plainSecret, row.DeviceSecret) {
		return nil, fmt.Errorf("设备密钥不正确")
	}
	return nil, ErrDeviceAlreadyRegistered
}

// refreshRegistrationForExistingDevice 在「软删除恢复」或唯一键冲突后恢复档案时，校验密钥并签发 JWT 与当前时间戳签名（响应形状与首次注册一致）。
func (l *DeviceRegisterLogic) refreshRegistrationForExistingDevice(device *model.Device, sn, plainSecret string) (*types.DeviceRegisterResp, error) {
	if device == nil || device.ID <= 0 {
		return nil, fmt.Errorf("内部错误：设备数据无效")
	}
	if !verifyDeviceSecretAgainstStored(plainSecret, device.DeviceSecret) {
		return nil, fmt.Errorf("设备密钥不正确")
	}

	now := time.Now()
	ts := now.UnixMilli()
	sig := calculateHMACSignatureHex(sn, ts, plainSecret)

	token, err := l.generateJWTToken(device.ID, sn)
	if err != nil {
		return nil, fmt.Errorf("生成访问令牌失败: %w", err)
	}

	logx.Infof("[Device Register] ✅ 已刷新注册凭证: device_id=%d sn=%s", device.ID, sn)

	return &types.DeviceRegisterResp{
		Sn:                sn,
		Token:             token,
		ExpiresIn:         deviceRegisterTokenExpireSeconds,
		DeviceID:          device.ID,
		RegisterTime:      now.Format(time.RFC3339),
		RegisterTimestamp: ts,
		Signature:         sig,
	}, nil
}

func (l *DeviceRegisterLogic) validateRegisterRequest(sn string, deviceSecret string) error {
	if sn == "" {
		return fmt.Errorf("设备序列号不能为空")
	}

	if len(sn) != 16 {
		return fmt.Errorf("设备序列号长度错误: 期望16位, 实际%d位", len(sn))
	}

	if deviceSecret == "" {
		return fmt.Errorf("设备密钥不能为空")
	}

	if len(strings.TrimSpace(deviceSecret)) < 16 {
		return fmt.Errorf("设备密钥长度不足: 至少需要16位")
	}

	return nil
}

func (l *DeviceRegisterLogic) generateJWTToken(deviceID int64, sn string) (string, error) {
	deviceAuthSvc := deviceauthsvc.New(l.svcCtx)
	principal := &deviceauthsvc.Principal{
		DeviceID: deviceID,
		DeviceSN: sn,
	}
	token, _, tokenErr := deviceAuthSvc.IssueDeviceToken(principal)
	return token, tokenErr
}

func calculateHMACSignatureHex(sn string, timestampMs int64, plainSecret string) string {
	signData := fmt.Sprintf("%s%d", sn, timestampMs)
	mac := hmac.New(sha256.New, []byte(strings.TrimSpace(plainSecret)))
	mac.Write([]byte(signData))
	return hex.EncodeToString(mac.Sum(nil))
}

func verifyDeviceSecretAgainstStored(requestPlaintext, stored string) bool {
	requestPlaintext = strings.TrimSpace(requestPlaintext)
	stored = strings.TrimSpace(stored)
	if requestPlaintext == "" || stored == "" {
		return false
	}
	if strings.HasPrefix(stored, "$2a$") || strings.HasPrefix(stored, "$2b$") || strings.HasPrefix(stored, "$2y$") {
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

func normalizeSN(sn string) string {
	result := make([]byte, 0, len(sn))
	for i := 0; i < len(sn); i++ {
		c := sn[i]
		if c >= 'a' && c <= 'z' {
			result = append(result, c-32)
		} else if (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			result = append(result, c)
		}
	}
	return string(result)
}

func isPostgresUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return true
	}
	return strings.Contains(err.Error(), "23505")
}
