package verifycode

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/jacklau/audio-ai-platform/common/errorx"
)

const sixDigitMod = 1_000_000

// GenSixDigits 使用 CSPRNG 生成 6 位数字验证码，不可预测；失败时返回统一业务错误（细节仅打服务端日志）。
func GenSixDigits() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(sixDigitMod))
	if err != nil {
		return "", errorx.NewDefaultError(errorx.CodeSystemError)
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
