package avatar

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
)

// SeedFromTarget 生成不可逆 seed（避免把邮箱/手机号明文暴露在头像 URL 中）。
func SeedFromTarget(target string) string {
	sum := sha256.Sum256([]byte(target))
	// 取前 12 字节（24 hex）足够区分且 URL 更短
	return hex.EncodeToString(sum[:12])
}

// DefaultAvatarURL 使用公共头像生成服务（稳定、可复现）。
// 返回值长度 < 500，适配 users.avatar VARCHAR(500)。
func DefaultAvatarURL(seed string) string {
	return "https://api.dicebear.com/7.x/identicon/svg?seed=" + url.QueryEscape(seed)
}
