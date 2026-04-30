package reg

import (
	"regexp"
	"strings"

	"github.com/jacklau/audio-ai-platform/services/device/internal/config"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	defaultSN  = `^[A-Za-z0-9][A-Za-z0-9_-]{6,62}[A-Za-z0-9]$|^[A-Za-z0-9]{8}$`
	defaultPK  = `^[A-Za-z0-9][A-Za-z0-9._-]{0,62}$`
	defaultMAC = `^([0-9A-F]{2}:){5}[0-9A-F]{2}$`
)

// CompilePatterns 从配置编译 SN / ProductKey / MAC 正则，非法时回退内置规则。
func CompilePatterns(cfg *config.DeviceRegister) (sn, pk, mac *regexp.Regexp) {
	return mustPattern(cfg.SnPattern, defaultSN, "SnPattern"),
		mustPattern(cfg.ProductKeyPattern, defaultPK, "ProductKeyPattern"),
		mustPattern(cfg.MacPattern, defaultMAC, "MacPattern")
}

func mustPattern(override, def, name string) *regexp.Regexp {
	p := strings.TrimSpace(override)
	if p == "" {
		return regexp.MustCompile(def)
	}
	re, err := regexp.Compile(p)
	if err != nil {
		logx.Errorf("device register %s invalid, using default: %v", name, err)
		return regexp.MustCompile(def)
	}
	return re
}
