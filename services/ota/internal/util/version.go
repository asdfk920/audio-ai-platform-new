package util

import (
	"strconv"
	"strings"
)

func CompareVersion(v1, v2 string) int {
	parts1 := parseVersion(v1)
	parts2 := parseVersion(v2)

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		num1 := 0
		num2 := 0

		if i < len(parts1) {
			num1 = parts1[i]
		}
		if i < len(parts2) {
			num2 = parts2[i]
		}

		if num1 > num2 {
			return 1
		}
		if num1 < num2 {
			return -1
		}
	}

	return 0
}

func parseVersion(version string) []int {
	version = strings.TrimPrefix(version, "v")
	version = strings.TrimPrefix(version, "V")
	version = strings.TrimPrefix(version, "FW_")
	version = strings.TrimPrefix(version, "fw_")

	parts := strings.Split(version, ".")
	result := make([]int, 0, len(parts))

	for _, part := range parts {
		num, err := strconv.Atoi(part)
		if err != nil {
			continue
		}
		result = append(result, num)
	}

	return result
}
