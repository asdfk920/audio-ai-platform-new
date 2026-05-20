package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	deviceCmdCachePrefix   = "device:cmd:cache:"
	deviceCmdCacheTTL      = 24 * time.Hour
	maxCachedCmdsPerDevice = 50
)

// CachedCommand 离线缓存的指令结构
type CachedCommand struct {
	Action        string                 `json:"action"`
	Cmd           string                 `json:"cmd"`
	Params        map[string]interface{} `json:"params"`
	Timestamp     int64                  `json:"timestamp"`
	InstructionID int64                  `json:"instruction_id"`
	UserID        int64                  `json:"user_id"`
	IssuedAt      string                 `json:"issued_at"`
}

// DeviceCmdCacheManager 设备指令缓存管理器
type DeviceCmdCacheManager struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewDeviceCmdCacheManager 创建设备指令缓存管理器实例
func NewDeviceCmdCacheManager(ctx context.Context, svcCtx *svc.ServiceContext) *DeviceCmdCacheManager {
	return &DeviceCmdCacheManager{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// CacheCommand 缓存离线指令到Redis
func (m *DeviceCmdCacheManager) CacheCommand(sn string, cmd *CachedCommand) error {
	if m.svcCtx.Redis == nil {
		logx.Infof("[DeviceCmdCache] Redis未配置，无法缓存指令: sn=%s, action=%s", sn, cmd.Action)
		return fmt.Errorf("Redis未配置")
	}

	sn = strings.ToUpper(strings.TrimSpace(sn))
	cacheKey := m.buildCacheKey(sn)

	logx.Infof("\n[DeviceCmdCache] 开始缓存离线指令...")
	logx.Infof("   设备SN:     %s", sn)
	logx.Infof("   指令类型:    %s", cmd.Action)
	logx.Infof("   指令ID:      %d", cmd.InstructionID)
	logx.Infof("   下发时间:    %s", cmd.IssuedAt)

	existingCmds, err := m.getCachedCommands(sn)
	if err != nil {
		logx.Errorf("[DeviceCmdCache] 读取现有缓存失败: %v", err)
		existingCmds = []*CachedCommand{}
	}

	if len(existingCmds) >= maxCachedCmdsPerDevice {
		logx.Infof("[DeviceCmdCache] 设备 %s 已达最大缓存数(%d)，将丢弃最旧的指令", sn, maxCachedCmdsPerDevice)
		existingCmds = existingCmds[1:]
	}

	existingCmds = append(existingCmds, cmd)

	cmdData, jsonErr := json.Marshal(existingCmds)
	if jsonErr != nil {
		logx.Errorf("[DeviceCmdCache] JSON序列化失败: %v", jsonErr)
		return fmt.Errorf("序列化指令失败: %v", jsonErr)
	}

	setErr := m.svcCtx.Redis.Set(m.ctx, cacheKey, string(cmdData), deviceCmdCacheTTL).Err()
	if setErr != nil {
		logx.Errorf("[DeviceCmdCache] 写入Redis失败: %v", setErr)
		return fmt.Errorf("写入缓存失败: %v", setErr)
	}

	currentCount := len(existingCmds)
	logx.Infof("[DeviceCmdCache] 指令缓存成功!")
	logx.Infof("   缓存Key:    %s", cacheKey)
	logx.Infof("   当前缓存数:  %d / %d", currentCount, maxCachedCmdsPerDevice)
	logx.Infof("   有效期:      %.1f 小时", deviceCmdCacheTTL.Hours())
	logx.Infof("   过期时间:    %s", time.Now().Add(deviceCmdCacheTTL).Format(time.RFC3339))

	return nil
}

// GetCachedCommands 获取设备的所有缓存指令
func (m *DeviceCmdCacheManager) GetCachedCommands(sn string) ([]*CachedCommand, error) {
	sn = strings.ToUpper(strings.TrimSpace(sn))
	return m.getCachedCommands(sn)
}

// ClearCachedCommands 清空设备的所有缓存指令
func (m *DeviceCmdCacheManager) ClearCachedCommands(sn string) error {
	if m.svcCtx.Redis == nil {
		return nil
	}

	sn = strings.ToUpper(strings.TrimSpace(sn))
	cacheKey := m.buildCacheKey(sn)

	delErr := m.svcCtx.Redis.Del(m.ctx, cacheKey).Err()
	if delErr != nil {
		logx.Errorf("[DeviceCmdCache] 清除缓存失败: %v", delErr)
		return fmt.Errorf("清除缓存失败: %v", delErr)
	}

	logx.Infof("[DeviceCmdCache] 已清空设备 %s 的所有缓存指令", sn)
	return nil
}

// RemoveSingleCachedCommand 移除单条已发送的缓存指令
func (m *DeviceCmdCacheManager) RemoveSingleCachedCommand(sn string, instructionID int64) error {
	if m.svcCtx.Redis == nil {
		return nil
	}

	sn = strings.ToUpper(strings.TrimSpace(sn))
	cacheKey := m.buildCacheKey(sn)

	cachedCmds, err := m.getCachedCommands(sn)
	if err != nil || len(cachedCmds) == 0 {
		return err
	}

	filteredCmds := make([]*CachedCommand, 0, len(cachedCmds))
	for _, cmd := range cachedCmds {
		if cmd.InstructionID != instructionID {
			filteredCmds = append(filteredCmds, cmd)
		}
	}

	if len(filteredCmds) == 0 {
		return m.ClearCachedCommands(sn)
	}

	cmdData, jsonErr := json.Marshal(filteredCmds)
	if jsonErr != nil {
		return fmt.Errorf("序列化失败: %v", jsonErr)
	}

	setErr := m.svcCtx.Redis.Set(m.ctx, cacheKey, string(cmdData), deviceCmdCacheTTL).Err()
	if setErr != nil {
		return fmt.Errorf("更新缓存失败: %v", setErr)
	}

	logx.Infof("[DeviceCmdCache] 已移除单条缓存指令: sn=%s, instruction_id=%d", sn, instructionID)
	return nil
}

// HasCachedCommands 检查设备是否有缓存指令
func (m *DeviceCmdCacheManager) HasCachedCommands(sn string) bool {
	if m.svcCtx.Redis == nil {
		return false
	}

	sn = strings.ToUpper(strings.TrimSpace(sn))
	cacheKey := m.buildCacheKey(sn)

	data, err := m.svcCtx.Redis.Get(m.ctx, cacheKey).Result()
	if err != nil || data == "" {
		return false
	}
	return true
}

// GetCachedCommandCount 获取缓存指令数量
func (m *DeviceCmdCacheManager) GetCachedCommandCount(sn string) int64 {
	if m.svcCtx.Redis == nil {
		return 0
	}

	sn = strings.ToUpper(strings.TrimSpace(sn))
	cachedCmds, _ := m.getCachedCommands(sn)
	return int64(len(cachedCmds))
}

// buildCacheKey 构建Redis缓存Key
func (m *DeviceCmdCacheManager) buildCacheKey(sn string) string {
	return deviceCmdCachePrefix + strings.ToUpper(strings.TrimSpace(sn))
}

// getCachedCommands 内部方法：从Redis获取并解析缓存指令列表（自动过滤过期数据）
func (m *DeviceCmdCacheManager) getCachedCommands(sn string) ([]*CachedCommand, error) {
	if m.svcCtx.Redis == nil {
		return nil, fmt.Errorf("Redis未配置")
	}

	sn = strings.ToUpper(strings.TrimSpace(sn))
	cacheKey := m.buildCacheKey(sn)

	data, err := m.svcCtx.Redis.Get(m.ctx, cacheKey).Result()
	if err != nil {
		return nil, nil
	}

	var cachedCmds []*CachedCommand
	unmarshalErr := json.Unmarshal([]byte(data), &cachedCmds)
	if unmarshalErr != nil {
		logx.Errorf("[DeviceCmdCache] JSON解析缓存数据失败: %v", unmarshalErr)
		return nil, fmt.Errorf("解析缓存数据失败: %v", unmarshalErr)
	}

	now := time.Now().UnixMilli()
	validCmds := make([]*CachedCommand, 0, len(cachedCmds))
	for _, cmd := range cachedCmds {
		expireTime := cmd.Timestamp + int64(deviceCmdCacheTTL.Milliseconds())
		if now <= expireTime {
			validCmds = append(validCmds, cmd)
		} else {
			logx.Infof("[DeviceCmdCache] 发现过期缓存指令: sn=%s, instruction_id=%d, 已丢弃",
				sn, cmd.InstructionID)
		}
	}

	if len(validCmds) != len(cachedCmds) && len(validCmds) > 0 {
		logx.Infof("[DeviceCmdCache] 清理了 %d 条过期缓存指令，剩余 %d 条有效指令",
			len(cachedCmds)-len(validCmds), len(validCmds))

		updatedData, marshalErr := json.Marshal(validCmds)
		if marshalErr == nil {
			m.svcCtx.Redis.Set(m.ctx, cacheKey, string(updatedData), deviceCmdCacheTTL)
		}
	} else if len(validCmds) == 0 && len(cachedCmds) > 0 {
		m.ClearCachedCommands(sn)
	}

	return validCmds, nil
}

// CleanupExpiredCachedCommands 清理所有设备的过期缓存指令
func (m *DeviceCmdCacheManager) CleanupExpiredCachedCommands() (int64, error) {
	if m.svcCtx.Redis == nil {
		return 0, fmt.Errorf("Redis未配置")
	}

	logx.Infof("\n[DeviceCmdCache] 开始清理过期缓存指令...")

	var cursor uint64
	var cleanedCount int64

	for {
		keys, nextCursor, scanErr := m.svcCtx.Redis.Scan(m.ctx, cursor, deviceCmdCachePrefix+"*", 100).Result()
		if scanErr != nil {
			logx.Errorf("[DeviceCmdCache] 扫描缓存键失败: %v", scanErr)
			break
		}

		cursor = nextCursor

		for _, key := range keys {
			data, getErr := m.svcCtx.Redis.Get(m.ctx, key).Result()
			if getErr != nil {
				continue
			}

			var cachedCmds []*CachedCommand
			unmarshalErr := json.Unmarshal([]byte(data), &cachedCmds)
			if unmarshalErr != nil {
				m.svcCtx.Redis.Del(m.ctx, key)
				cleanedCount++
				continue
			}

			now := time.Now().UnixMilli()
			validCmds := make([]*CachedCommand, 0)

			for _, cmd := range cachedCmds {
				expireTime := cmd.Timestamp + int64(deviceCmdCacheTTL.Milliseconds())
				if now <= expireTime {
					validCmds = append(validCmds, cmd)
				}
			}

			if len(validCmds) == 0 {
				delErr := m.svcCtx.Redis.Del(m.ctx, key).Err()
				if delErr == nil {
					cleanedCount++
					sn := strings.TrimPrefix(key, deviceCmdCachePrefix)
					logx.Infof("[DeviceCmdCache] 已清空设备 %s 的全部过期缓存", sn)
				}
			} else if len(validCmds) < len(cachedCmds) {
				updatedData, _ := json.Marshal(validCmds)
				m.svcCtx.Redis.Set(m.ctx, key, string(updatedData), deviceCmdCacheTTL)
				sn := strings.TrimPrefix(key, deviceCmdCachePrefix)
				logx.Infof("[DeviceCmdCache] 设备 %s 清理了 %d 条过期指令，保留 %d 条",
					sn, len(cachedCmds)-len(validCmds), len(validCmds))
			}
		}

		if cursor == 0 {
			break
		}
	}

	logx.Infof("[DeviceCmdCache] 过期缓存清理完成，共处理 %d 个设备", cleanedCount)
	return cleanedCount, nil
}
