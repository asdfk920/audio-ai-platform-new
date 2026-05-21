package logic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// postDeviceSvc 调用设备服务 JSON API（形如 { code, msg, data }）。
func postDeviceSvc(ctx context.Context, baseURL string, authorization string, apiPath string, payload any, dataInto any) error {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return fmt.Errorf("DeviceService.BaseURL 未配置")
	}
	path := "/" + strings.TrimLeft(apiPath, "/")
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化请求体失败: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+path, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("构造请求失败: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	auth := strings.TrimSpace(authorization)
	if auth != "" {
		httpReq.Header.Set("Authorization", auth)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("请求设备服务失败: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取设备服务响应失败: %w", err)
	}

	var envelope struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(bodyBytes, &envelope); err != nil {
		return fmt.Errorf("设备服务返回非 JSON: %s", string(bodyBytes))
	}

	if envelope.Code != http.StatusOK {
		if envelope.Msg != "" {
			return fmt.Errorf("%s", envelope.Msg)
		}
		return fmt.Errorf("设备服务错误 code=%d", envelope.Code)
	}

	if dataInto != nil && len(envelope.Data) > 0 && string(envelope.Data) != "null" {
		if err := json.Unmarshal(envelope.Data, dataInto); err != nil {
			return fmt.Errorf("解析 data 字段失败: %w", err)
		}
	}

	return nil
}

// mapForwardedDeviceInstructionStatus 将设备服务下发的 status（delivered/queued/cached）映射为点播 API 一贯的 data.status。
func mapForwardedDeviceInstructionStatus(remote string) string {
	switch strings.ToLower(strings.TrimSpace(remote)) {
	case "cached":
		return "offline"
	case "dispatched", "delivered", "queued", "queued_mq":
		return "success"
	default:
		return "success"
	}
}
