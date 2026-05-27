package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// fetchDownloadableContent 调用内容服务 GET /api/v1/content/:id（与登录用户相同 Authorization），解析标题、播放地址与是否允许下载。
func fetchDownloadableContent(ctx context.Context, baseURL, authorization string, contentID int64) (title, playURL string, canDownload bool, err error) {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" || contentID <= 0 {
		return "", "", false, fmt.Errorf("内容服务地址或 content_id 无效")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/api/v1/content/%d", base, contentID), nil)
	if err != nil {
		return "", "", false, err
	}
	authz := strings.TrimSpace(authorization)
	if authz != "" {
		if !strings.HasPrefix(strings.ToLower(authz), "bearer ") {
			authz = "Bearer " + authz
		}
		req.Header.Set("Authorization", authz)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", false, fmt.Errorf("请求内容服务失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", "", false, err
	}
	if resp.StatusCode != http.StatusOK {
		msg := strings.TrimSpace(string(body))
		if len(msg) > 200 {
			msg = msg[:200] + "…"
		}
		if msg == "" {
			msg = resp.Status
		}
		return "", "", false, fmt.Errorf("内容服务 HTTP %d: %s", resp.StatusCode, msg)
	}

	var env struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return "", "", false, fmt.Errorf("解析内容详情失败: %w", err)
	}
	if env.Code != http.StatusOK {
		msg := strings.TrimSpace(env.Msg)
		if msg == "" {
			msg = "内容服务返回非成功状态"
		}
		return "", "", false, fmt.Errorf("内容服务: %s", msg)
	}
	if len(env.Data) == 0 {
		return "", "", false, fmt.Errorf("内容详情 data 为空")
	}
	var data struct {
		Title      string `json:"title"`
		PlayURL    string `json:"play_url"`
		Permission struct {
			CanDownload bool `json:"can_download"`
		} `json:"permission"`
	}
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return "", "", false, fmt.Errorf("解析内容详情字段失败: %w", err)
	}
	title = strings.TrimSpace(data.Title)
	playURL = strings.TrimSpace(data.PlayURL)
	return title, playURL, data.Permission.CanDownload, nil
}
