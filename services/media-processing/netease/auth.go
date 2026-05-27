package netease

import (
	"fmt"

	"github.com/zeromicro/go-zero/core/logx"
)

// GetQRCodeKey 获取二维码登录 Key
// 接口地址: /openapi/music/basic/user/oauth2/qrcodekey/get/v2
// 请求方式: POST
// 请求参数: {"type": 2, "expiredKey": 300}
func (c *Client) GetQRCodeKey() (*GetQRCodeKeyResp, error) {
	var resp GetQRCodeKeyResp

	// 构建请求参数
	req := &GetQRCodeKeyReq{
		Type:       2,   // 二维码类型：扫码登录
		ExpiredKey: 300, // 有效期：5分钟（300秒）
	}

	logx.Infof("[Netease Auth] 🎫 获取二维码 Key | Type: %d | ExpiredKey: %d", req.Type, req.ExpiredKey)

	// 调用 API
	err := c.Post("/openapi/music/basic/user/oauth2/qrcodekey/get/v2", req, &resp)
	if err != nil {
		return nil, fmt.Errorf("获取二维码失败: %w", err)
	}

	// 检查业务状态码
	if resp.Code != 200 && resp.Code != 0 {
		return nil, fmt.Errorf("获取二维码失败: code=%d, msg=%s", resp.Code, resp.Message)
	}

	logx.Infof("[Netease Auth] ✅ 获取二维码成功 | UniKey: %s | URL: %s",
		resp.Data.UniKey, resp.Data.URL)

	return &resp, nil
}

// CheckQRCodeStatus 检查二维码登录状态
// 接口地址: /openapi/music/basic/oauth2/device/login/qrcode/get
// 请求方式: POST
// 请求参数: {"key": "uniKey", "clientId": "AppID"}
func (c *Client) CheckQRCodeStatus(key string) (*CheckQRCodeStatusResp, error) {
	var resp CheckQRCodeStatusResp

	// 构建请求参数
	req := &CheckQRCodeStatusReq{
		Key:      key,          // 二维码唯一标识
		ClientID: c.GetAppID(), // 客户端 ID（使用 AppID）
	}

	logx.Infof("[Netease Auth] 🔍 查询二维码状态 | Key: %s | ClientID: %s", req.Key, req.ClientID)

	// 调用 API
	err := c.Post("/openapi/music/basic/oauth2/device/login/qrcode/get", req, &resp)
	if err != nil {
		return nil, fmt.Errorf("查询二维码状态失败: %w", err)
	}

	// 检查业务状态码
	if resp.Code != 200 && resp.Code != 0 {
		return nil, fmt.Errorf("查询二维码状态失败: code=%d, msg=%s", resp.Code, resp.Message)
	}

	// 根据状态码输出日志
	statusText, ok := QRCodeStatusText[resp.Data.Status]
	if !ok {
		statusText = fmt.Sprintf("未知状态(%d)", resp.Data.Status)
	}

	switch resp.Data.Status {
	case QRCodeStatusWaiting:
		logx.Infof("[Netease Auth] ⏳ 等待用户扫描二维码 | Status: %d (%s)", resp.Data.Status, statusText)
	case QRCodeStatusScanned:
		logx.Infof("[Netease Auth] 👀 用户已扫描，等待确认 | Status: %d (%s)", resp.Data.Status, statusText)
	case QRCodeStatusConfirmed:
		logx.Infof("\n====================================")
		logx.Infof("🎉 [Netease Auth] ✅✅✅ 登录成功! ✅✅✅")
		logx.Infof("====================================")
		logx.Infof("📋 用户信息:")
		logx.Infof("   👤 User ID: %d", resp.Data.UserID)
		logx.Infof("   😊 Nickname: %s", resp.Data.Nickname)
		logx.Infof("   🖼️  Avatar: %s", resp.Data.Avatar)
		logx.Infof("   🔑 Token: %s...", resp.Data.AccessToken[:min(20, len(resp.Data.AccessToken))])
		logx.Infof("   ⏱️  ExpiresIn: %d 秒", resp.Data.ExpiresIn)
		logx.Infof("====================================\n")
	case QRCodeStatusExpired:
		logx.Errorf("[Netease Auth] ⏰ 二维码已过期 | Status: %d (%s)", resp.Data.Status, statusText)
	case QRCodeStatusCancelled:
		logx.Errorf("[Netease Auth] ❌ 用户取消授权 | Status: %d (%s)", resp.Data.Status, statusText)
	default:
		logx.Infof("[Netease Auth] 📊 当前状态 | Status: %d (%s)", resp.Data.Status, statusText)
	}

	return &resp, nil
}

// min 辅助函数（Go 1.21+ 可用内置 min）
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
