package logic

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/services/device/internal/deviceauthsvc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/model"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
)

const (
	deviceAuthTokenExpireSeconds = 2592000 // 30天有效期（秒）
	deviceAuthRedisKeyPrefix     = "device:auth:token:"
	deviceAuthRedisTTL           = time.Hour * 24 * 31 // Redis缓存31天（略长于Token有效期）
)

type DeviceAuthLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeviceAuthLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeviceAuthLogic {
	return &DeviceAuthLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// DeviceAuth 设备认证接口（密钥模式，获取JWT Token）
//
// 完整流程：
// 1. 接收SN、deviceSecret参数
// 2. 校验参数合法性（非空、格式）
// 3. 根据SN查询数据库设备信息
// 4. 设备不存在 → 返回认证失败
// 5. 校验密钥是否匹配 → 不匹配 → 返回认证失败
// 6. 检查设备状态（已激活、未禁用）→ 状态异常 → 返回认证失败
// 7. 身份验证通过 → 生成JWT Token（30天有效期）
// 8. 将Token存入Redis缓存（用于快速校验和黑名单管理）
// 9. 返回Token + 有效期 + 设备基础信息
func (l *DeviceAuthLogic) DeviceAuth(req *types.DeviceAuthReq) (*types.DeviceAuthResp, error) {
	logx.Infof("====================================")
	logx.Infof("[Device Auth] 🔐 开始设备认证流程...")

	sn := normalizeSN(req.Sn)
	logx.Infof("[Device Auth] 📋 设备SN: %s", sn)

	if err := l.validateAuthRequest(sn, req.DeviceSecret); err != nil {
		return nil, fmt.Errorf("请求参数校验失败: %v", err)
	}

	logx.Infof("[Device Auth] 🔍 查询设备信息...")

	device, err := l.svcCtx.DeviceRepo.FindBySnForActivation(l.ctx, sn)
	if err != nil {
		return nil, fmt.Errorf("查询设备失败: %w", err)
	}

	if device == nil || device.ID == 0 {
		logx.Errorf("[Device Auth] ❌ 设备不存在: sn=%s", sn)
		return nil, fmt.Errorf("认证失败：设备不存在")
	}

	logx.Infof("[Device Auth] ✅ 找到设备: id=%d status=%d", device.ID, device.Status)

	if !verifyDeviceSecretAgainstStored(req.DeviceSecret, device.DeviceSecret) {
		logx.Errorf("[Device Auth] ❌ 密钥不匹配: sn=%s", sn)
		return nil, fmt.Errorf("认证失败：设备密钥不正确")
	}

	logx.Infof("[Device Auth] ✅ 密钥验证通过")

	if err := l.checkDeviceStatus(device); err != nil {
		return nil, err
	}

	logx.Infof("[Device Auth] ✅ 设备状态正常，开始生成访问凭证...")

	token, tokenErr := l.generateAndCacheToken(device)
	if tokenErr != nil {
		return nil, fmt.Errorf("生成访问凭证失败: %v", tokenErr)
	}

	logx.Infof("\n====================================")
	logx.Infof("🎉 [Device Auth] ✅✅✅ 设备认证成功! ✅✅✅")
	logx.Infof("====================================")

	return &types.DeviceAuthResp{
		Token:           token.Token,
		ExpiresIn:       token.ExpiresIn,
		DeviceID:        device.ID,
		Sn:              device.Sn,
		Model:           device.Model,
		FirmwareVersion: device.FirmwareVersion,
		HardwareVersion: device.HardwareVersion,
		OnlineStatus:    device.OnlineStatus,
		Status:          device.Status,
		Message:         "设备认证成功",
	}, nil
}

func (l *DeviceAuthLogic) validateAuthRequest(sn string, deviceSecret string) error {
	if strings.TrimSpace(sn) == "" {
		return fmt.Errorf("设备序列号不能为空")
	}

	sn = strings.TrimSpace(sn)
	if len(sn) != 16 {
		return fmt.Errorf("设备序列号长度错误: 期望16位, 实际%d位", len(sn))
	}

	for _, c := range sn {
		if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z')) {
			return fmt.Errorf("设备序列号格式错误: 只允许字母和数字")
		}
	}

	if strings.TrimSpace(deviceSecret) == "" {
		return fmt.Errorf("设备密钥不能为空")
	}

	if len(strings.TrimSpace(deviceSecret)) < 16 {
		return fmt.Errorf("设备密钥长度不足: 至少需要16位")
	}

	return nil
}

func (l *DeviceAuthLogic) checkDeviceStatus(device *model.Device) error {
	switch device.Status {
	case model.DeviceStatusUnregistered:
		logx.Errorf("[Device Auth] ❌ 设备未激活: sn=%s status=%d", device.Sn, device.Status)
		return fmt.Errorf("认证失败：设备未激活，请先调用注册接口激活设备")

	case model.DeviceStatusDisabled:
		logx.Errorf("[Device Auth] ❌ 设备已被禁用: sn=%s status=%d", device.Sn, device.Status)
		return fmt.Errorf("认证失败：设备已被管理员禁用，请联系客服")

	case model.DeviceStatusInactive:
		logx.Errorf("[Device Auth] ❌ 设备已停用: sn=%s status=%d", device.Sn, device.Status)
		return fmt.Errorf("认证失败：设备已停用或报废，无法使用")

	case model.DeviceStatusNormal:
		logx.Infof("[Device Auth] ✅ 设备状态正常: sn=%s (status=1)", device.Sn)
		return nil

	default:
		logx.Infof("[Device Auth] ⚠️ 设备状态未知: sn=%s status=%d (允许通过)", device.Sn, device.Status)
		return nil
	}
}

type authToken struct {
	Token     string
	ExpiresIn int64
}

func (l *DeviceAuthLogic) generateAndCacheToken(device *model.Device) (*authToken, error) {
	deviceAuthSvc := deviceauthsvc.New(l.svcCtx)
	principal := &deviceauthsvc.Principal{
		DeviceID: device.ID,
		DeviceSN: device.Sn,
	}

	jwtToken, expiresIn, err := deviceAuthSvc.IssueDeviceToken(principal)
	if err != nil {
		return nil, fmt.Errorf("生成JWT Token失败: %w", err)
	}

	logx.Infof("[Device Auth] 🔑 JWT Token生成成功: 长度=%d 有效期=%d秒", len(jwtToken), expiresIn)

	if cacheErr := l.cacheTokenToRedis(device.ID, jwtToken, expiresIn); cacheErr != nil {
		logx.Infof("[Device Auth] ⚠️ Redis缓存Token失败（不影响认证结果）: %v", cacheErr)
	} else {
		logx.Infof("[Device Auth] ✅ Token已存入Redis缓存")
	}

	return &authToken{
		Token:     jwtToken,
		ExpiresIn: expiresIn,
	}, nil
}

func (l *DeviceAuthLogic) cacheTokenToRedis(deviceID int64, token string, expireSeconds int64) error {
	if l.svcCtx.Redis == nil {
		return fmt.Errorf("Redis客户端未初始化")
	}

	key := fmt.Sprintf("%s%d", deviceAuthRedisKeyPrefix, deviceID)
	ttl := time.Duration(expireSeconds) * time.Second

	if ttl < deviceAuthRedisTTL {
		ttl = deviceAuthRedisTTL
	}

	err := l.svcCtx.Redis.Set(l.ctx, key, token, ttl).Err()
	if err != nil {
		return fmt.Errorf("Redis Set操作失败: %w", err)
	}

	return nil
}

func ValidateTokenFromRedis(ctx context.Context, rdb *redis.Client, deviceID int64, token string) (bool, error) {
	if rdb == nil {
		return false, fmt.Errorf("Redis客户端未初始化")
	}

	key := fmt.Sprintf("%s%d", deviceAuthRedisKeyPrefix, deviceID)
	cachedToken, err := rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, fmt.Errorf("Redis Get操作失败: %w", err)
	}

	return cachedToken == token, nil
}
