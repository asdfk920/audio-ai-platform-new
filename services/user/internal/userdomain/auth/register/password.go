package register

import (
	"strings"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/common/validate"
)

// 常见弱口令（小写比对）；与长度、字母数字规则叠加使用。
var weakPasswords = map[string]struct{}{
	"123456": {}, "123456789": {}, "111111": {}, "000000": {}, "password": {},
	"qwerty": {}, "abc123": {}, "123123": {}, "admin123": {}, "letmein": {},
	"654321": {}, "888888": {}, "666666": {},
}

// ValidateRegisterPassword 注册密码：长度+字母数字（CheckPasswordMin）+ 弱口令/全相同字符/长连续升序数字。
func ValidateRegisterPassword(password string, minLen int) error {
	if err := validate.CheckPasswordMin(password, minLen); err != nil {
		return err
	}
	p := strings.TrimSpace(password)
	low := strings.ToLower(p)
	if _, ok := weakPasswords[low]; ok {
		return errorx.NewDefaultError(errorx.CodeWeakPassword)
	}
	if passwordAllSameChar(p) {
		return errorx.NewDefaultError(errorx.CodeWeakPassword)
	}
	if longAscendingDigitRun(p, 6) {
		return errorx.NewDefaultError(errorx.CodeWeakPassword)
	}
	return nil
}

// longAscendingDigitRun 是否存在长度 ≥ min 的连续升序数字子串（如 123456）。
func longAscendingDigitRun(s string, min int) bool {
	if min <= 1 {
		return false
	}
	run := 1
	var last byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= '0' && c <= '9' {
			if run > 0 && c == last+1 {
				run++
				if run >= min {
					return true
				}
			} else {
				run = 1
			}
			last = c
		} else {
			run = 0
		}
	}
	return false
}

func passwordAllSameChar(s string) bool {
	var first rune
	for i, r := range s {
		if i == 0 {
			first = r
			continue
		}
		if r != first {
			return false
		}
	}
	return s != ""
}
