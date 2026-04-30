package ctxuser

import (
	"context"
	"encoding/json"
	"testing"
)

func TestParseUserID_nilContext(t *testing.T) {
	if got := ParseUserID(nil); got != 0 { //nolint:staticcheck // SA1012: exercise nil ctx branch
		t.Fatalf("got %d", got)
	}
}

func TestParseUserID_missing(t *testing.T) {
	if ParseUserID(context.Background()) != 0 {
		t.Fatal("expected 0 when userId not set")
	}
}

func TestParseUserID_int64(t *testing.T) {
	ctx := context.WithValue(context.Background(), JWTUserIDKey, int64(7)) //nolint:staticcheck // SA1029: string key matches go-zero JWT
	if ParseUserID(ctx) != 7 {
		t.Fatalf("got %d", ParseUserID(ctx))
	}
}

func TestParseUserID_jsonNumber(t *testing.T) {
	ctx := context.WithValue(context.Background(), JWTUserIDKey, json.Number("42")) //nolint:staticcheck // SA1029
	if ParseUserID(ctx) != 42 {
		t.Fatalf("got %d", ParseUserID(ctx))
	}
}

func TestParseUserID_string(t *testing.T) {
	ctx := context.WithValue(context.Background(), JWTUserIDKey, "99") //nolint:staticcheck // SA1029
	if ParseUserID(ctx) != 99 {
		t.Fatalf("got %d", ParseUserID(ctx))
	}
}
