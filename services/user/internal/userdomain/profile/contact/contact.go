package contact

import (
	"strings"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/userdomain/auth/verifycode"
)

// ParseExclusiveEmailOrMobile 邮箱、手机二选一（Trim）；返回 channel 与作为 Redis/业务主键的 target。
func ParseExclusiveEmailOrMobile(emailRaw, mobileRaw string) (target, channel string, err error) {
	email := strings.TrimSpace(emailRaw)
	mobile := strings.TrimSpace(mobileRaw)
	hasE := email != ""
	hasM := mobile != ""
	if hasE && hasM {
		return "", "", errorx.NewCodeError(errorx.CodeInvalidParam, "邮箱与手机号只能填其中一项")
	}
	if !hasE && !hasM {
		return "", "", errorx.NewCodeError(errorx.CodeInvalidParam, "请填写邮箱或手机号其中一项")
	}
	if hasE {
		email = strings.ToLower(email)
		if err := verifycode.ValidateFormat(verifycode.ChannelEmail, email); err != nil {
			return "", "", err
		}
		return email, verifycode.ChannelEmail, nil
	}
	if err := verifycode.ValidateFormat(verifycode.ChannelMobile, mobile); err != nil {
		return "", "", err
	}
	return mobile, verifycode.ChannelMobile, nil
}
