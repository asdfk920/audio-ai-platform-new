package logic

import (
	"context"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/services/device/internal/model"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/util"
	"github.com/zeromicro/go-zero/core/logx"
)

type MqttACLLogic struct {
	svcCtx *svc.ServiceContext
}

func NewMqttACLLogic(svcCtx *svc.ServiceContext) *MqttACLLogic {
	return &MqttACLLogic{svcCtx: svcCtx}
}

type MqttACLReq struct {
	ClientID string `json:"clientid"`
	Username string `json:"username"`
	Topic    string `json:"topic"`
	Action   string `json:"action"`
}

type MqttACLResp struct {
	Result string `json:"result"` // "allow" 或 "deny"
}

func (l *MqttACLLogic) CheckPermission(req *MqttACLReq) (*MqttACLResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sn := strings.ToUpper(strings.TrimSpace(req.ClientID))
	topic := strings.TrimSpace(req.Topic)
	action := strings.ToLower(strings.TrimSpace(req.Action))

	logx.Infof("MQTT ACL 鉴权请求: sn=%s, topic=%s, action=%s", sn, topic, action)

	if sn == "" || topic == "" || action == "" {
		logx.Slowf("MQTT ACL: 参数不完整 clientid=%s, topic=%s, action=%s", req.ClientID, topic, action)
		return &MqttACLResp{Result: "deny"}, nil
	}

	actionCode := l.parseAction(action)

	deviceRules, err := l.svcCtx.DeviceTopicACLRepo.GetDeviceACLRules(ctx, sn)
	if err != nil {
		logx.Errorf("MQTT ACL: 查询设备规则失败 sn=%s, err=%v", sn, err)
		return &MqttACLResp{Result: "deny"}, nil
	}

	globalRules, err := l.svcCtx.DeviceTopicACLRepo.GetGlobalACLRules(ctx)
	if err != nil {
		logx.Errorf("MQTT ACL: 查询全局规则失败 err=%v", err)
		return &MqttACLResp{Result: "deny"}, nil
	}

	allRules := append(deviceRules, globalRules...)

	result := l.evaluateRules(allRules, topic, actionCode, sn)

	logx.Infof("MQTT ACL 鉴权结果: sn=%s, topic=%s, action=%s, result=%s",
		sn, topic, action, result)

	if result == "allow" {
		return &MqttACLResp{Result: "allow"}, nil
	}

	return &MqttACLResp{Result: "deny"}, nil
}

func (l *MqttACLLogic) parseAction(action string) int16 {
	switch action {
	case "publish":
		return model.TopicActionPublish
	case "subscribe":
		return model.TopicActionSubscribe
	default:
		return model.TopicActionBoth
	}
}

func (l *MqttACLLogic) evaluateRules(rules []model.DeviceTopicACL, topic string, action int16, sn string) string {
	var matchedAllow bool
	var matchedDeny bool

	for _, rule := range rules {
		actionMatch := (rule.Action == model.TopicActionBoth) || (rule.Action == action)
		if !actionMatch {
			continue
		}

		if !util.MatchTopic(rule.TopicPattern, topic) {
			continue
		}

		logx.Infof("MQTT ACL: 规则命中 pattern=%s, permission=%d, action=%d",
			rule.TopicPattern, rule.Permission, rule.Action)

		if rule.Permission == model.PermissionAllow {
			matchedAllow = true
		} else {
			matchedDeny = true
		}
	}

	if matchedDeny {
		return "deny"
	}
	if matchedAllow {
		return "allow"
	}

	return "deny"
}
