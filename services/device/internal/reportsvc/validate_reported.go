package reportsvc

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/jacklau/audio-ai-platform/common/errorx"
)

// ValidateReportedFields 校验 reported 中已出现字段的取值范围；未传的嵌套对象不校验。
func ValidateReportedFields(reported map[string]interface{}) error {
	if reported == nil {
		return nil
	}
	if v, ok := int32Value(reported["battery"]); ok {
		if v < 0 || v > 100 {
			return errorx.NewCodeError(errorx.CodeInvalidParam, "battery 须在 0–100 之间")
		}
	}
	if power := nestedMap(reported, "power"); power != nil {
		if v, ok := int32Value(power["percent"]); ok {
			if v < 0 || v > 100 {
				return errorx.NewCodeError(errorx.CodeInvalidParam, "power.percent 须在 0–100 之间")
			}
		}
		if v, ok := toFloat64(power["health_percent"]); ok {
			if v < 0 || v > 100 {
				return errorx.NewCodeError(errorx.CodeInvalidParam, "power.health_percent 须在 0–100 之间")
			}
		}
	}
	if st := nestedMap(reported, "storage"); st != nil {
		total, okT := toFloat64(st["total_bytes"])
		used, okU := toFloat64(st["used_bytes"])
		if okT && total < 0 {
			return errorx.NewCodeError(errorx.CodeInvalidParam, "storage.total_bytes 不能为负")
		}
		if okU && used < 0 {
			return errorx.NewCodeError(errorx.CodeInvalidParam, "storage.used_bytes 不能为负")
		}
		if okT && okU && used > total {
			return errorx.NewCodeError(errorx.CodeInvalidParam, "storage.used_bytes 不能大于 total_bytes")
		}
	}
	if uwb := nestedMap(reported, "uwb"); uwb != nil {
		if v, ok := toFloat64(uwb["accuracy_m"]); ok && v < 0 {
			return errorx.NewCodeError(errorx.CodeInvalidParam, "uwb.accuracy_m 不能为负")
		}
		for _, coord := range []string{"x", "y", "z"} {
			if _, exists := uwb[coord]; !exists {
				continue
			}
			if _, ok := toFloat64(uwb[coord]); !ok {
				return errorx.NewCodeError(errorx.CodeInvalidParam, fmt.Sprintf("uwb.%s 须为数字", coord))
			}
		}
	}
	return nil
}

func toFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		if err != nil {
			return 0, false
		}
		return f, true
	case string:
		n = strings.TrimSpace(n)
		if n == "" {
			return 0, false
		}
		f, err := strconv.ParseFloat(n, 64)
		if err != nil {
			return 0, false
		}
		return f, true
	}
	return 0, false
}
