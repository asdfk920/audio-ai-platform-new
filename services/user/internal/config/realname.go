package config

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"time"
)

// RealName 实名认证：加密密钥、提交互斥、Mock 核验、人工审核开关、管理端口令。
type RealName struct {
	// IdNumberEncryptKeyBase64 AES-256-GCM 密钥，32 字节经标准 Base64 编码；空则使用 Auth.AccessSecret 派生（仅便于本地开发，生产务必配置独立密钥）。
	IdNumberEncryptKeyBase64 string `json:",optional"`
	SubmitLockSeconds          int    `json:",optional"`
	// MockOutcome 非空且 Provider=mock 时生效：pass | fail | error
	MockOutcome string `json:",optional"`
	Provider    string `json:",optional"` // mock（默认）| http（预留）
	// RequireManualReview 为 true 时，三方核验通过后仍保持审核中，由独立管理后台审核落库（可调用 DAO RealNameAdminFinalize 或直连 DB）。
	RequireManualReview bool `json:",optional"`
	// SkipFaceForPersonal 为 true 时个人身份证可不传人脸 Base64；默认 false（要求人脸，与产品「人脸识别」一致）。
	SkipFaceForPersonal bool   `json:",optional"`
	ThirdPartyChannel   string `json:",optional"` // 审计展示用渠道名
}

// EffectiveRequireFaceForPersonal 个人是否必须上传人脸（未配置 Skip 时默认必须）。
func (r RealName) EffectiveRequireFaceForPersonal() bool {
	return !r.SkipFaceForPersonal
}

// EffectiveSubmitLock 提交互斥锁 TTL。
func (r RealName) EffectiveSubmitLock() time.Duration {
	sec := r.SubmitLockSeconds
	if sec <= 0 {
		sec = 5
	}
	if sec > 60 {
		sec = 60
	}
	return time.Duration(sec) * time.Second
}

// EffectiveMockOutcome 默认 pass。
func (r RealName) EffectiveMockOutcome() string {
	switch r.MockOutcome {
	case "fail", "error":
		return r.MockOutcome
	default:
		return "pass"
	}
}

// EffectiveProvider 默认 mock。
func (r RealName) EffectiveProvider() string {
	if r.Provider == "http" {
		return "http"
	}
	return "mock"
}

// EffectiveThirdPartyChannel 展示名。
func (r RealName) EffectiveThirdPartyChannel() string {
	if r.ThirdPartyChannel != "" {
		return r.ThirdPartyChannel
	}
	return "mock"
}

// ResolveIDNumberKey 解析 32 字节 AES 密钥。
func (c Config) ResolveIDNumberKey() ([]byte, error) {
	raw := c.RealName.IdNumberEncryptKeyBase64
	if raw != "" {
		k, err := base64.StdEncoding.DecodeString(raw)
		if err != nil {
			return nil, err
		}
		if len(k) != 32 {
			return nil, errors.New("realname: IdNumberEncryptKeyBase64 must decode to 32 bytes")
		}
		return k, nil
	}
	// 开发兜底：从 JWT secret 派生（不可用于生产合规）
	sum := sha256.Sum256([]byte(c.Auth.AccessSecret))
	return sum[:], nil
}
