package refreshtoken

import "fmt"

// KeyRefresh Redis 键：refresh_token -> userId。
func KeyRefresh(token string) string {
	return fmt.Sprintf("user:refresh:%s", token)
}

// KeyUserIndex Redis 键：userId -> 当前有效的 refresh_token。
func KeyUserIndex(userID int64) string {
	return fmt.Sprintf("user:%d:refresh", userID)
}
