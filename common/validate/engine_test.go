package validate

import (
	"testing"

	"github.com/go-playground/validator/v10"
)

func TestCustomValidators(t *testing.T) {
	v := NewEngine()

	if err := v.Var("13800138000", "mobile"); err != nil {
		t.Fatalf("mobile: %v", err)
	}
	if err := v.Var("123", "mobile"); err == nil {
		t.Fatal("mobile: want error for short number")
	}

	if err := v.Var("abc12345", "password"); err != nil {
		t.Fatalf("password: %v", err)
	}
	if err := v.Var("abcdef", "password"); err == nil {
		t.Fatal("password: want error when no digit")
	}
	if err := v.Var("123456", "password"); err == nil {
		t.Fatal("password: want error when no letter")
	}

	// 有效校验码样例（算法自洽）
	if err := v.Var("11010519491231002X", "idcard"); err != nil {
		t.Fatalf("idcard: %v", err)
	}
	if err := v.Var("110105194912310021", "idcard"); err == nil {
		t.Fatal("idcard: want error for wrong check digit")
	}
}

func TestCheckPasswordMin(t *testing.T) {
	if err := CheckPasswordMin("ab12cd", 6); err != nil {
		t.Fatal(err)
	}
	if err := CheckPasswordMin("ab12", 6); err == nil {
		t.Fatal("want min len")
	}
	if err := CheckPasswordMin("abcdef", 6); err == nil {
		t.Fatal("want digit")
	}
	if err := CheckPasswordMin("123456", 6); err == nil {
		t.Fatal("want letter")
	}
	if err := CheckPasswordMin("ab12cd", 8); err == nil {
		t.Fatal("want min 8")
	}
}

func TestCheckEmailMobile(t *testing.T) {
	if err := CheckEmail("a@b.co"); err != nil {
		t.Fatal(err)
	}
	if err := CheckEmail("not-an-email"); err == nil {
		t.Fatal("want error")
	}
	if err := CheckMobile("13800138000"); err != nil {
		t.Fatal(err)
	}
	if err := CheckMobile("123"); err == nil {
		t.Fatal("want error")
	}
}

func TestMapToCodeError(t *testing.T) {
	type S struct {
		Email string `validate:"required,email"`
	}
	v := NewEngine()
	err := v.Struct(&S{Email: "bad"})
	out := MapToCodeError(err)
	if out == nil {
		t.Fatal("expected error")
	}
	if _, ok := out.(interface{ GetCode() int }); !ok {
		t.Fatalf("expected CodeError, got %T", out)
	}
}

func TestScalarHelpers(t *testing.T) {
	if err := CheckURL("https://example.com"); err != nil {
		t.Fatal(err)
	}
	if err := CheckURL("not-a-url"); err == nil {
		t.Fatal("want url error")
	}
	if err := CheckIP("127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	if err := CheckIPv4("127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	if err := CheckOneOf("register", "register", "reset", "bind"); err != nil {
		t.Fatal(err)
	}
	if err := CheckOneOf("x", "a", "b"); err == nil {
		t.Fatal("want oneof error")
	}
	if err := RequireNonBlank("  "); err == nil {
		t.Fatal("want required error")
	}
	if err := CheckLen("abcd", 4); err != nil {
		t.Fatal(err)
	}
}

func TestValidateStructNestedAndDive(t *testing.T) {
	type inner struct {
		Email string `validate:"required,email"`
	}
	type outer struct {
		I inner
	}
	if err := ValidateStruct(&outer{}); err == nil {
		t.Fatal("want nested error")
	}

	type row struct {
		Email string `validate:"email"`
	}
	type holder struct {
		Rows []row `validate:"dive"`
	}
	if err := ValidateStruct(&holder{Rows: []row{{Email: "bad"}}}); err == nil {
		t.Fatal("want dive error")
	}
}

func TestValidateContactTarget(t *testing.T) {
	if err := ValidateContactTarget(ContactEmail, "a@b.co"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateContactTarget(ContactMobile, "13800138000"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateContactTarget(ContactEmail, ""); err == nil {
		t.Fatal("want empty error")
	}
	if err := ValidateContactTarget("wechat", "x"); err == nil {
		t.Fatal("want channel error")
	}
}

func TestRegisterValidation(t *testing.T) {
	fn := func(fl validator.FieldLevel) bool {
		return fl.Field().String() == "ok"
	}
	if err := RegisterValidation("validate_pkg_test_marker", fn); err != nil {
		t.Fatal(err)
	}
	if err := VarTag("ok", "validate_pkg_test_marker"); err != nil {
		t.Fatal(err)
	}
	if err := VarTag("no", "validate_pkg_test_marker"); err == nil {
		t.Fatal("want error")
	}
}
