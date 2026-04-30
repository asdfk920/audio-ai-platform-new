package reportsvc

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/jacklau/audio-ai-platform/services/device/internal/config"
	"github.com/jacklau/audio-ai-platform/services/device/internal/repo"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
)

func TestShouldAdvanceCurrentState(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	older := now.Add(-time.Minute)
	newer := now.Add(time.Minute)

	if !shouldAdvanceCurrentState(nil, now) {
		t.Fatalf("nil current should accept incoming")
	}
	if shouldAdvanceCurrentState(&now, older) {
		t.Fatalf("older report should not override current state")
	}
	if !shouldAdvanceCurrentState(&now, now) {
		t.Fatalf("same timestamp should be idempotently accepted")
	}
	if !shouldAdvanceCurrentState(&now, newer) {
		t.Fatalf("newer report should advance current state")
	}
}

func TestNormalizeSingleReportAddsChildSummary(t *testing.T) {
	row := &repo.DeviceAuthRow{
		SN:              "SN0001",
		ProductKey:      "pk-a",
		Mac:             "AA:BB:CC",
		FirmwareVersion: "1.0.0",
		IP:              "10.0.0.1",
	}
	reported, _ := json.Marshal(map[string]interface{}{"temperature": 22})
	online := true
	battery := int32(15)
	report, err := normalizeSingleReport(
		"r1",
		1_700_000_100,
		reported,
		"normal",
		&battery,
		"",
		&online,
		"",
		[]ChildReportInput{
			{
				ChildKey: "c1",
				Online:   &online,
				Reported: json.RawMessage(`{"power":"on"}`),
			},
			{
				ChildKey: "c2",
				Reported: json.RawMessage(`{"status":"offline"}`),
			},
		},
		false,
		"http",
		row,
	)
	if err != nil {
		t.Fatalf("normalizeSingleReport returned error: %v", err)
	}
	if got := report.Reported["child_total"]; got != 2 {
		t.Fatalf("child_total = %v, want 2", got)
	}
	if got := report.Reported["child_online"]; got != 1 {
		t.Fatalf("child_online = %v, want 1", got)
	}
	if got := report.Reported["child_offline"]; got != 1 {
		t.Fatalf("child_offline = %v, want 1", got)
	}
	if got := report.Reported["product_key"]; got != "pk-a" {
		t.Fatalf("product_key = %v", got)
	}
	if got := report.Reported["battery"]; got != battery {
		t.Fatalf("battery = %v, want %v", got, battery)
	}
	s := &Service{svcCtx: &svc.ServiceContext{Config: config.Config{}}}
	if next := s.resolveNextIntervalSeconds([]normalizedReport{report}); next != 300 {
		t.Fatalf("resolveNextIntervalSeconds(low battery) = %d, want 300", next)
	}
	report.Reported["power"] = map[string]interface{}{"charging_state": "charging"}
	if next := s.resolveNextIntervalSeconds([]normalizedReport{report}); next != 60 {
		t.Fatalf("resolveNextIntervalSeconds(charging) = %d, want 60", next)
	}
}
