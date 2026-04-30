package register

import (
	"testing"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
)

func TestParseAndValidate_nilReq(t *testing.T) {
	_, err := ParseAndValidate(nil, 6)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseAndValidate_bothEmailAndMobile(t *testing.T) {
	_, err := ParseAndValidate(&types.RegisterReq{
		Email:  "a@b.com",
		Mobile: "13900000000",
	}, 6)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseAndValidate_invalidEmailFormat(t *testing.T) {
	_, err := ParseAndValidate(&types.RegisterReq{
		Email:    "not-an-email",
		Password: "123456",
	}, 6)
	ce, ok := err.(*errorx.CodeError)
	if !ok || ce.GetCode() != errorx.CodeInvalidEmail {
		t.Fatalf("got %v", err)
	}
}

func TestParseAndValidate_passwordTooShort(t *testing.T) {
	_, err := ParseAndValidate(&types.RegisterReq{
		Mobile:   "13912345678",
		Password: "12345",
	}, 6)
	if err == nil {
		t.Fatal("expected error")
	}
}
