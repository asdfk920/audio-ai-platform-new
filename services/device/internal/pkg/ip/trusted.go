// Package ip 可信代理下的客户端 IP 解析（设备服务通用能力）。
package ip

import (
	"net"
	"net/http"
	"strings"
)

// ParseTrustedProxies 解析 CIDR 或单 IP，用于判断是否信任 X-Forwarded-For / X-Real-IP。
func ParseTrustedProxies(ss []string) ([]*net.IPNet, error) {
	var out []*net.IPNet
	for _, s := range ss {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if strings.Contains(s, "/") {
			_, n, err := net.ParseCIDR(s)
			if err != nil {
				return nil, err
			}
			out = append(out, n)
			continue
		}
		ip := net.ParseIP(s)
		if ip == nil {
			continue
		}
		if v4 := ip.To4(); v4 != nil {
			out = append(out, &net.IPNet{IP: v4, Mask: net.CIDRMask(32, 32)})
		} else {
			out = append(out, &net.IPNet{IP: ip, Mask: net.CIDRMask(128, 128)})
		}
	}
	return out, nil
}

// ClientIP 仅使用 TCP 直连地址（不信任转发头），适合未配置反向代理或安全默认。
func ClientIP(r *http.Request) string {
	return ClientIPTrusted(r, nil)
}

// ClientIPTrusted 当直连客户端 IP 落在 trusted 网段内时，才采用 X-Forwarded-For 首段或 X-Real-IP；trusted 为空则永不信任转发头。
func ClientIPTrusted(r *http.Request, trusted []*net.IPNet) string {
	if r == nil {
		return ""
	}
	direct := remoteHostFromRequest(r)
	if len(trusted) == 0 {
		return clampIPLen(direct)
	}
	dip := net.ParseIP(direct)
	if dip == nil || !ipInAnyNet(dip, trusted) {
		return clampIPLen(direct)
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			if ip := strings.TrimSpace(parts[0]); ip != "" {
				return clampIPLen(ip)
			}
		}
	}
	if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
		return clampIPLen(xri)
	}
	return clampIPLen(direct)
}

func remoteHostFromRequest(r *http.Request) string {
	addr := strings.TrimSpace(r.RemoteAddr)
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}

func ipInAnyNet(ip net.IP, nets []*net.IPNet) bool {
	for _, n := range nets {
		if n != nil && n.Contains(ip) {
			return true
		}
	}
	return false
}

func clampIPLen(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 45 {
		return s[:45]
	}
	return s
}
