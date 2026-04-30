package validate

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/jacklau/audio-ai-platform/common/errorx"
)

// MapToCodeError 将 validator 错误映射为业务 CodeError，便于 httpx 统一 JSON 响应。
func MapToCodeError(err error) error {
	var verrs validator.ValidationErrors
	if !errors.As(err, &verrs) || len(verrs) == 0 {
		return errorx.NewCodeError(errorx.CodeParamError, "参数错误")
	}
	fe := verrs[0]
	msg := messageForFieldError(fe)
	code := codeForFieldError(fe)
	return errorx.NewCodeError(code, msg)
}

func codeForFieldError(fe validator.FieldError) int {
	switch fe.Tag() {
	case "email":
		return errorx.CodeInvalidEmail
	case "mobile":
		return errorx.CodeInvalidMobile
	case "password":
		return errorx.CodeInvalidParam
	case "idcard":
		return errorx.CodeInvalidParam
	default:
		return errorx.CodeParamError
	}
}

func messageForFieldError(fe validator.FieldError) string {
	tag := fe.Tag()
	switch tag {
	case "required":
		return "请填写" + chineseFieldName(fe.Field())
	case "required_without":
		return "请完整填写联系方式（邮箱与手机号需二选一）"
	case "excluded_with":
		return "邮箱与手机号只能填写其中一项"
	case "email":
		return "邮箱格式不正确，请填写有效地址（需含 @ 与域名，如 name@example.com）"
	case "mobile":
		return "手机号格式不正确，请填写11位中国大陆手机号（1开头，第二位3-9）"
	case "password":
		return fmt.Sprintf("密码需 6–%d 位且同时包含字母与数字", PasswordMaxLen)
	case "idcard":
		return "身份证号格式不正确"
	case "len":
		return fe.Field() + "长度必须为" + fe.Param()
	case "min":
		if fe.Type().Kind() == reflect.String {
			return fe.Field() + "长度不能少于" + fe.Param()
		}
		return fe.Field() + "不能小于" + fe.Param()
	case "max":
		if fe.Type().Kind() == reflect.String {
			return fe.Field() + "长度不能超过" + fe.Param()
		}
		return fe.Field() + "不能大于" + fe.Param()
	case "gte", "lte":
		return fe.Field() + "取值不在允许范围内"
	case "eqfield":
		return "两次输入的密码不一致"
	case "oneof":
		return fe.Field() + "取值无效，允许：" + strings.ReplaceAll(fe.Param(), " ", "、")
	case "numeric":
		return fe.Field() + "须为数字"
	case "url":
		return fe.Field() + "须为有效 URL"
	case "uuid":
		return fe.Field() + "须为有效 UUID"
	case "ip":
		return fe.Field() + "须为有效 IP 地址"
	case "ipv4":
		return fe.Field() + "须为有效 IPv4 地址"
	case "ipv6":
		return fe.Field() + "须为有效 IPv6 地址"
	case "dive":
		return fe.Field() + "包含无效元素"
	default:
		return "参数校验未通过（" + fe.Field() + "）"
	}
}

func chineseFieldName(field string) string {
	switch field {
	case "Email", "OldEmail", "NewEmail":
		return "邮箱"
	case "Mobile", "OldMobile", "NewMobile":
		return "手机号"
	case "Password", "OldPassword", "NewPassword":
		return "密码"
	case "VerifyCode", "OldVerifyCode", "NewVerifyCode":
		return "验证码"
	case "RefreshToken":
		return "刷新令牌"
	case "Code":
		return "授权码"
	case "UserId":
		return "用户 ID"
	case "Nickname":
		return "昵称"
	case "Avatar":
		return "头像"
	case "RealName":
		return "姓名"
	case "IdNumber":
		return "身份证号"
	case "Scene":
		return "业务场景"
	default:
		return field
	}
}
