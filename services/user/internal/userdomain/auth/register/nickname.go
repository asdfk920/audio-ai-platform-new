package register

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jacklau/audio-ai-platform/common/errorx"
)

const nicknameMaxRunes = 100

var nicknameForbidden = []string{"管理员", "官方", "系统", "客服", "admin", "root"}

// ValidateNickname 校验用户填写的昵称（空串表示走默认昵称，不报错）。
func ValidateNickname(nickname string) error {
	s := strings.TrimSpace(nickname)
	if s == "" {
		return nil
	}
	if utf8.RuneCountInString(s) > nicknameMaxRunes {
		return errorx.NewCodeError(errorx.CodeInvalidNickname, "昵称不能超过100个字符")
	}
	for _, r := range s {
		if unicode.IsControl(r) {
			return errorx.NewDefaultError(errorx.CodeInvalidNickname)
		}
	}
	low := strings.ToLower(s)
	for _, w := range nicknameForbidden {
		if strings.Contains(s, w) || strings.Contains(low, strings.ToLower(w)) {
			return errorx.NewDefaultError(errorx.CodeInvalidNickname)
		}
	}
	return nil
}

// DefaultNickname 未传昵称时生成「用户」+ 6 位随机数字。
func DefaultNickname() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(900_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("用户%06d", int(n.Int64())+100_000), nil
}
