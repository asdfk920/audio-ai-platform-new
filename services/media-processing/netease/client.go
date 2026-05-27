package netease

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// Client 网易云音乐 IoT API 客户端
type Client struct {
	baseURL    string
	httpClient *http.Client
	signer     *Signer
	appID      string
}

// NewClient 创建新的 API 客户端
func NewClient(baseURL, appID, appSecret string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			// 自动处理重定向（保持 Cookie）
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return nil // 允许所有重定向
			},
		},
		signer: NewSigner(appID, appSecret),
		appID:  appID,
	}
}

// request 发送 HTTP 请求（核心方法）
// method: GET/POST
// path: API 路径（如 /openapi/music/basic/user/oauth2/qrcodekey/get/v2）
// req: 请求参数（会被序列化为 JSON）
// resp: 响应结构体（会被反序列化）
func (c *Client) request(method, path string, req interface{}, resp interface{}) error {
	// 1. 构建完整 URL
	url := c.baseURL + path

	// 2. 序列化请求体
	var bodyBytes []byte
	var err error

	if req != nil {
		bodyBytes, err = json.Marshal(req)
		if err != nil {
			return fmt.Errorf("序列化请求参数失败: %w", err)
		}
	}

	logx.Infof("[Netease SDK] 📤 %s %s | Body: %s", method, path, string(bodyBytes))

	// 3. 创建 HTTP 请求
	var httpReq *http.Request

	if method == "GET" {
		httpReq, err = http.NewRequest(method, url, nil)
	} else {
		httpReq, err = http.NewRequest(method, url, bytes.NewBuffer(bodyBytes))
	}

	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	// 4. 设置 IOT 签名请求头
	headers := c.signer.BuildHeaders()
	for key, value := range headers {
		httpReq.Header.Set(key, value)
	}

	// 如果是 POST/PUT，需要重新签名（包含 body）
	if method == "POST" || method == "PUT" && len(bodyBytes) > 0 {
		signWithBody := c.signer.SignWithBody(string(bodyBytes))
		parts := strings.Split(signWithBody, ",")
		if len(parts) == 3 {
			httpReq.Header.Set("X-Timestamp", parts[0])
			httpReq.Header.Set("X-Nonce", parts[1])
			httpReq.Header.Set("X-Signature", parts[2])
		}
	}

	// 5. 发送请求
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer httpResp.Body.Close()

	// 6. 读取响应体
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	logx.Infof("[Netease SDK] 📥 %s %s | Status: %d | Response: %s",
		method, path, httpResp.StatusCode, string(respBody))

	// 7. 检查 HTTP 状态码
	if httpResp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP 错误: status=%d, body=%s", httpResp.StatusCode, string(respBody))
	}

	// 8. 反序列化响应
	err = json.Unmarshal(respBody, resp)
	if err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}

	return nil
}

// Get 发送 GET 请求
func (c *Client) Get(path string, resp interface{}) error {
	return c.request("GET", path, nil, resp)
}

// Post 发送 POST 请求
func (c *Client) Post(path string, req interface{}, resp interface{}) error {
	return c.request("POST", path, req, resp)
}

// GetAppID 获取 AppID
func (c *Client) GetAppID() string {
	return c.appID
}
