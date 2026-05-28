//go:build ignore

package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	TARGET_SERVER = "http://localhost:8004"
	PORT          = ":8080"
)

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Status  int         `json:"status"`
	Message string      `json:"message,omitempty"`
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	targetURL := TARGET_SERVER + r.URL.Path
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	log.Printf("[PROXY] %s %s → %s", r.Method, r.URL.Path, targetURL)

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DisableKeepAlives: false,
		},
	}

	var reqBody io.Reader
	if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
		reqBody = r.Body
	}

	proxyReq, err := http.NewRequest(r.Method, targetURL, reqBody)
	if err != nil {
		sendError(w, http.StatusInternalServerError, fmt.Sprintf("创建请求失败: %v", err))
		return
	}

	for key, values := range r.Header {
		if !strings.HasPrefix(strings.ToLower(key), "origin") &&
			!strings.HasPrefix(strings.ToLower(key), "access-control") {
			for _, value := range values {
				proxyReq.Header.Add(key, value)
			}
		}
	}

	resp, err := client.Do(proxyReq)
	if err != nil {
		sendError(w, http.StatusBadGateway, fmt.Sprintf("请求目标服务器失败: %v", err))
		return
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	log.Printf("[RESPONSE] %d %s", resp.StatusCode, targetURL)

	for key, values := range resp.Header {
		if strings.ToLower(key) != "transfer-encoding" {
			w.Header()[key] = values
		}
	}

	w.WriteHeader(resp.StatusCode)

	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Printf("[ERROR] 复制响应体失败: %v", err)
	}
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	response := APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"status":    "ok",
			"proxy_for": TARGET_SERVER,
			"timestamp": time.Now().Format(time.RFC3339),
			"endpoints": []string{
				"/api/v1/thirdparty/stream/*",
				"/api/v1/protocol/convert",
				"/api/v1/song/play",
				"/proxy/*",
				"/api/debug/token",
			},
		},
		Message: "✅ 流媒体代理服务器运行中",
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(response)
}

func tokenDebugHandler(w http.ResponseWriter, r *http.Request) {
	token := ""

	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		token = strings.TrimPrefix(authHeader, "Bearer ")
	}

	if token == "" {
		token = r.URL.Query().Get("token")
	}

	if token == "" {
		var req struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			token = req.Token
		}
	}

	if token == "" {
		sendError(w, http.StatusBadRequest, "请提供Token：通过Header Authorization、查询参数?token=或JSON body {token:'xxx'}")
		return
	}

	result := analyzeJWTToken(token)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	isValid := true
	if result.Valid != nil {
		isValid = *result.Valid
	}
	json.NewEncoder(w).Encode(APIResponse{
		Success: isValid,
		Data:    result,
		Message: "Token诊断完成",
	})
}

type TokenAnalysis struct {
	RawToken     string                 `json:"raw_token"`
	TokenLength  int                    `json:"token_length"`
	Valid        *bool                  `json:"valid,omitempty"`
	Header       map[string]interface{} `json:"header,omitempty"`
	Payload      map[string]interface{} `json:"payload,omitempty"`
	Error        string                 `json:"error,omitempty"`
	IssuedAt     string                 `json:"issued_at,omitempty"`
	ExpiresAt    string                 `json:"expires_at,omitempty"`
	IsExpired    bool                   `json:"is_expired"`
	ExpiresInSec int64                  `json:"expires_in_sec,omitempty"`
	DeviceID     float64                `json:"device_id,omitempty"`
	SN           string                 `json:"sn,omitempty"`
	Diagnostics  []string               `json:"diagnostics"`
}

func analyzeJWTToken(tokenString string) TokenAnalysis {
	result := TokenAnalysis{
		RawToken:    tokenString,
		TokenLength: len(tokenString),
		Diagnostics: make([]string, 0),
	}

	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		result.Error = "Token格式错误：JWT应有3部分（header.payload.signature），实际有" + fmt.Sprintf("%d", len(parts)) + "部分"
		result.Valid = boolPtr(false)
		result.Diagnostics = append(result.Diagnostics, "❌ 格式错误："+result.Error)
		return result
	}

	result.Diagnostics = append(result.Diagnostics, "✅ Token格式正确（3部分）")

	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		result.Diagnostics = append(result.Diagnostics, "⚠️ Header解码失败："+err.Error())
	} else {
		var header map[string]interface{}
		if json.Unmarshal(headerBytes, &header) == nil {
			result.Header = header
			result.Diagnostics = append(result.Diagnostics, fmt.Sprintf("✅ Header算法: %v", header["alg"]))
		}
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		result.Error = "Payload解码失败：" + err.Error()
		result.Valid = boolPtr(false)
		result.Diagnostics = append(result.Diagnostics, "❌ "+result.Error)
		return result
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		result.Error = "Payload解析失败：" + err.Error()
		result.Valid = boolPtr(false)
		result.Diagnostics = append(result.Diagnostics, "❌ "+result.Error)
		return result
	}

	result.Payload = payload

	if deviceID, ok := payload["device_id"].(float64); ok {
		result.DeviceID = deviceID
		result.Diagnostics = append(result.Diagnostics, fmt.Sprintf("✅ DeviceID: %.0f", deviceID))
	}

	if sn, ok := payload["sn"].(string); ok {
		result.SN = sn
		result.Diagnostics = append(result.Diagnostics, fmt.Sprintf("✅ SN: %s", sn))
	}

	if iat, ok := payload["iat"].(float64); ok {
		t := time.Unix(int64(iat), 0)
		result.IssuedAt = t.Format(time.RFC3339)
		result.Diagnostics = append(result.Diagnostics, fmt.Sprintf("📅 签发时间: %s", result.IssuedAt))
	}

	if exp, ok := payload["exp"].(float64); ok {
		t := time.Unix(int64(exp), 0)
		result.ExpiresAt = t.Format(time.RFC3339)
		now := time.Now()
		result.IsExpired = now.After(t)
		result.ExpiresInSec = int64(t.Sub(now).Seconds())

		if result.IsExpired {
			result.Valid = boolPtr(false)
			result.Diagnostics = append(result.Diagnostics, fmt.Sprintf("❌ Token已过期！过期时间: %s (距今%ds)", result.ExpiresAt, -result.ExpiresInSec))
		} else {
			result.Diagnostics = append(result.Diagnostics, fmt.Sprintf("✅ Token未过期，剩余有效期: %ds (%.1f小时)", result.ExpiresInSec, float64(result.ExpiresInSec)/3600))
		}
	}

	if nbf, ok := payload["nbf"].(float64); ok {
		t := time.Unix(int64(nbf), 0)
		now := time.Now()
		if now.Before(t) {
			result.Valid = boolPtr(false)
			result.Diagnostics = append(result.Diagnostics, fmt.Sprintf("❌ Token尚未生效！生效时间: %s", t.Format(time.RFC3339)))
		} else {
			result.Diagnostics = append(result.Diagnostics, fmt.Sprintf("✅ Token已生效 (nbf: %s)", t.Format(time.RFC3339)))
		}
	}

	if alg, ok := payload["alg"]; ok && alg != "HS256" {
		result.Diagnostics = append(result.Diagnostics, fmt.Sprintf("⚠️ 签名算法异常: %v (期望HS256)", alg))
	}

	if result.Valid == nil {
		result.Valid = boolPtr(true)
		result.Diagnostics = append(result.Diagnostics, "✅ Token结构完整，但需要服务端验证签名")
	}

	return result
}

func boolPtr(b bool) *bool {
	return &b
}

func proxyAPIHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/proxy/")
	if path == "" || path == "/" {
		sendError(w, http.StatusBadRequest, "请提供要代理的API路径，例如: /proxy/api/v1/thirdparty/stream/url?url=xxx")
		return
	}

	targetURL := TARGET_SERVER + "/" + path
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	log.Printf("[API PROXY] %s %s → %s", r.Method, r.URL.Path, targetURL)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	var body io.Reader
	if r.Body != nil {
		body = r.Body
	}

	req, err := http.NewRequest(r.Method, targetURL, body)
	if err != nil {
		sendError(w, http.StatusInternalServerError, fmt.Sprintf("创建请求失败: %v", err))
		return
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		sendError(w, http.StatusBadGateway, fmt.Sprintf("请求失败: %v", err))
		return
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		sendError(w, http.StatusInternalServerError, fmt.Sprintf("读取响应失败: %v", err))
		return
	}

	var result interface{}
	if err := json.Unmarshal(respData, &result); err != nil {
		result = string(respData)
	}

	apiResp := APIResponse{
		Success: resp.StatusCode >= 200 && resp.StatusCode < 300,
		Data:    result,
		Status:  resp.StatusCode,
	}

	if resp.StatusCode >= 400 {
		apiResp.Error = fmt.Sprintf("目标服务器返回错误: HTTP %d", resp.StatusCode)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(apiResp)

	log.Printf("[API RESPONSE] status=%d success=%v", resp.StatusCode, apiResp.Success)
}

func testStreamHandler(w http.ResponseWriter, r *http.Request) {
	streamURL := r.URL.Query().Get("url")
	if streamURL == "" {
		streamURL = "https://www.soundhelix.com/examples/mp3/SoundHelix-Song-1.mp3"
	}

	targetURL := fmt.Sprintf("%s/api/v1/thirdparty/stream/url?url=%s",
		TARGET_SERVER,
		url.QueryEscape(streamURL),
	)

	log.Printf("[TEST] 测试流代理: %s", streamURL)

	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest(http.MethodGet, targetURL, nil)

	token := r.Header.Get("Authorization")
	if token != "" {
		req.Header.Set("Authorization", token)
	}

	resp, err := client.Do(req)
	if err != nil {
		sendError(w, http.StatusBadGateway, fmt.Sprintf("请求失败: %v", err))
		return
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)

	testResult := map[string]interface{}{
		"test_result":   "success",
		"target_url":    targetURL,
		"source_stream": streamURL,
		"http_status":   resp.StatusCode,
		"response_body": json.RawMessage(data),
		"timestamp":     time.Now().Format(time.RFC3339),
		"cors_status":   "✅ 通过代理服务器，无CORS问题",
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(testResult)

	log.Printf("[TEST] ✅ 测试完成: HTTP %d", resp.StatusCode)
}

func sendError(w http.ResponseWriter, statusCode int, message string) {
	response := APIResponse{
		Success: false,
		Error:   message,
		Status:  statusCode,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)

	log.Printf("[ERROR] %d: %s", statusCode, message)
}

func main() {
	fmt.Println(`
╔═══════════════════════════════════════════════════════════╗
║                                                           ║
║   🎵 流媒体代理服务器 (Media Processing Proxy Server)     ║
║                                                           ║
║   📡 代理目标: http://localhost:8004                       ║
║   🌐 本地地址: http://localhost:8080                      ║
║   🚀 状态: 启动中...                                      ║
║                                                           ║
║   ✅ 解决所有 CORS 跨域问题                               ║
║   ✅ 支持所有流媒体接口                                   ║
║   ✅ 完整日志记录                                         ║
║   🔧 新增: JWT Token诊断工具                              ║
║                                                           ║
╚═══════════════════════════════════════════════════════════╝
`)

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" || r.URL.Path == "/media-console.html" {
			http.ServeFile(w, r, "media-console.html")
			return
		}
		corsMiddleware(proxyHandler)(w, r)
	})

	mux.HandleFunc("/health", corsMiddleware(healthHandler))
	mux.HandleFunc("/api/test-stream", corsMiddleware(testStreamHandler))
	mux.HandleFunc("/api/debug/token", corsMiddleware(tokenDebugHandler))
	mux.HandleFunc("/proxy/", corsMiddleware(proxyAPIHandler))

	server := &http.Server{
		Addr:         PORT,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	fmt.Printf("\n🚀 服务器启动成功!\n")
	fmt.Printf("   📍 地址: http://localhost%s\n", PORT)
	fmt.Printf("   🎯 代理: %s\n", TARGET_SERVER)
	fmt.Printf("   📄 控制台: http://localhost%s/media-console.html\n", PORT)
	fmt.Printf("   🔧 Token诊断: http://localhost%s/api/debug/token?token=YOUR_TOKEN\n\n", PORT)
	fmt.Printf("💡 提示:\n")
	fmt.Printf("   - 所有请求都会自动添加 CORS 头\n")
	fmt.Printf("   - 访问 /health 检查状态\n")
	fmt.Printf("   - 访问 /api/debug/token 诊断JWT Token\n")
	fmt.Printf("   - 访问 /proxy/api/v1/* 代理API请求\n")
	fmt.Printf("   - 按 Ctrl+C 停止服务器\n\n")

	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Println("========================================")

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("❌ 服务器启动失败: %v", err)
	}
}
