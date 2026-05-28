package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
)

const (
	DiagnosisCacheKeyPrefix = "diagnosis:pending:" // Redis Key前缀
	DiagnosisCacheTTL       = 24 * time.Hour       // 缓存过期时间：24小时
	MaxCachedCommands       = 100                  // 每台设备最大缓存指令数
)

// DiagnosisCommandCache 诊断指令缓存管理器
// 用于存储设备离线时的诊断指令，待设备上线后自动下发
type DiagnosisCommandCache struct {
	svcCtx      *svc.ServiceContext
	mu          sync.RWMutex
	memoryCache map[string][]*types.DiagnosisCommandDevicePayload // 内存缓存（Redis不可用时降级）
}

var (
	globalDiagnosisCache *DiagnosisCommandCache
	cacheOnce            sync.Once
)

// GetDiagnosisCache 获取全局诊断指令缓存实例（单例模式）
func GetDiagnosisCache(svcCtx *svc.ServiceContext) *DiagnosisCommandCache {
	cacheOnce.Do(func() {
		globalDiagnosisCache = &DiagnosisCommandCache{
			svcCtx:      svcCtx,
			memoryCache: make(map[string][]*types.DiagnosisCommandDevicePayload),
		}
		logx.Infof("[DiagnosisCache] 初始化诊断指令缓存管理器")
	})
	return globalDiagnosisCache
}

// CacheCommand 缓存诊断指令（设备离线时调用）
// 参数 deviceSN string: 设备序列号
// 参数 command *types.DiagnosisCommandDevicePayload: 诊断指令载荷
// 返回 error: 缓存失败时的错误信息
func (c *DiagnosisCommandCache) CacheCommand(deviceSN string, command *types.DiagnosisCommandDevicePayload) error {
	deviceSN = strings.ToUpper(strings.TrimSpace(deviceSN))
	if deviceSN == "" || command == nil {
		return fmt.Errorf("设备序列号或指令不能为空")
	}

	cacheKey := DiagnosisCacheKeyPrefix + deviceSN

	logx.Infof("[DiagnosisCache] 缓存诊断指令: device_sn=%s, trace_id=%s", deviceSN, command.TraceID)

	if c.svcCtx.Redis != nil {
		err := c.cacheToRedis(cacheKey, command)
		if err != nil {
			logx.Errorf("[DiagnosisCache] Redis缓存失败，降级到内存: device_sn=%s, err=%v", deviceSN, err)
			return c.cacheToMemory(deviceSN, command)
		}
		logx.Infof("[DiagnosisCache] ✅ Redis缓存成功: device_sn=%s, trace_id=%s", deviceSN, command.TraceID)
		return nil
	}

	return c.cacheToMemory(deviceSN, command)
}

// cacheToRedis 将指令缓存到Redis（使用List结构，支持FIFO）
func (c *DiagnosisCommandCache) cacheToRedis(cacheKey string, command *types.DiagnosisCommandDevicePayload) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmdJSON, err := json.Marshal(command)
	if err != nil {
		return fmt.Errorf("指令序列化失败: %w", err)
	}

	pipe := c.svcCtx.Redis.Pipeline()

	lpushCmd := pipe.LPush(ctx, cacheKey, cmdJSON)
	pipe.LTrim(ctx, cacheKey, 0, MaxCachedCommands-1)
	pipe.Expire(ctx, cacheKey, DiagnosisCacheTTL)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("Redis管道执行失败: %w", err)
	}

	if lpushCmd.Err() != nil {
		return fmt.Errorf("Redis LPush失败: %w", lpushCmd.Err())
	}

	logx.Debugf("[DiagnosisCache] Redis LPush成功: key=%s, length=%d", cacheKey, lpushCmd.Val())

	return nil
}

// cacheToMemory 将指令缓存到内存（Redis不可用时的降级方案）
func (c *DiagnosisCommandCache) cacheToMemory(deviceSN string, command *types.DiagnosisCommandDevicePayload) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	commands := c.memoryCache[deviceSN]
	if len(commands) >= MaxCachedCommands {
		commands = commands[1:] // 移除最旧的指令
	}
	commands = append(commands, command)
	c.memoryCache[deviceSN] = commands

	logx.Infof("[DiagnosisCache] ✅ 内存缓存成功: device_sn=%s, trace_id=%s, cached_count=%d",
		deviceSN, command.TraceID, len(commands))
	return nil
}

// FlushCachedCommands 获取并清除设备的所有缓存指令（设备上线时调用）
// 参数 deviceSN string: 设备序列号
// 返回 []*types.DiagnosisCommandDevicePayload: 缓存的指令列表
// 返回 error: 获取失败时的错误信息
func (c *DiagnosisCommandCache) FlushCachedCommands(deviceSN string) ([]*types.DiagnosisCommandDevicePayload, error) {
	deviceSN = strings.ToUpper(strings.TrimSpace(deviceSN))
	if deviceSN == "" {
		return nil, fmt.Errorf("设备序列号不能为空")
	}

	cacheKey := DiagnosisCacheKeyPrefix + deviceSN
	var commands []*types.DiagnosisCommandDevicePayload

	if c.svcCtx.Redis != nil {
		redisCmds, err := c.flushFromRedis(cacheKey)
		if err != nil {
			logx.Errorf("[DiagnosisCache] Redis获取失败，尝试内存: device_sn=%s, err=%v", deviceSN, err)
			memoryCmds := c.flushFromMemory(deviceSN)
			commands = append(commands, memoryCmds...)
		} else {
			commands = append(commands, redisCmds...)
		}
	} else {
		memoryCmds := c.flushFromMemory(deviceSN)
		commands = append(commands, memoryCmds...)
	}

	if len(commands) > 0 {
		logx.Infof("[DiagnosisCache] 📤 获取到 %d 条缓存指令: device_sn=%s", len(commands), deviceSN)
		for i, cmd := range commands {
			logx.Infof("[DiagnosisCache]    [%d] trace_id=%s, cmd_action=%s", i+1, cmd.TraceID, cmd.CmdAction)
		}
	} else {
		logx.Infof("[DiagnosisCache] 无缓存指令: device_sn=%s", deviceSN)
	}

	return commands, nil
}

// flushFromRedis 从Redis获取并删除缓存指令
func (c *DiagnosisCommandCache) flushFromRedis(cacheKey string) ([]*types.DiagnosisCommandDevicePayload, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pipe := c.svcCtx.Redis.Pipeline()
	lrangeCmd := pipe.LRange(ctx, cacheKey, 0, -1)
	delCmd := pipe.Del(ctx, cacheKey)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("Redis管道执行失败: %w", err)
	}

	if delCmd.Val() > 0 {
		logx.Infof("[DiagnosisCache] 已从Redis清除缓存: key=%s, deleted_count=%d", cacheKey, delCmd.Val())
	}

	cmdStrs := lrangeCmd.Val()
	commands := make([]*types.DiagnosisCommandDevicePayload, 0, len(cmdStrs))

	for _, cmdStr := range cmdStrs {
		var cmd types.DiagnosisCommandDevicePayload
		if err := json.Unmarshal([]byte(cmdStr), &cmd); err != nil {
			logx.Errorf("[DiagnosisCache] 指令反序列化失败: %v", err)
			continue
		}
		commands = append(commands, &cmd)
	}

	return commands, nil
}

// flushFromMemory 从内存获取并删除缓存指令
func (c *DiagnosisCommandCache) flushFromMemory(deviceSN string) []*types.DiagnosisCommandDevicePayload {
	c.mu.Lock()
	defer c.mu.Unlock()

	commands := c.memoryCache[deviceSN]
	delete(c.memoryCache, deviceSN)

	if len(commands) > 0 {
		logx.Infof("[DiagnosisCache] 已从内存清除缓存: device_sn=%s, count=%d", deviceSN, len(commands))
	}

	return commands
}

// GetCachedCount 获取设备的缓存指令数量（用于监控和调试）
func (c *DiagnosisCommandCache) GetCachedCount(deviceSN string) int {
	deviceSN = strings.ToUpper(strings.TrimSpace(deviceSN))
	cacheKey := DiagnosisCacheKeyPrefix + deviceSN

	if c.svcCtx.Redis != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		count, err := c.svcCtx.Redis.LLen(ctx, cacheKey).Result()
		if err != nil && err != redis.Nil {
			logx.Errorf("[DiagnosisCache] Redis获取数量失败: device_sn=%s, err=%v", deviceSN, err)
		} else {
			return int(count)
		}
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.memoryCache[deviceSN])
}

// ClearCache 清除指定设备的所有缓存指令（管理员手动清理时使用）
func (c *DiagnosisCommandCache) ClearCache(deviceSN string) error {
	deviceSN = strings.ToUpper(strings.TrimSpace(deviceSN))
	cacheKey := DiagnosisCacheKeyPrefix + deviceSN

	if c.svcCtx.Redis != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		deleted, err := c.svcCtx.Redis.Del(ctx, cacheKey).Result()
		if err != nil {
			return fmt.Errorf("Redis删除失败: %w", err)
		}
		if deleted > 0 {
			logx.Infof("[DiagnosisCache] 已手动清除Redis缓存: device_sn=%s, count=%d", deviceSN, deleted)
		}
	}

	c.mu.Lock()
	if _, ok := c.memoryCache[deviceSN]; ok {
		delete(c.memoryCache, deviceSN)
		logx.Infof("[DiagnosisCache] 已手动清除内存缓存: device_sn=%s", deviceSN)
	}
	c.mu.Unlock()

	return nil
}
