package refreshtoken

// MaskToken 审计日志中 refresh_token 脱敏（仅前缀，防日志泄露完整凭证）。
func MaskToken(t string) string {
	if len(t) <= 8 {
		return "***"
	}
	return t[:4] + "…" + t[len(t)-4:]
}
