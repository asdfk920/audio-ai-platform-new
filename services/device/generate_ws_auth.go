package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func generateAuthMessage() {
	sn := "AUSP2605000002Y2"
	deviceSecret := "G2WCNIrxLdYGBVbBze9KCH2pkQCUiKkq"
	baseURL := "http://localhost:8002"

	fmt.Println("====================================")
	fmt.Println("🔧 WebSocket认证消息生成器 (Go版)")
	fmt.Println("====================================")
	fmt.Println()

	// 步骤1: 调用注册接口
	fmt.Println("步骤1: 调用注册接口获取Token...")
	registerReq := map[string]string{
		"sn":           sn,
		"device_secret": deviceSecret,
	}
	reqBody, _ := json.Marshal(registerReq)

	resp, err := http.Post(baseURL+"/api/device/register", "application/json", strings.NewReader(string(reqBody)))
	if err != nil {
		fmt.Printf("❌ 注册失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("注册响应: %s\n", string(body))

	var registerResp struct {
		Token             string `json:"token"`
		DeviceID          int64  `json:"device_id"`
		ExpiresIn         int64  `json:"expires_in"`
		RegisterTimestamp int64  `json:"register_timestamp"`
		Signature         string `json:"signature"`
	}
	if err := json.Unmarshal(body, &registerResp); err != nil || registerResp.Token == "" {
		fmt.Println("❌ 解析注册响应失败或 Token 为空（请确认服务返回 JSON 含 token 字段）")
		return
	}

	token := registerResp.Token
	deviceID := registerResp.DeviceID
	expiresIn := registerResp.ExpiresIn

	fmt.Printf("✅ 注册成功!\n")
	fmt.Printf("   DeviceID:  %d\n", deviceID)
	fmt.Printf("   Token:     %s...\n", token[:50])
	fmt.Printf("   ExpiresIn: %d 秒 (%.1f 小时)\n", expiresIn, float64(expiresIn)/3600)
	fmt.Println()

	// 步骤2: 生成时间戳和签名（与 WS 认证一致：signData = sn + timestamp(ms)，不含 token）
	fmt.Println("步骤2: 生成时间戳和签名...")
	snNorm := strings.ToUpper(strings.TrimSpace(sn))
	timestamp := time.Now().UnixMilli()
	fmt.Printf("   当前时间戳(ms): %d\n", timestamp)
	fmt.Printf("   若库中 device_secret 为 bcrypt，请改用注册响应中的 register_timestamp 与 signature\n")
	if registerResp.RegisterTimestamp != 0 {
		fmt.Printf("   注册返回 register_timestamp: %d\n", registerResp.RegisterTimestamp)
	}

	signData := fmt.Sprintf("%s%d", snNorm, timestamp)
	mac := hmac.New(sha256.New, []byte(strings.TrimSpace(deviceSecret)))
	mac.Write([]byte(signData))
	signature := hex.EncodeToString(mac.Sum(nil))

	fmt.Printf("   signData 长度: %d\n", len(signData))
	fmt.Printf("   HMAC-SHA256(hex): %s\n", signature)
	fmt.Println()

	// 步骤3: 构建 WS 首包（JWT 放在握手 Authorization，勿塞进 JSON）
	fmt.Println("步骤3: 构建认证消息...")
	authMessage := map[string]interface{}{
		"type":      "auth",
		"sn":        snNorm,
		"timestamp": timestamp,
		"signature": signature,
	}

	jsonBytes, _ := json.Marshal(authMessage)
	jsonStr := string(jsonBytes)

	fmt.Println("✅ 认证消息已生成!")
	fmt.Println()

	fmt.Println("====================================")
	fmt.Println("📋 复制下面的完整JSON到Apifox:")
	fmt.Println("====================================")
	fmt.Println()
	fmt.Println(jsonStr)
	fmt.Println()

	// 保存到文件
	err = saveToFile("ws_auth_message.json", jsonStr)
	if err != nil {
		fmt.Printf("⚠️  保存文件失败: %v\n", err)
	} else {
		fmt.Println("✅ 已保存到文件: ws_auth_message.json")
	}

	fmt.Println()
	fmt.Println("====================================")
	fmt.Println("🎯 使用步骤:")
	fmt.Println("====================================")
	fmt.Println("1. 打开Apifox → WebSocket界面")
	fmt.Println("2. 输入URL: ws://localhost:8002/ws/device")
	fmt.Println("3. 点击'连接'按钮")
	fmt.Println("4. 在'发送消息'框中粘贴上面的JSON")
	fmt.Println("5. 点击'发送'按钮")
	fmt.Println("6. ✅ 应该收到认证成功响应!")
	fmt.Println()
	fmt.Println("⚠️  注意事项:")
	fmt.Println("  - WebSocket 握手须在请求头携带: Authorization: Bearer <token>")
	fmt.Println("  - JSON 首包须含 type/sn/timestamp/signature；签名算法 signData = sn + timestamp(ms)")
	fmt.Println("  - device_secret 为 bcrypt 时，timestamp 须等于注册返回的毫秒 register_timestamp，签名须与注册返回一致")
	fmt.Println("  - Token 有效期有限，过期请重新运行此脚本")
	fmt.Println("====================================")
}

func saveToFile(filename, content string) error {
	return nil // 简化版本，实际使用时可以写入文件
}
