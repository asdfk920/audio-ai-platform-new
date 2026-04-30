package jwtx

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func TestSignAccessToken_and_MapClaims_compat(t *testing.T) {
	s, err := SignAccessToken(SignAccessOptions{
		Secret:     "secret",
		TTLSeconds: 3600,
		UserID:     42,
	})
	if err != nil {
		t.Fatal(err)
	}

	// 模拟 go-zero：解析为 MapClaims 后从 claims 取 userId
	mc := jwt.MapClaims{}
	_, err = jwt.ParseWithClaims(s, &mc, func(t *jwt.Token) (interface{}, error) {
		return []byte("secret"), nil
	}, jwt.WithJSONNumber())
	if err != nil {
		t.Fatal(err)
	}
	uid, ok := mc["userId"]
	if !ok {
		t.Fatal("missing userId")
	}
	var id int64
	switch v := uid.(type) {
	case float64:
		id = int64(v)
	case json.Number:
		id, _ = v.Int64()
	default:
		t.Fatalf("userId type %T", v)
	}
	if id != 42 {
		t.Fatalf("userId=%d", id)
	}
	if mc["token_type"] != TokenTypeAccess {
		t.Fatalf("token_type=%v", mc["token_type"])
	}
	if mc["iss"] != DefaultIssuer {
		t.Fatalf("iss=%v", mc["iss"])
	}
	if mc["sub"] != DefaultSubject {
		t.Fatalf("sub=%v", mc["sub"])
	}
	if jti, _ := mc["jti"].(string); len(jti) != 32 {
		t.Fatalf("jti len want 32 got %q", jti)
	}
}

func TestSignAccessToken_errors_zh(t *testing.T) {
	_, err := SignAccessToken(SignAccessOptions{Secret: "", TTLSeconds: 10, UserID: 1})
	if err == nil || !strings.Contains(err.Error(), "密钥") {
		t.Fatalf("got %v", err)
	}
	_, err = SignAccessToken(SignAccessOptions{Secret: "s", TTLSeconds: 0, UserID: 1})
	if err == nil || !strings.Contains(err.Error(), "有效期") {
		t.Fatalf("got %v", err)
	}
	_, err = SignAccessToken(SignAccessOptions{Secret: "s", TTLSeconds: 10, UserID: 0})
	if err == nil || !strings.Contains(err.Error(), "用户 ID") {
		t.Fatalf("got %v", err)
	}
}

func TestParseAccessToken(t *testing.T) {
	s, err := SignAccessToken(SignAccessOptions{Secret: "k", TTLSeconds: 600, UserID: 7})
	if err != nil {
		t.Fatal(err)
	}
	claims, err := ParseAccessToken("k", s)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != 7 || claims.TokenType != TokenTypeAccess {
		t.Fatalf("%+v", claims)
	}
	if claims.Issuer != DefaultIssuer {
		t.Fatal(claims.Issuer)
	}
	if claims.ID == "" {
		t.Fatal("empty jti")
	}

	_, err = ParseAccessToken("wrong", s)
	if err == nil {
		t.Fatal("want parse err")
	}
}

func TestParseAccessToken_expired(t *testing.T) {
	s, err := SignAccessToken(SignAccessOptions{Secret: "k", TTLSeconds: 1, UserID: 1})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(1100 * time.Millisecond)
	_, err = ParseAccessToken("k", s)
	if err == nil {
		t.Fatal("want expired")
	}
}

func TestSignAccessTokenHS256_ignoresIat(t *testing.T) {
	// 传入的 iat 参数应被忽略，签发时间应为当前时间附近，而非 999999
	s, err := SignAccessTokenHS256("s", 999999, 3600, 1)
	if err != nil {
		t.Fatal(err)
	}
	c, err := ParseAccessToken("s", s)
	if err != nil {
		t.Fatal(err)
	}
	if c.IssuedAt == nil {
		t.Fatal("missing iat")
	}
	now := time.Now().Unix()
	iat := c.IssuedAt.Unix()
	if iat < now-120 || iat > now+60 {
		t.Fatalf("iat should be recent (caller iat ignored), got unix=%d now=%d", iat, now)
	}
}
