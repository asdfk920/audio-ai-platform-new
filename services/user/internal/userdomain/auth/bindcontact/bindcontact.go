package bindcontact

import (
	"strings"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/jacklau/audio-ai-platform/services/user/internal/userdomain/profile/contact"
)

// ParsedBind 绑定联系方式（已校验格式与验证码非空）。
type ParsedBind struct {
	Target   string
	Channel  string
	Email    string // trim 后，与 channel 对应的一项有值
	Mobile   string
	CodeTrim string
}

// ParseAndValidate 入参校验（不含库表、不含 Redis 验证码内容比对）。
func ParseAndValidate(req *types.BindContactReq) (*ParsedBind, error) {
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
	email := strings.TrimSpace(req.Email)
	mobile := strings.TrimSpace(req.Mobile)
	return &ParsedBind{
		Target:   target,
		Channel:  channel,
		Email:    email,
		Mobile:   mobile,
		CodeTrim: code,
	}, nil
}
