package register

import (
	"strings"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/jacklau/audio-ai-platform/services/user/internal/userdomain/auth/verifycode"
)

// ParsedContact 注册用联系方式（邮箱、手机二选一，均已 Trim）；Password 为去空白后的密码，供哈希使用。
type ParsedContact struct {
	Channel  string
	Target   string // 与 verifycode Redis 键一致，等于 Email 或 Mobile 中非空项
	Email    string
	Mobile   string
	Password string
}

// ParseAndValidate 参数与格式、昵称、密码强度（不含库表、不含 Redis 验证码比对）。
func ParseAndValidate(req *types.RegisterReq, minPasswordLen int) (*ParsedContact, error) {
	if req == nil {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "请使用邮箱或手机号其中一种方式注册")
	}
	if minPasswordLen <= 0 {
		minPasswordLen = 6
	}

	email := strings.TrimSpace(req.Email)
	mobile := strings.TrimSpace(req.Mobile)
	hasEmail := email != ""
	hasMobile := mobile != ""

	if hasEmail && hasMobile {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "邮箱与手机号只能填其中一项")
	}
	if !hasEmail && !hasMobile {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "请使用邮箱或手机号其中一种方式注册")
	}

	var ch string
	var target string
	if hasEmail {
		email = strings.ToLower(email)
		ch = verifycode.ChannelEmail
		target = email
		if err := verifycode.ValidateFormat(ch, email); err != nil {
			return nil, err
		}
	} else {
		ch = verifycode.ChannelMobile
		target = mobile
		if err := verifycode.ValidateFormat(ch, mobile); err != nil {
			return nil, err
		}
	}

	if err := ValidateNickname(req.Nickname); err != nil {
		return nil, err
	}
	if a := strings.TrimSpace(req.Avatar); a != "" {
		if len(a) > 500 {
			return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "头像地址过长")
		}
	}

	pwd := strings.TrimSpace(req.Password)
	if err := ValidateRegisterPassword(pwd, minPasswordLen); err != nil {
		return nil, err
	}

	return &ParsedContact{
		Channel:  ch,
		Target:   target,
		Email:    email,
		Mobile:   mobile,
		Password: pwd,
	}, nil
}
