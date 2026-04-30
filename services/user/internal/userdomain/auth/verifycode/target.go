package verifycode

import (
	"strings"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/common/validate"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
)

// ParseTarget 邮箱、手机二选一；去空白；未填或同时填返回错误。
func ParseTarget(req *types.SendVerifyCodeReq) (target, channel string, err error) {
	if req == nil {
		return "", "", errorx.NewCodeError(errorx.CodeInvalidParam, "请填写邮箱或手机号其中一项")
	}
	email := strings.TrimSpace(req.Email)
	mobile := strings.TrimSpace(req.Mobile)
	hasEmail := email != ""
	hasMobile := mobile != ""
	if hasEmail && hasMobile {
		return "", "", errorx.NewCodeError(errorx.CodeInvalidParam, "邮箱与手机号只能填其中一项")
	}
	if !hasEmail && !hasMobile {
		return "", "", errorx.NewCodeError(errorx.CodeInvalidParam, "请填写邮箱或手机号其中一项")
	}
	if hasEmail {
		// 邮箱统一小写，与发码 Redis 键、库内查询一致（登录即注册 / 密码注册共用发码键语义）。
		return strings.ToLower(email), ChannelEmail, nil
	}
	return mobile, ChannelMobile, nil
}

// ValidateChannel 二次校验：仅允许已定义的通道常量。
func ValidateChannel(channel string) error {
	return validate.ValidateContactChannel(channel)
}

// ValidateFormat 非空与格式校验（须在 ParseTarget 之后调用），逻辑在 common/validate.ValidateContactTarget。
func ValidateFormat(channel, target string) error {
	return validate.ValidateContactTarget(channel, target)
}
