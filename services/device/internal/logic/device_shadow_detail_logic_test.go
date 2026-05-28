package logic

import (
	"context"
	"strings"
	"testing"

	"github.com/jacklau/audio-ai-platform/services/device/internal/middleware/jwt"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
)

func TestDeviceShadowDetailHandler_MissingSN(t *testing.T) {
	ctx := jwt.WithUserID(context.Background(), 1)
	l := NewDeviceShadowDetailLogic(ctx, &svc.ServiceContext{})

	resp, err := l.GetDeviceShadowDetail("")
	if err == nil || !strings.Contains(err.Error(), "设备序列号") {
		t.Fatalf("GetDeviceShadowDetail(\"\") err = %v, want 设备序列号错误", err)
	}
	if resp != nil {
		t.Fatal("expected nil response for empty device_sn")
	}
}

func TestFormatAttributeLabel(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"power", "电源状态"},
		{"volume", "音量"},
		{"play_state", "播放状态"},
		{"battery", "电量"},
		{"temperature", "温度"},
		{"custom_field", "Custom Field"},
	}

	for _, tt := range tests {
		result := formatAttributeLabel(tt.input)
		if result != tt.expected {
			t.Errorf("formatAttributeLabel(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestFormatAttributeValue(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{nil, "-"},
		{"", "-"},
		{"hello", "hello"},
		{80.0, "80"},
		{3.14, "3.14"},
		{int64(100), "100"},
		{42, "42"},
		{true, "是"},
		{false, "否"},
	}

	for _, tt := range tests {
		result := formatAttributeValue(tt.input)
		if result != tt.expected {
			t.Errorf("formatAttributeValue(%v) = %s, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestGetAttributeType(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{nil, "null"},
		{"hello", "string"},
		{float64(1), "number"},
		{int64(1), "number"},
		{true, "boolean"},
		{[]interface{}{}, "array"},
		{map[string]interface{}{}, "object"},
	}

	for _, tt := range tests {
		result := getAttributeType(tt.input)
		if result != tt.expected {
			t.Errorf("getAttributeType(%v) = %s, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestCategorizeAttribute(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"power", "system"},
		{"battery", "system"},
		{"volume", "media"},
		{"play_state", "media"},
		{"custom_attr", "custom"},
	}

	for _, tt := range tests {
		result := categorizeAttribute(tt.input)
		if result != tt.expected {
			t.Errorf("categorizeAttribute(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestIsEqual(t *testing.T) {
	tests := []struct {
		a, b     interface{}
		expected bool
	}{
		{"hello", "hello", true},
		{"hello", "world", false},
		{1.0, 1.0, true},
		{1.0, 2.0, false},
		{int64(1), int64(1), true},
		{true, true, true},
		{false, true, false},
		{nil, nil, true},
		{nil, "", false},
	}

	for _, tt := range tests {
		result := isEqual(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("isEqual(%v, %v) = %v, want %v", tt.a, tt.b, result, tt.expected)
		}
	}
}
