package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jacklau/audio-ai-platform/services/device/internal/device/shadowv2"
)

func TestAdminShadowHandler_GetDeviceShadowDetail_MissingSN(t *testing.T) {
	handler := &AdminShadowHandler{}

	req := httptest.NewRequest(http.MethodGet, "/api/admin/device/shadow/detail", nil)
	w := httptest.NewRecorder()

	handler.GetDeviceShadowDetail(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestAdminShadowHandler_DeleteDeviceShadow_MissingSN(t *testing.T) {
	handler := &AdminShadowHandler{}

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/device/shadow", nil)
	w := httptest.NewRecorder()

	handler.DeleteDeviceShadow(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGetStatusText(t *testing.T) {
	tests := []struct {
		status   shadowv2.DeviceStatus
		expected string
	}{
		{shadowv2.StatusOnline, "在线"},
		{shadowv2.StatusOffline, "离线"},
		{shadowv2.StatusAbnormal, "异常"},
		{"unknown", "未知"},
	}

	for _, tt := range tests {
		result := getStatusText(tt.status)
		if result != tt.expected {
			t.Errorf("getStatusText(%s) = %s, want %s", tt.status, result, tt.expected)
		}
	}
}

func TestCalculateRate(t *testing.T) {
	tests := []struct {
		count, total int64
		expected     float64
	}{
		{50, 100, 50.0},
		{0, 100, 0.0},
		{100, 100, 100.0},
		{25, 50, 50.0},
		{0, 0, 0.0},
	}

	for _, tt := range tests {
		result := calculateRate(tt.count, tt.total)
		if result != tt.expected {
			t.Errorf("calculateRate(%d, %d) = %f, want %f", tt.count, tt.total, result, tt.expected)
		}
	}
}

func TestTrimSpace(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  hello  ", "hello"},
		{"\tworld\t", "world"},
		{"\nnewline\n", "newline"},
		{"no_spaces", "no_spaces"},
		{"   ", ""},
		{"", ""},
	}

	for _, tt := range tests {
		result := trimSpace(tt.input)
		if result != tt.expected {
			t.Errorf("trimSpace(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
