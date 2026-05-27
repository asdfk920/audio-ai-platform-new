package netease

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Signer IOT 签名工具
type Signer struct {
	AppID     string
	AppSecret string
}

// NewSigner 创建新的签名器
func NewSigner(appID, appSecret string) *Signer {
	return &Signer{
		AppID:     appID,
		AppSecret: appSecret,
	}
}

// GenerateNonce 生成随机字符串（32字节）
func (s *Signer) GenerateNonce() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// GetTimestamp 获取当前时间戳（毫秒）
func (s *Signer) GetTimestamp() string {
	return fmt.Sprintf("%d", time.Now().UnixMilli())
}

// Sign 生成 HMAC-SHA256 签名
func (s *Signer) Sign(params map[string]string) string {
	// 1. 按照参数名 ASCII 字典序排序
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 2. 拼接排序后的参数
	var sortedParams []string
	for _, k := range keys {
		sortedParams = append(sortedParams, fmt.Sprintf("%s=%s", k, params[k]))
	}
	queryString := strings.Join(sortedParams, "&")

	// 3. 使用 AppSecret 进行 HMAC-SHA256 签名
	mac := hmac.New(sha256.New, []byte(s.AppSecret))
	mac.Write([]byte(queryString))

	// 4. 返回十六进制签名字符串
	return hex.EncodeToString(mac.Sum(nil))
}

// BuildCommonParams 构建 IOT 公共参数（用于请求头或查询参数）
func (s *Signer) BuildCommonParams() IOTCommonParams {
	timestamp := s.GetTimestamp()
	nonce := s.GenerateNonce()

	params := IOTCommonParams{
		AppID:     s.AppID,
		Timestamp: timestamp,
		Nonce:     nonce,
		Version:   "1.0",
	}

	// 构建待签名的参数字典
	signMap := map[string]string{
		"appId":     s.AppID,
		"timestamp": timestamp,
		"nonce":     nonce,
		"version":   "1.0",
	}

	// 生成签名
	params.Signature = s.Sign(signMap)

	return params
}

// BuildHeaders 构建带签名的 HTTP 请求头
func (s *Signer) BuildHeaders() map[string]string {
	commonParams := s.BuildCommonParams()

	return map[string]string{
		"X-App-Id":     commonParams.AppID,
		"X-Timestamp":  commonParams.Timestamp,
		"X-Nonce":      commonParams.Nonce,
		"X-Signature":  commonParams.Signature,
		"X-Version":    commonParams.Version,
		"Content-Type": "application/json",
		"Accept":       "application/json",
	}
}

// SignWithBody 对包含 Body 的请求进行签名
// 用于 POST/PUT 请求，需要将 Body 内容也加入签名
func (s *Signer) SignWithBody(body string) string {
	timestamp := s.GetTimestamp()
	nonce := s.GenerateNonce()

	// 构建待签名参数（包含 body）
	signMap := map[string]string{
		"appId":     s.AppID,
		"timestamp": timestamp,
		"nonce":     nonce,
		"version":   "1.0",
		"body":      body,
	}

	signature := s.Sign(signMap)

	// 返回格式：timestamp,nonce,signature
	return fmt.Sprintf("%s,%s,%s", timestamp, nonce, signature)
}

// VerifySignature 验证签名（可选，用于回调验证）
func (s *Signer) VerifySignature(receivedSignature string, params map[string]string) bool {
	expectedSign := s.Sign(params)
	return hmac.Equal([]byte(expectedSign), []byte(receivedSignature))
}
