package streamutil

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

func HMACSHA256Hex(secret, msg string) string {
	h := hmac.New(sha256.New, []byte(secret))
	_, _ = h.Write([]byte(msg))
	return hex.EncodeToString(h.Sum(nil))
}

