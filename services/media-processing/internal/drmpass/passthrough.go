package drmpass

import (
	"net/http"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"
)

// ApplyPassthrough writes upstream DRM headers verbatim, optionally adds platform RSA-PSS attestation (doSign && signer!=nil).
func ApplyPassthrough(upstream http.Header, w http.ResponseWriter, signer *Signer, issuer string, doSign bool) int {
	n := CopyUpstreamDRMHeaders(upstream, w)
	if n == 0 && (!doSign || signer == nil) {
		return 0
	}
	meta := MergeFromHeaders(upstream)
	if doSign && signer != nil {
		iss := strings.TrimSpace(issuer)
		if err := AttachPlatformAttestation(w, meta, signer, iss); err != nil {
			logx.Errorf("[drmpass] platform attestation failed: %v", err)
			return n
		}
		logx.Infof("[drmpass] upstream_drm_headers=%d platform_attestation=1 issuer=%s", n, truncateForLog(iss, 48))
		return n
	}
	if n > 0 {
		logx.Infof("[drmpass] upstream_drm_headers=%d (no platform key, skip attest)", n)
	}
	return n
}

func truncateForLog(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
