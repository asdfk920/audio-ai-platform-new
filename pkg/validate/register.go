package validate

import (
	"regexp"
	"strings"

	"github.com/jacklau/audio-ai-platform/common/errorx"
)

var (
	emailPattern  = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	mobilePattern = regexp.MustCompile(`^1[3-9]\d{9}$`)
)

// ValidateEmailFormat 邮箱格式。
func ValidateEmailFormat(email string) error {
	if !emailPattern.MatchString(strings.TrimSpace(email)) {
		return errorx.NewDefaultError(errorx.CodeInvalidEmail)
	}
	return nil
}

// ValidateMobileFormat 中国大陆手机号。
func ValidateMobileFormat(mobile string) error {
	if !mobilePattern.MatchString(strings.TrimSpace(mobile)) {
		return errorx.NewDefaultError(errorx.CodeInvalidMobile)
	}
	return nil
}

// ValidateRegisterPassword 非空与最小长度（不对密码做 Trim，避免改变用户输入）。
func ValidateRegisterPassword(password string, minLen int) error {
	if minLen <= 0 {
		minLen = 6
	}
	if password == "" {
		return errorx.NewCodeError(errorx.CodeInvalidParam, "请输入密码")
	}
	if len(password) < minLen {
		return errorx.NewCodeError(errorx.CodeInvalidParam, "密码长度不足")
	}
	return nil
}

// ParseRegisterAccount 注册账号维度：邮箱或手机二选一；返回规范化 target 与 by。
func ParseRegisterAccount(email, mobile string) (target, by string, err error) {
	email = strings.TrimSpace(email)
	mobile = strings.TrimSpace(mobile)
	he := email != ""
	hm := mobile != ""
	if he && hm {
		return "", "", errorx.NewCodeError(errorx.CodeInvalidParam, "邮箱与手机号只能填其中一项")
	}
	if !he && !hm {
		return "", "", errorx.NewCodeError(errorx.CodeInvalidParam, "请使用邮箱或手机号其中一种方式注册")
	}
	if he {
		el := strings.ToLower(email)
		if err := ValidateEmailFormat(el); err != nil {
			return "", "", err
		}
		return el, "email", nil
	}
	if err := ValidateMobileFormat(mobile); err != nil {
		return "", "", err
	}
	return mobile, "mobile", nil
}
