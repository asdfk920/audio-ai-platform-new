package ctxutil

import (
	"context"
	"strconv"

	"github.com/zeromicro/go-zero/core/logx"
)

// GetUserId 从上下文中获取当前登录用户ID
func GetUserId(ctx context.Context) int64 {
	// 从上下文中获取用户ID
	// 这里假设用户ID存储在上下文的特定键中
	// 实际实现需要根据你的JWT中间件来确定

	// 示例实现：从上下文的"userId"键获取
	if userIdVal := ctx.Value("userId"); userIdVal != nil {
		switch v := userIdVal.(type) {
		case int64:
			return v
		case int:
			return int64(v)
		case string:
			if id, err := strconv.ParseInt(v, 10, 64); err == nil {
				return id
			}
		}
	}

	// 如果无法获取用户ID，记录警告日志
	logx.WithContext(ctx).Error("无法从上下文中获取用户ID")
	return 0
}

// GetUserIdWithDefault 从上下文中获取用户ID，如果不存在则返回默认值
func GetUserIdWithDefault(ctx context.Context, defaultValue int64) int64 {
	userId := GetUserId(ctx)
	if userId <= 0 {
		return defaultValue
	}
	return userId
}

// HasUserId 检查上下文中是否存在有效的用户ID
func HasUserId(ctx context.Context) bool {
	return GetUserId(ctx) > 0
}

// GetAdminId 从上下文中获取当前管理员ID
func GetAdminId(ctx context.Context) int64 {
	// 从上下文中获取管理员ID
	// 这里假设管理员ID存储在上下文的特定键中
	// 实际实现需要根据你的JWT中间件来确定
	
	// 示例实现：从上下文的"adminId"键获取
	if adminIdVal := ctx.Value("adminId"); adminIdVal != nil {
		switch v := adminIdVal.(type) {
		case int64:
			return v
		case int:
			return int64(v)
		case string:
			if id, err := strconv.ParseInt(v, 10, 64); err == nil {
				return id
			}
		}
	}
	
	// 如果无法获取管理员ID，记录警告日志
	logx.WithContext(ctx).Error("无法从上下文中获取管理员ID")
	return 0
}

// IsAdmin 检查当前用户是否为管理员
func IsAdmin(ctx context.Context) bool {
	return GetAdminId(ctx) > 0
}
