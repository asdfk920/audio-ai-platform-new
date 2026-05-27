package drmpass

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"testing"
	"time"
)

func TestPlatformSignatureRoundTrip(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	signer := &Signer{priv: key}
	pub, ok := key.Public().(*rsa.PublicKey)
	if !ok {
		t.Fatal("public key type")
	}

	ts := time.Now().UTC().Format(time.RFC3339Nano)
	meta := Meta{
		DRMType:        "widevine",
		ContentID:      "song-123",
		Copyright:      "© test",
		UpstreamSig:    "upstream-original",
		TimestampRFC:   ts,
		PlatformIssuer: "unit-test-issuer",
	}

	body, err := CanonicalJSON(meta, ts, meta.PlatformIssuer)
	if err != nil {
		t.Fatal(err)
	}
	sig, err := signer.SignPSS(body)
	if err != nil {
		t.Fatal(err)
	}
	b64 := base64.StdEncoding.EncodeToString(sig)
	if err := VerifyPlatformSignature(pub, meta, b64); err != nil {
		t.Fatal(err)
	}

	meta.PlatformIssuer = "tampered"
	if err := VerifyPlatformSignature(pub, meta, b64); err == nil {
		t.Fatal("expected verify failure")
	}
}
