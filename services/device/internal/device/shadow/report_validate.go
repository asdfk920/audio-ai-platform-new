package shadow

import (
	"fmt"
	"math"
)

const MaxBatchReportSize = 50

// allowedReportedKeys 网关批量上报允许的 reported 字段（产品模型）
var allowedReportedKeys = map[string]struct{}{
	"battery":     {},
	"volume":      {},
	"online":      {},
	"temperature": {},
}

// ValidateDeviceReported 校验设备端 reported 字段名、类型与取值范围
func ValidateDeviceReported(reported map[string]interface{}) error {
	if len(reported) == 0 {
		return fmt.Errorf("reported 不能为空")
	}
	for key, val := range reported {
		if _, ok := allowedReportedKeys[key]; !ok {
			return fmt.Errorf("reported 含不允许的字段: %s", key)
		}
		switch key {
		case "battery", "volume":
			if err := validateIntInRange(val, 0, 100, key); err != nil {
				return err
			}
		case "online":
			if val == nil {
				return fmt.Errorf("online 不能为 null")
			}
			if _, ok := val.(bool); !ok {
				return fmt.Errorf("online 必须为布尔类型")
			}
		case "temperature":
			if val == nil {
				continue
			}
			if err := validateNumber(val, key); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateIntInRange(v interface{}, min, max int, field string) error {
	n, err := toInt64(v)
	if err != nil {
		return fmt.Errorf("%s 必须为整数", field)
	}
	if n < int64(min) || n > int64(max) {
		return fmt.Errorf("%s 取值范围为 %d-%d", field, min, max)
	}
	return nil
}

func validateNumber(v interface{}, field string) error {
	f, err := toFloat64(v)
	if err != nil {
		return fmt.Errorf("%s 必须为数字", field)
	}
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return fmt.Errorf("%s 数值非法", field)
	}
	return nil
}

func toInt64(v interface{}) (int64, error) {
	switch x := v.(type) {
	case int:
		return int64(x), nil
	case int32:
		return int64(x), nil
	case int64:
		return x, nil
	case float32:
		if x == float32(int64(x)) {
			return int64(x), nil
		}
	case float64:
		if x == float64(int64(x)) {
			return int64(x), nil
		}
	}
	return 0, fmt.Errorf("not int")
}

func toFloat64(v interface{}) (float64, error) {
	switch x := v.(type) {
	case float32:
		return float64(x), nil
	case float64:
		return x, nil
	case int:
		return float64(x), nil
	case int64:
		return float64(x), nil
	}
	return 0, fmt.Errorf("not number")
}
