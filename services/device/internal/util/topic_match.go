package util

import (
	"strings"
)

func MatchTopic(pattern, topic string) bool {
	if pattern == topic {
		return true
	}

	if strings.HasSuffix(pattern, "/#") {
		prefix := strings.TrimSuffix(pattern, "/#")
		return topic == prefix || strings.HasPrefix(topic, prefix+"/")
	}

	patternParts := strings.Split(pattern, "/")
	topicParts := strings.Split(topic, "/")

	if len(patternParts) != len(topicParts) {
		return false
	}

	for i := 0; i < len(patternParts); i++ {
		if patternParts[i] == "+" {
			continue
		}
		if patternParts[i] != topicParts[i] {
			return false
		}
	}

	return true
}
