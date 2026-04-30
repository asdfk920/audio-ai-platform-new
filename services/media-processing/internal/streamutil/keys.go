package streamutil

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

func BuildStreamKey(sourceType, sourceID string, now time.Time) string {
	// 示例：content_12345_1713000000_abc123def
	ts := now.Unix()
	rand := uuid.New().String()
	// 去掉 '-'，短一些
	rand = trimDash(rand)
	if len(rand) > 10 {
		rand = rand[:10]
	}
	return fmt.Sprintf("%s_%s_%d_%s", sourceType, sourceID, ts, rand)
}

func trimDash(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '-' {
			continue
		}
		out = append(out, s[i])
	}
	return string(out)
}

func RedisPushTokenKey(streamKey string) string {
	return "push:token:" + streamKey
}

// TrimSpace 去除字符串前后空格（辅助函数）
func TrimSpace(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != ' ' && s[i] != '\t' && s[i] != '\r' && s[i] != '\n' {
			out = append(out, s[i])
		}
	}
	return string(out)
}

