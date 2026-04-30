package shadowsvc

import (
	"encoding/json"
	"testing"
)

func TestComputeJSONDelta(t *testing.T) {
	desired := map[string]interface{}{
		"power":       "on",
		"temperature": float64(25),
		"mode":        "cool",
	}
	reported := map[string]interface{}{
		"power":       "on",
		"temperature": float64(22),
		"mode":        "cool",
	}

	delta := computeJSONDelta(desired, reported)
	if len(delta) != 1 {
		t.Fatalf("expected 1 delta field, got %d: %#v", len(delta), delta)
	}
	if got, ok := delta["temperature"].(float64); !ok || got != 25 {
		t.Fatalf("unexpected delta temperature: %#v", delta["temperature"])
	}
}

func TestMergeMapsNested(t *testing.T) {
	original := map[string]interface{}{
		"network": map[string]interface{}{
			"ip":   "1.1.1.1",
			"ssid": "old",
		},
		"power": "off",
	}
	patch := map[string]interface{}{
		"network": map[string]interface{}{
			"ssid": "new",
		},
		"power": "on",
	}

	got := mergeMaps(original, patch)
	network, ok := got["network"].(map[string]interface{})
	if !ok {
		t.Fatalf("network should remain nested map: %#v", got["network"])
	}
	if network["ip"] != "1.1.1.1" || network["ssid"] != "new" {
		t.Fatalf("unexpected merged network: %#v", network)
	}
	if got["power"] != "on" {
		t.Fatalf("unexpected power: %#v", got["power"])
	}
}

func TestMergeMetadataSectionNested(t *testing.T) {
	existing := map[string]interface{}{
		"power": map[string]interface{}{"timestamp": float64(100)},
		"network": map[string]interface{}{
			"ip": map[string]interface{}{"timestamp": float64(101)},
		},
	}
	patch := map[string]interface{}{
		"power": "on",
		"network": map[string]interface{}{
			"ssid": "wifi-1",
		},
	}

	got := mergeMetadataSection(existing, patch, 200)

	power, ok := got["power"].(map[string]interface{})
	if !ok || power["timestamp"] != int64(200) {
		t.Fatalf("unexpected power metadata: %#v", got["power"])
	}

	network, ok := got["network"].(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected network metadata: %#v", got["network"])
	}
	if _, ok := network["ip"]; !ok {
		t.Fatalf("expected existing nested ip metadata to be preserved: %#v", network)
	}
	ssid, ok := network["ssid"].(map[string]interface{})
	if !ok || ssid["timestamp"] != int64(200) {
		t.Fatalf("unexpected ssid metadata: %#v", network["ssid"])
	}
}

func TestAggregateReportedPrefersRedisJSONAndScalarFields(t *testing.T) {
	row := &shadowRow{
		Reported: mustJSON(map[string]interface{}{
			"power":       "off",
			"temperature": float64(20),
		}),
	}
	redisMap := map[string]string{
		"reported_json":     `{"temperature":25,"mode":"cool"}`,
		"firmware_version":  "1.0.3",
		"run_state":         "normal",
		"battery":           "88",
		"online":            "1",
		"product_key":       "pk-1",
		"mac":               "AA:BB",
		"ip":                "192.168.1.2",
	}

	got := aggregateReported(row, redisMap)
	if got["power"] != "off" {
		t.Fatalf("expected db reported field to remain, got %#v", got["power"])
	}
	if got["mode"] != "cool" {
		t.Fatalf("expected redis reported_json merge, got %#v", got["mode"])
	}
	if got["firmware_version"] != "1.0.3" {
		t.Fatalf("expected scalar firmware merge, got %#v", got["firmware_version"])
	}
	if got["run_state"] != "normal" {
		t.Fatalf("expected scalar run_state merge, got %#v", got["run_state"])
	}
	if got["online"] != true {
		t.Fatalf("expected online=true, got %#v", got["online"])
	}
}

func TestDecodeJSONObjectRejectsArray(t *testing.T) {
	_, err := decodeJSONObject(json.RawMessage(`[]`))
	if err == nil {
		t.Fatal("expected object decode to fail for array")
	}
}

func TestComputeJSONDeltaEmptyWhenMatched(t *testing.T) {
	desired := map[string]interface{}{
		"power": "on",
		"mode":  "cool",
	}
	reported := map[string]interface{}{
		"power": "on",
		"mode":  "cool",
	}

	delta := computeJSONDelta(desired, reported)
	if len(delta) != 0 {
		t.Fatalf("expected empty delta, got %#v", delta)
	}
}

func TestMergeMetadataSectionOverridesLeafTimestamp(t *testing.T) {
	existing := map[string]interface{}{
		"temperature": map[string]interface{}{"timestamp": float64(100)},
	}
	patch := map[string]interface{}{
		"temperature": float64(26),
	}

	got := mergeMetadataSection(existing, patch, 300)
	temperature, ok := got["temperature"].(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected metadata node: %#v", got["temperature"])
	}
	if temperature["timestamp"] != int64(300) {
		t.Fatalf("expected timestamp overwrite to 300, got %#v", temperature["timestamp"])
	}
}
