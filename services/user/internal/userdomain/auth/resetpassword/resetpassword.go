package resetpassword

import (
	"strings"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/common/validate"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/jacklau/audio-ai-platform/services/user/internal/userdomain/profile/contact"
)

// ParsedReset 重置密码联系方式与新密码（已 Trim、已校验格式与一致性）。
type ParsedReset struct {
	Target      string
	Channel     string
	Email       string
	Mobile      string
	CodeTrim    string
	NewPassword string
}

// ParseAndValidate 入参校验（不含库表、不含 Redis 验证码比对）。minNewPasswordLen 建议传 config.Register.EffectiveMinPasswordLen()。
func ParseAndValidate(req *types.ResetPasswordReq, minNewPasswordLen int) (*ParsedReset, error) {
	if req == nil {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "参数错误")
	}
	target, channel, err := contact.ParseExclusiveEmailOrMobile(req.Email, req.Mobile)
	if err != nil {
		return nil, err
	}
	code := strings.TrimSpace(req.VerifyCode)
	if code == "" {
		return nil, errorx.NewDefaultError(errorx.CodeVerifyCodeInvalid)
	}
	np := strings.TrimSpace(req.NewPassword)
	nc := strings.TrimSpace(req.NewPasswordConfirm)
	if err := validate.CheckPasswordMin(np, minNewPasswordLen); err != nil {
		return nil, err
	}
	if nc == "" {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "请再次输入新密码")
	}
	if np != nc {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "两次新密码不一致")
	}
	email := strings.TrimSpace(req.Email)
	mobile := strings.TrimSpace(req.Mobile)
	return &ParsedReset{
		Target:      target,
		Channel:     channel,
		Email:       email,
		Mobile:      mobile,
		CodeTrim:    code,
		NewPassword: np,
	}, nil
}
