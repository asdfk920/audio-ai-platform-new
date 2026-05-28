//go:build ignore

package main

import (
	"fmt"
	"time"

	"github.com/jacklau/audio-ai-platform/services/media-processing/netease"
)

func main() {
	fmt.Println("====================================")
	fmt.Println("🎵 网易云音乐 SDK 测试")
	fmt.Println("====================================")

	client := netease.NewClient("http://localhost:8001", "your-app-id", "your-app-secret")

	fmt.Println("\n📱 1. 获取二维码 Key...")
	qrResp, err := client.GetQRCodeKey()
	if err != nil {
		fmt.Printf("❌ 获取二维码失败: %v\n", err)
		return
	}

	fmt.Printf("✅ 二维码获取成功!\n")
	fmt.Printf("   📌 Key: %s\n", qrResp.Data.UniKey)
	fmt.Printf("   🖼️  URL: %s\n", qrResp.Data.URL)
	fmt.Println("\n💡 请使用网易云音乐 App 扫描上面的二维码...")

	key := qrResp.Data.UniKey

	fmt.Println("\n⏳ 2. 开始轮询登录状态...")
	maxAttempts := 30 // 最多轮询 30 次（约 150 秒）
	for i := 0; i < maxAttempts; i++ {
		time.Sleep(5 * time.Second) // 每 5 秒查询一次

		statusResp, err := client.CheckQRCodeStatus(key)
		if err != nil {
			fmt.Printf("⚠️  查询状态失败 (第%d次): %v\n", i+1, err)
			continue
		}

		switch statusResp.Data.Status {
		case 800:
			fmt.Println("❌ 二维码已过期，请重新获取")
			return
		case 801:
			fmt.Printf("🔄 等待扫码... (第%d次)\n", i+1)
		case 802:
			fmt.Println("✅ 已扫码，等待确认...")
		case 803:
			fmt.Println("\n🎉 登录成功!")
			fmt.Printf("   👤 用户ID: %d\n", statusResp.Data.UserID)
			fmt.Printf("   😊 昵称: %s\n", statusResp.Data.Nickname)
			fmt.Printf("   🖼️  头像: %s\n", statusResp.Data.Avatar)
			return
		case 805:
			fmt.Println("❌ 用户取消授权")
			return
		default:
			fmt.Printf("❓ 未知状态: %d\n", statusResp.Data.Status)
		}
	}

	fmt.Println("\n⏰ 轮询超时，请重试")
}
