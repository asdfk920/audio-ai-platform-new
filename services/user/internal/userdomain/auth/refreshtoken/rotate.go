package refreshtoken

import (
	"context"
	"strconv"

	"github.com/jacklau/audio-ai-platform/pkg/redisx"
)

// Lua 原子轮换：校验 refresh 映射与 user 索引一致后，删除旧 refresh 键、写入新 refresh 键并更新索引，避免中间态丢会话。
const rotateScript = `
local uid = redis.call('GET', KEYS[1])
if uid == false or uid ~= ARGV[1] then
  return 0
end
local bound = redis.call('GET', KEYS[2])
if bound == false or bound ~= ARGV[2] then
  return 1
end
local ttl = tonumber(ARGV[4])
if ttl == nil or ttl <= 0 then
  return 3
end
redis.call('DEL', KEYS[1])
redis.call('SET', KEYS[3], ARGV[1], 'EX', ttl)
redis.call('SET', KEYS[2], ARGV[3], 'EX', ttl)
return 2
`

const (
	RotateInvalidOld    = 0 // refresh 键不存在或 userId 不匹配
	RotateNotCurrent    = 1 // 与 user 索引不一致（盗用旧 token / 已顶号）
	RotateOK            = 2
	RotateBadTTL        = 3
	RotateResultUnknown = -1
)

// AtomicRotate 在同一 Redis 原子脚本内完成校验与轮换。
func AtomicRotate(ctx context.Context, oldToken, newToken string, userID int64, ttlSec int64) (int, error) {
	if ttlSec <= 0 {
		return RotateBadTTL, nil
	}
	keys := []string{
		KeyRefresh(oldToken),
		KeyUserIndex(userID),
		KeyRefresh(newToken),
	}
	res, err := redisx.Eval(ctx, rotateScript, keys,
		strconv.FormatInt(userID, 10),
		oldToken,
		newToken,
		strconv.FormatInt(ttlSec, 10),
	)
	if err != nil {
		return RotateResultUnknown, err
	}
	return parseIntResult(res), nil
}

func parseIntResult(v interface{}) int {
	switch x := v.(type) {
	case int64:
		return int(x)
	case int:
		return x
	default:
		return RotateResultUnknown
	}
}
