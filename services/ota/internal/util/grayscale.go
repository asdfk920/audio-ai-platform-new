package util

import (
	"hash/fnv"
)

func IsInGrayScale(deviceSn string, grayPercent int) bool {
	if grayPercent <= 0 {
		return false
	}
	if grayPercent >= 100 {
		return true
	}

	h := fnv.New32a()
	h.Write([]byte(deviceSn))
	hashValue := h.Sum32()

	return int(hashValue%100) < grayPercent
}
