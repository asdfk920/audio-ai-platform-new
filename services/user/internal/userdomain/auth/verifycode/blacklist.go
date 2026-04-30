package verifycode

import (
	"strings"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/config"
)

// CheckBlacklist 配置中的号段/邮箱片段子串命中则拒绝发码。
func CheckBlacklist(cfg config.VerifyCodeConfig, channel, target string) error {
	if err := ValidateChannel(channel); err != nil {
		return err
	}
	t := strings.ToLower(strings.TrimSpace(target))
	switch channel {
	case ChannelMobile:
		for _, frag := range cfg.BlacklistMobiles {
			if frag == "" {
				continue
			}
			if strings.Contains(t, strings.ToLower(strings.TrimSpace(frag))) {
				return errorx.NewCodeError(errorx.CodeInvalidParam, "该号码暂不支持接收验证码")
			}
		}
	case ChannelEmail:
		for _, frag := range cfg.BlacklistEmails {
			if frag == "" {
				continue
			}
			if strings.Contains(t, strings.ToLower(strings.TrimSpace(frag))) {
				return errorx.NewCodeError(errorx.CodeInvalidParam, "该邮箱暂不支持接收验证码")
			}
		}
	}
	return nil
}
