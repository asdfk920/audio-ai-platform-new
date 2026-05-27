package drmpass

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
)

// Signer RSA-PSS SHA256（仅用于 DRM 元数据快照防篡改）
type Signer struct {
	priv *rsa.PrivateKey
}

// LoadSignerFromPath 读取 PEM PKCS1/PKCS8 私钥
func LoadSignerFromPath(path string) (*Signer, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("drmpass: read private key: %w", err)
	}
	return LoadSignerFromPEM(raw)
}

// LoadSignerFromPEM PEM 解码 RSA 私钥
func LoadSignerFromPEM(b []byte) (*Signer, error) {
	block, _ := pem.Decode(b)
	if block == nil {
		return nil, fmt.Errorf("drmpass: no PEM block in key")
	}
	if k, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return &Signer{priv: k}, nil
	}
	keyAny, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("drmpass: parse pem: %w", err)
	}
	priv, ok := keyAny.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("drmpass: expected RSA private key")
	}
	return &Signer{priv: priv}, nil
}

// LoadPublicKeyFromPEM 加载公钥（设备 / 离线验签）
func LoadPublicKeyFromPEM(b []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(b)
	if block == nil {
		return nil, fmt.Errorf("drmpass: no PEM block in pubkey")
	}
	pubAny, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	pub, ok := pubAny.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("drmpass: expected RSA public key")
	}
	return pub, nil
}

func (s *Signer) SignPSS(msg []byte) ([]byte, error) {
	if s == nil || s.priv == nil {
		return nil, fmt.Errorf("drmpass: nil signer")
	}
	sum := sha256.Sum256(msg)
	return rsa.SignPSS(rand.Reader, s.priv, crypto.SHA256, sum[:], &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash})
}

// VerifyPlatformSignature 校验 Base64 的 X-DRM-Platform-Signature；Issuer 取自 m.PlatformIssuer。
func VerifyPlatformSignature(pub *rsa.PublicKey, m Meta, platformSigB64 string) error {
	if pub == nil {
		return fmt.Errorf("drmpass: nil public key")
	}
	sig, err := decodeB64(platformSigB64)
	if err != nil {
		return err
	}
	issuer := strings.TrimSpace(m.PlatformIssuer)
	body, err := CanonicalJSON(m, m.TimestampRFC, issuer)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(body)
	return rsa.VerifyPSS(pub, crypto.SHA256, sum[:], sig, &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash})
}

func decodeB64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(strings.TrimSpace(s))
}
