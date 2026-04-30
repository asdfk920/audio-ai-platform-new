package shadowmqtt

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/jacklau/audio-ai-platform/pkg/mqttx"
	"github.com/jacklau/audio-ai-platform/services/device/internal/config"
)

// ExpandTopic 将模板中的 {sn}、{id} 替换为实际值（sn 建议已大写规范化）。
func ExpandTopic(template string, snUpper string, deviceID int64) string {
	t := strings.TrimSpace(template)
	if t == "" {
		return ""
	}
	t = strings.ReplaceAll(t, "{sn}", snUpper)
	t = strings.ReplaceAll(t, "{id}", strconv.FormatInt(deviceID, 10))
	return t
}

// Defaults 填充空模板为协议约定默认值，并规范化 QoS。
func Defaults(c config.ShadowMQTT) config.ShadowMQTT {
	out := c
	if strings.TrimSpace(out.LegacyDesiredTopic) == "" {
		out.LegacyDesiredTopic = "device/{sn}/desired"
	}
	if strings.TrimSpace(out.ShadowDesiredTopicSN) == "" {
		out.ShadowDesiredTopicSN = "device/shadow/{sn}/desired"
	}
	if strings.TrimSpace(out.ShadowDesiredTopicID) == "" {
		out.ShadowDesiredTopicID = "device/shadow/{id}/desired"
	}
	if strings.TrimSpace(out.ShadowDeltaTopicSN) == "" {
		out.ShadowDeltaTopicSN = "device/shadow/{sn}/desired/delta"
	}
	if strings.TrimSpace(out.ShadowDeltaTopicID) == "" {
		out.ShadowDeltaTopicID = "device/shadow/{id}/desired/delta"
	}
	return out
}

func qosByte(q int) byte {
	if q < 0 {
		return 0
	}
	if q > 2 {
		return 2
	}
	return byte(q)
}

func desiredQoS(c config.ShadowMQTT) byte {
	if c.DesiredPublishQOS != 0 {
		return qosByte(c.DesiredPublishQOS)
	}
	return 1
}

func deltaQoS(c config.ShadowMQTT) byte {
	if c.DeltaPublishQOS != 0 {
		return qosByte(c.DeltaPublishQOS)
	}
	return 1
}

func collectCommandTopics(sm config.ShadowMQTT, snUpper string, deviceID int64) []string {
	sm = Defaults(sm)
	// 全零时与历史行为一致：仅 legacy 主题
	if !sm.EnableLegacyTopics && !sm.PublishShadowBySN && !sm.PublishShadowByID {
		sm.EnableLegacyTopics = true
	}
	var topics []string
	if sm.EnableLegacyTopics {
		if t := ExpandTopic(sm.LegacyDesiredTopic, snUpper, deviceID); t != "" {
			topics = append(topics, t)
		}
	}
	if sm.PublishShadowBySN {
		if t := ExpandTopic(sm.ShadowDesiredTopicSN, snUpper, deviceID); t != "" {
			topics = append(topics, t)
		}
	}
	if sm.PublishShadowByID {
		if t := ExpandTopic(sm.ShadowDesiredTopicID, snUpper, deviceID); t != "" {
			topics = append(topics, t)
		}
	}
	if strings.TrimSpace(sm.ShadowCommandTopicSN) != "" {
		if t := ExpandTopic(sm.ShadowCommandTopicSN, snUpper, deviceID); t != "" {
			topics = append(topics, t)
		}
	}
	if strings.TrimSpace(sm.ShadowCommandTopicID) != "" {
		if t := ExpandTopic(sm.ShadowCommandTopicID, snUpper, deviceID); t != "" {
			topics = append(topics, t)
		}
	}
	return topics
}

// PublishDesiredCommand 将 shadow_delta 指令信封发布到 legacy + device/shadow 等主题；任一次失败则返回错误。
func PublishDesiredCommand(cfg config.Config, client *mqttx.Client, snUpper string, deviceID int64, payload []byte) error {
	if client == nil || len(payload) == 0 {
		return nil
	}
	sm := Defaults(cfg.ShadowMQTT)
	topics := collectCommandTopics(sm, snUpper, deviceID)
	if len(topics) == 0 {
		return nil
	}
	q := desiredQoS(sm)
	for _, topic := range topics {
		if err := client.Publish(topic, q, false, payload); err != nil {
			return err
		}
	}
	return nil
}

// PublishJSONDelta 向 desired/delta 主题发布纯 JSON delta（仅 SN/ID 两套主题，不含 legacy）。
func PublishJSONDelta(cfg config.Config, client *mqttx.Client, snUpper string, deviceID int64, delta map[string]interface{}) error {
	if client == nil || len(delta) == 0 {
		return nil
	}
	sm := Defaults(cfg.ShadowMQTT)
	if !sm.PublishJSONDelta {
		return nil
	}
	payload, err := json.Marshal(delta)
	if err != nil {
		return err
	}
	q := deltaQoS(sm)
	var topics []string
	if sm.PublishShadowBySN {
		if t := ExpandTopic(sm.ShadowDeltaTopicSN, snUpper, deviceID); t != "" {
			topics = append(topics, t)
		}
	}
	if sm.PublishShadowByID {
		if t := ExpandTopic(sm.ShadowDeltaTopicID, snUpper, deviceID); t != "" {
			topics = append(topics, t)
		}
	}
	for _, topic := range topics {
		if err := client.Publish(topic, q, false, payload); err != nil {
			return err
		}
	}
	return nil
}
