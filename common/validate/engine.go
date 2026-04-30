package validate

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"github.com/go-playground/validator/v10"
)

var mobilePattern = regexp.MustCompile(`^1[3-9]\d{9}$`)

// PasswordMaxLen 与 bcrypt 常用上限及 struct `password` 标签一致。
const PasswordMaxLen = 72

func effectiveMinPasswordLen(minLen int) int {
	if minLen <= 0 || minLen < 6 {
		return 6
	}
	return minLen
}

func passwordHasLetterAndDigit(s string) bool {
	var letter, digit bool
	for _, r := range s {
		if unicode.IsLetter(r) {
			letter = true
		}
		if unicode.IsDigit(r) {
			digit = true
		}
	}
	return letter && digit
}

// passwordMeetsPolicy 要求 s 已 Trim；空串视为不满足（由调用方区分未填）。
func passwordMeetsPolicy(s string, minLen int) bool {
	minLen = effectiveMinPasswordLen(minLen)
	if s == "" || len(s) < minLen || len(s) > PasswordMaxLen {
		return false
	}
	return passwordHasLetterAndDigit(s)
}

var (
	sharedEngine     *validator.Validate
	sharedEngineOnce sync.Once
)

// SharedEngine 返回进程内单例校验器（与 NewHTTPValidator / NewEngine 规则一致），供非 HTTP 场景复用。
func SharedEngine() *validator.Validate {
	sharedEngineOnce.Do(func() {
		sharedEngine = NewEngine()
	})
	return sharedEngine
}

// NewEngine 返回已注册 mobile、password、idcard 等自定义规则的校验器（供 go-playground/validator 使用）。
func NewEngine() *validator.Validate {
	v := validator.New()
	_ = v.RegisterValidation("mobile", validateMobile)
	_ = v.RegisterValidation("password", validatePassword)
	_ = v.RegisterValidation("idcard", validateIDCard)
	return v
}

// HTTPValidator 实现 go-zero httpx.Validator，可在 main 中 httpx.SetValidator(...) 注册。
type HTTPValidator struct {
	V *validator.Validate
}

// NewHTTPValidator 构造用于 httpx.SetValidator 的校验适配器。
func NewHTTPValidator() HTTPValidator {
	return HTTPValidator{V: SharedEngine()}
}

// Validate 在 httpx.Parse 完成后对解析结果做 struct tag 校验。
func (h HTTPValidator) Validate(_ *http.Request, data any) error {
	if h.V == nil || data == nil {
		return nil
	}
	if err := h.V.Struct(data); err != nil {
		return MapToCodeError(err)
	}
	return nil
}

func validateMobile(fl validator.FieldLevel) bool {
	s := strings.TrimSpace(fl.Field().String())
	if s == "" {
		return true
	}
	return mobilePattern.MatchString(s)
}

// 与 CheckPassword / CheckPasswordMin 同一套规则；struct 标签场景默认最小长度 6。
func validatePassword(fl validator.FieldLevel) bool {
	s := strings.TrimSpace(fl.Field().String())
	if s == "" {
		return true
	}
	return passwordMeetsPolicy(s, 6)
}

// 中国大陆 18 位居民身份证（含末位校验码）。
func validateIDCard(fl validator.FieldLevel) bool {
	s := strings.ToUpper(strings.TrimSpace(fl.Field().String()))
	if s == "" {
		return true
	}
	if len(s) != 18 {
		return false
	}
	for i := 0; i < 17; i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	last := s[17]
	if (last < '0' || last > '9') && last != 'X' {
		return false
	}
	weights := []int{7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2}
	checkCodes := "10X98765432"
	sum := 0
	for i := 0; i < 17; i++ {
		d, err := strconv.Atoi(s[i : i+1])
		if err != nil {
			return false
		}
		sum += d * weights[i]
	}
	return checkCodes[sum%11] == last
}
