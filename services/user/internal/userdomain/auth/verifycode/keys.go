package verifycode

import "fmt"

const codeKeyFmt = "user:verify_code:%s"

// CodeKey Redis 中验证码存储键（与校验、消费处保持一致）。
func CodeKey(target string) string {
	return fmt.Sprintf(codeKeyFmt, target)
}

func sendCountKey(target string) string {
	return fmt.Sprintf("user:verify_send_cnt:%s", target)
}

func blockKey(target string) string {
	return fmt.Sprintf("user:verify_block:%s", target)
}
