package validate

import (
	"fmt"
	"strings"

	"github.com/jacklau/audio-ai-platform/common/errorx"
)

// 与业务约定的联系方式通道（与 verifycode.ChannelEmail / ChannelMobile 取值一致）。
const (
	ContactEmail  = "email"
	ContactMobile = "mobile"
)

// ValidateContactChannel 仅校验 channel 是否为支持的联系方式类型。
func ValidateContactChannel(channel string) error {
	switch strings.TrimSpace(channel) {
	case ContactEmail, ContactMobile:
		return nil
	default:
		return errorx.NewCodeError(errorx.CodeInvalidParam, "无效的联系方式类型")
	}
}

// ValidateContactTarget 校验联系方式：channel + 非空 target + 格式（邮箱/手机）。
// 错误已映射为 errorx.CodeError（含 CodeInvalidEmail / CodeInvalidMobile 等）。
func ValidateContactTarget(channel, target string) error {
	if err := ValidateContactChannel(channel); err != nil {
		return err
	}
	t := strings.TrimSpace(target)
	if t == "" {
		return errorx.NewCodeError(errorx.CodeInvalidParam, "请填写有效的邮箱或手机号")
	}
	switch strings.TrimSpace(channel) {
	case ContactEmail:
		if err := CheckEmail(t); err != nil {
			return MapToCodeError(err)
		}
	case ContactMobile:
		if err := CheckMobile(t); err != nil {
			return MapToCodeError(err)
		}
	}
	return nil
}

// CheckEmail 内置 email 规则（TrimSpace）。
func CheckEmail(s string) error {
	return SharedEngine().Var(strings.TrimSpace(s), "email")
}

// CheckMobile 项目注册的 mobile 规则：中国大陆 11 位（TrimSpace）。
func CheckMobile(s string) error {
	return SharedEngine().Var(strings.TrimSpace(s), "mobile")
}

// CheckURL 内置 url 规则（TrimSpace）。
func CheckURL(s string) error {
	return SharedEngine().Var(strings.TrimSpace(s), "url")
}

// CheckIP 内置 ip 规则，支持 IPv4 / IPv6（TrimSpace）。
func CheckIP(s string) error {
	return SharedEngine().Var(strings.TrimSpace(s), "ip")
}

// CheckIPv4 内置 ipv4 规则（TrimSpace）。
func CheckIPv4(s string) error {
	return SharedEngine().Var(strings.TrimSpace(s), "ipv4")
}

// CheckPassword 项目注册的 password 规则：最小 6 位、最长 PasswordMaxLen，且含字母与数字（TrimSpace）。
func CheckPassword(s string) error {
	return SharedEngine().Var(strings.TrimSpace(s), "password")
}

// CheckPasswordMin 业务侧密码强度：最小长度为 max(6, minLen)，上限 PasswordMaxLen，须含字母与数字。
// minLen<=0 或未配置时按 6；与 config.Register.EffectiveMinPasswordLen() 配合可统一注册/改密/重置策略。
func CheckPasswordMin(s string, minLen int) error {
	minLen = effectiveMinPasswordLen(minLen)
	s = strings.TrimSpace(s)
	if s == "" {
		return errorx.NewCodeError(errorx.CodeInvalidParam, "请设置密码")
	}
	if len(s) > PasswordMaxLen {
		return errorx.NewCodeError(errorx.CodeInvalidParam, fmt.Sprintf("密码不能超过%d位", PasswordMaxLen))
	}
	if len(s) < minLen {
		return errorx.NewCodeError(errorx.CodeInvalidParam, fmt.Sprintf("密码不能少于%d位", minLen))
	}
	if !passwordHasLetterAndDigit(s) {
		return errorx.NewCodeError(errorx.CodeInvalidParam, "密码需同时包含字母与数字")
	}
	return nil
}

// CheckIDCard 项目注册的 idcard 规则：大陆 18 位身份证（TrimSpace、末位大写 X）。
func CheckIDCard(s string) error {
	return SharedEngine().Var(strings.TrimSpace(s), "idcard")
}

// RequireNonBlank 字符串去空白后 required（用于「非空」业务校验）。
func RequireNonBlank(s string) error {
	return SharedEngine().Var(strings.TrimSpace(s), "required")
}

// CheckLen 字符串长度等于 n（validator 按 UTF-8 计）。
func CheckLen(s string, n int) error {
	return SharedEngine().Var(s, fmt.Sprintf("len=%d", n))
}

// CheckMinMax 字符串长度落在 [min,max]（含边界）。
func CheckMinMax(s string, min, max int) error {
	return SharedEngine().Var(s, fmt.Sprintf("min=%d,max=%d", min, max))
}

// CheckOneOf 枚举：s 必须为 choices 之一（choices 勿含空格）。
func CheckOneOf(s string, choices ...string) error {
	if len(choices) == 0 {
		return nil
	}
	return SharedEngine().Var(s, "oneof="+strings.Join(choices, " "))
}

// VarTag 对单个值按标签串校验（如 "required,email"、"omitempty,url"）。
func VarTag(value interface{}, tag string) error {
	return SharedEngine().Var(value, tag)
}
