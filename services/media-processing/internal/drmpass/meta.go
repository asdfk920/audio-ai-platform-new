// Package drmpass 实现版权/DRM 相关 HTTP 头的透传与平台侧防篡改签名（不落盘解密密钥、不解密载荷）。
//
// 原则：
// - 上游返回的 X-DRM-*（或等价头）原样复制到下游，不改写业务字段取值；
// - 平台在配置私钥时追加 X-DRM-Platform-Signature，签名覆盖规范化后的元数字段快照（不含两把签名）。
package drmpass

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// 对外 HTTP 头名（canonical），与上层 JSON 语义对应。
const (
	HdrDRMType           = "X-DRM-Type"
	HdrDRMContentID      = "X-DRM-Content-Id"
	HdrDRMKeyID          = "X-DRM-Key-Id"
	HdrDRMPSSH           = "X-DRM-Pssh"
	HdrDRMLicenseURL     = "X-DRM-License-Url"
	HdrDRMAuthToken      = "X-DRM-Auth-Token"
	HdrDRMCopyright      = "X-DRM-Copyright"
	HdrDRMUsagePolicy    = "X-DRM-Usage-Policy"
	HdrDRMSignature      = "X-DRM-Signature"          // 通常来自上游（许可证链 / 唱片方）
	HdrDRMPlatformSig    = "X-DRM-Platform-Signature" // 本代理对元数据快照的 RSA-PSS 签名
	HdrDRMPlatformIssuer = "X-DRM-Platform-Issuer"    // 签名人 / 租户标识（非秘密）
	HdrDRMProxyTS        = "X-DRM-Proxy-Timestamp"    // RFC3339，纳入签名的防重放辅助
)

const headerPrefixDRM = "x-drm-"

// Meta DRM/版权快照（与服务端下发 JSON 含义对齐；不参与解密）
type Meta struct {
	DRMType        string `json:"drm_type"`
	ContentID      string `json:"content_id"`
	KeyID          string `json:"key_id"`
	PSSH           string `json:"pssh"`
	LicenseURL     string `json:"license_url"`
	AuthToken      string `json:"auth_token"`
	Copyright      string `json:"copyright"`
	UsagePolicy    string `json:"usage_policy"`
	UpstreamSig    string `json:"signature"` // 来自上游的 X-DRM-Signature（反序列化用）
	TimestampRFC   string `json:"timestamp_rfc3339_proxy,omitempty"`
	PlatformIssuer string `json:"platform_issuer,omitempty"`
}

func isDRMHeaderKey(key string) bool {
	k := strings.ToLower(strings.TrimSpace(key))
	return strings.HasPrefix(k, headerPrefixDRM)
}

func firstHV(h http.Header, key string) string {
	vals := h.Values(key)
	if len(vals) == 0 {
		return ""
	}
	return strings.TrimSpace(vals[0])
}

// MergeFromHeaders 从 HTTP 头抽取 DRM 快照（同名头多值仅取第一项，与浏览器下常见 CDN 一致）
func MergeFromHeaders(h http.Header) Meta {
	if h == nil {
		return Meta{}
	}
	return Meta{
		DRMType:        firstHV(h, HdrDRMType),
		ContentID:      firstHV(h, HdrDRMContentID),
		KeyID:          firstHV(h, HdrDRMKeyID),
		PSSH:           firstHV(h, HdrDRMPSSH),
		LicenseURL:     firstHV(h, HdrDRMLicenseURL),
		AuthToken:      firstHV(h, HdrDRMAuthToken),
		Copyright:      firstHV(h, HdrDRMCopyright),
		UsagePolicy:    firstHV(h, HdrDRMUsagePolicy),
		UpstreamSig:    firstHV(h, HdrDRMSignature),
		TimestampRFC:   firstHV(h, HdrDRMProxyTS),
		PlatformIssuer: firstHV(h, HdrDRMPlatformIssuer),
	}
}

// signingMaterial 将 Meta 规整为单次签名的快照（不包含平台签名与 JSON 外层包装）。
func (m Meta) signingMaterial(ts string, issuer string) metaWire {
	return metaWire{
		DRMType:     m.DRMType,
		ContentID:   m.ContentID,
		KeyID:       m.KeyID,
		PSSH:        m.PSSH,
		LicenseURL:  m.LicenseURL,
		AuthToken:   m.AuthToken,
		Copyright:   m.Copyright,
		UsagePolicy: m.UsagePolicy,
		UpstreamSig: m.UpstreamSig,
		ProxyTS:     ts,
		Issuer:      issuer,
	}
}

type metaWire struct {
	DRMType     string `json:"drm_type"`
	ContentID   string `json:"content_id"`
	KeyID       string `json:"key_id"`
	PSSH        string `json:"pssh"`
	LicenseURL  string `json:"license_url"`
	AuthToken   string `json:"auth_token"`
	Copyright   string `json:"copyright"`
	UsagePolicy string `json:"usage_policy"`
	UpstreamSig string `json:"signature_upstream"`
	ProxyTS     string `json:"proxy_timestamp_rfc3339"`
	Issuer      string `json:"platform_issuer"`
}

// CanonicalJSON 用于 RSA-PSS 签名输入（字节级稳定）。
func CanonicalJSON(m Meta, proxyTS string, issuer string) ([]byte, error) {
	payload := m.signingMaterial(proxyTS, issuer)
	return json.Marshal(payload)
}

// CopyUpstreamDRMHeaders 将上游 `X-DRM-*`（除平台专有写回头）拷贝到 downstream；返回拷贝的头键数量估算（按 value 条目计）
func CopyUpstreamDRMHeaders(src http.Header, dst http.ResponseWriter) int {
	if src == nil || dst == nil {
		return 0
	}
	dh := dst.Header()
	n := 0
	for k, vv := range src {
		kk := strings.TrimSpace(k)
		if kk == "" {
			continue
		}
		if !isDRMHeaderKey(kk) {
			continue
		}
		switch strings.ToLower(kk) {
		case strings.ToLower(HdrDRMPlatformSig), strings.ToLower(HdrDRMPlatformIssuer), strings.ToLower(HdrDRMProxyTS):
			continue
		}
		for _, v := range vv {
			dh.Add(kk, v)
			n++
		}
	}
	return n
}

// AttachPlatformAttestation 在原样透传头之后写入平台时间戳 / 签发者与签名（不改变已拷贝的业务字段）。
func AttachPlatformAttestation(dst http.ResponseWriter, parsedFromUpstream Meta, signer *Signer, issuer string) error {
	if dst == nil || signer == nil {
		return nil
	}
	ts := time.Now().UTC().Format(time.RFC3339Nano)
	issuer = strings.TrimSpace(issuer)
	parsedFromUpstream.TimestampRFC = ts
	parsedFromUpstream.PlatformIssuer = issuer

	body, err := CanonicalJSON(parsedFromUpstream, ts, issuer)
	if err != nil {
		return err
	}
	sig, err := signer.SignPSS(body)
	if err != nil {
		return err
	}
	h := dst.Header()
	h.Set(HdrDRMProxyTS, ts)
	if issuer != "" {
		h.Set(HdrDRMPlatformIssuer, issuer)
	}
	h.Set(HdrDRMPlatformSig, base64.StdEncoding.EncodeToString(sig))
	return nil
}
