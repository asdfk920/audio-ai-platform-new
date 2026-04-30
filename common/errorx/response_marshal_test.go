package errorx

import (
	"encoding/json"
	"testing"
)

type expirePayload struct {
	ExpireSeconds int `json:"expire_seconds"`
}

func TestSuccessJSONEnvelope(t *testing.T) {
	b, err := json.Marshal(Success(&expirePayload{ExpireSeconds: 180}))
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"code":0,"msg":"success","data":{"expire_seconds":180}}`
	if string(b) != want {
		t.Fatalf("got %s want %s", b, want)
	}
}
