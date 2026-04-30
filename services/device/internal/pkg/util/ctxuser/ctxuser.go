// Package ctxuser 从 context 读取 JWT 用户 ID（go-zero 中间件写入；无设备业务规则）。
package ctxuser

import (
	"context"
	"encoding/json"
	"strconv"
)

const JWTUserIDKey = "userId"

// ParseUserID 解析当前登录用户 ID；无效或缺失时返回 0。
func ParseUserID(ctx context.Context) int64 {
	if ctx == nil {
		return 0
	}
	v := ctx.Value(JWTUserIDKey)
	if v == nil {
		return 0
	}
	switch id := v.(type) {
	case json.Number:
		n, err := id.Int64()
		if err != nil {
			return 0
		}
		return n
	case float64:
		return int64(id)
	case int64:
		return id
	case int:
		return int64(id)
	case string:
		n, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			return 0
		}
		return n
	default:
		return 0
	}
}
