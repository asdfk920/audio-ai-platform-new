package mqttingest

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/jacklau/audio-ai-platform/pkg/mqttx"
	"github.com/jacklau/audio-ai-platform/services/device/internal/config"
	"github.com/jacklau/audio-ai-platform/services/device/internal/device/reg"
	"github.com/jacklau/audio-ai-platform/services/device/internal/reportsvc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/statuspersist"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
)

// Handler MQTT device/{sn}/report：解析统一报文后进入 reportsvc。
type Handler struct {
	Log    logx.Logger
	Report *reportsvc.Service
}

func NewHandler(db *sql.DB, rdb *redis.Client, c config.Config, _ *statuspersist.Pool, publisher *mqttx.Client) *Handler {
	sc := svc.NewServiceContext(c, db, rdb)
	sc.SetMQTTClient(publisher)
	return &Handler{
		Log:    logx.WithContext(context.Background()),
		Report: reportsvc.New(sc),
	}
}

type mqttReportPayload struct {
	DeviceSn        string             `json:"device_sn"`
	DeviceSecret    string             `json:"device_secret"`
	ReportId        string             `json:"report_id"`
	Timestamp       int64              `json:"timestamp"`
	Reported        json.RawMessage    `json:"reported"`
	RunState        string             `json:"run_state"`
	Battery         *int32             `json:"battery"`
	FirmwareVersion string             `json:"firmware_version"`
	Online          *bool              `json:"online"`
	IP              string             `json:"ip"`
	Children        []mqttChildPayload `json:"children"`
	History         []mqttHistoryItem  `json:"history"`
}

type mqttChildPayload struct {
	ChildKey  string          `json:"child_key"`
	ChildSn   string          `json:"child_sn"`
	ChildType string          `json:"child_type"`
	ChildName string          `json:"child_name"`
	Timestamp int64           `json:"timestamp"`
	Online    *bool           `json:"online"`
	Reported  json.RawMessage `json:"reported"`
	Metadata  json.RawMessage `json:"metadata"`
}

type mqttHistoryItem struct {
	ReportId        string             `json:"report_id"`
	Timestamp       int64              `json:"timestamp"`
	Reported        json.RawMessage    `json:"reported"`
	RunState        string             `json:"run_state"`
	Battery         *int32             `json:"battery"`
	FirmwareVersion string             `json:"firmware_version"`
	Online          *bool              `json:"online"`
	IP              string             `json:"ip"`
	Children        []mqttChildPayload `json:"children"`
}

// OnMessage 实现 mqtt.MessageHandler。
func (h *Handler) OnMessage(_ mqtt.Client, m mqtt.Message) {
	if h == nil || m == nil {
		return
	}
	snRaw := parseSNFromTopic(m.Topic())
	snNorm := reg.NormalizeSN(snRaw)
	if len(snNorm) < 8 {
		h.Log.Infof("mqtt report ignore topic=%q bad sn", m.Topic())
		return
	}
	var payload mqttReportPayload
	if err := json.Unmarshal(m.Payload(), &payload); err != nil {
		h.Log.Infof("mqtt report sn=%s bad json: %v", reg.MaskSN(snNorm), err)
		return
	}
	deviceSN := strings.TrimSpace(payload.DeviceSn)
	if deviceSN == "" {
		deviceSN = snNorm
	}
	secret := strings.TrimSpace(payload.DeviceSecret)
	if secret == "" {
		h.Log.Infof("mqtt report sn=%s missing device_secret", reg.MaskSN(snNorm))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	input := reportsvc.ReportInput{
		DeviceSN:        deviceSN,
		DeviceSecret:    secret,
		ReportID:        payload.ReportId,
		Timestamp:       payload.Timestamp,
		Reported:        payload.Reported,
		RunState:        payload.RunState,
		Battery:         payload.Battery,
		FirmwareVersion: payload.FirmwareVersion,
		Online:          payload.Online,
		IP:              payload.IP,
		Children:        make([]reportsvc.ChildReportInput, 0, len(payload.Children)),
		History:         make([]reportsvc.CachedReportInput, 0, len(payload.History)),
		Source:          "mqtt",
	}
	for _, child := range payload.Children {
		input.Children = append(input.Children, reportsvc.ChildReportInput{
			ChildKey:  child.ChildKey,
			ChildSN:   child.ChildSn,
			ChildType: child.ChildType,
			ChildName: child.ChildName,
			Timestamp: child.Timestamp,
			Online:    child.Online,
			Reported:  child.Reported,
			Metadata:  child.Metadata,
		})
	}
	for _, item := range payload.History {
		historyChildren := make([]reportsvc.ChildReportInput, 0, len(item.Children))
		for _, child := range item.Children {
			historyChildren = append(historyChildren, reportsvc.ChildReportInput{
				ChildKey:  child.ChildKey,
				ChildSN:   child.ChildSn,
				ChildType: child.ChildType,
				ChildName: child.ChildName,
				Timestamp: child.Timestamp,
				Online:    child.Online,
				Reported:  child.Reported,
				Metadata:  child.Metadata,
			})
		}
		input.History = append(input.History, reportsvc.CachedReportInput{
			ReportID:        item.ReportId,
			Timestamp:       item.Timestamp,
			Reported:        item.Reported,
			RunState:        item.RunState,
			Battery:         item.Battery,
			FirmwareVersion: item.FirmwareVersion,
			Online:          item.Online,
			IP:              item.IP,
			Children:        historyChildren,
		})
	}
	if _, err := h.Report.Ingest(ctx, input); err != nil {
		h.Log.Errorf("mqtt unified report sn=%s: %v", reg.MaskSN(deviceSN), err)
	}
}

func parseSNFromTopic(topic string) string {
	topic = strings.TrimSpace(topic)
	parts := strings.Split(topic, "/")
	if len(parts) < 3 {
		return ""
	}
	last := strings.ToLower(strings.TrimSpace(parts[len(parts)-1]))
	if last != "report" {
		return ""
	}
	return strings.TrimSpace(parts[len(parts)-2])
}
