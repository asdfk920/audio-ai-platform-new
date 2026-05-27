package logic

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/model"
	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/svc"
)

// GetQRCodeKeyLogic 获取二维码 Key 的业务逻辑
type GetQRCodeKeyLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewGetQRCodeKeyLogic 创建获取二维码 Key 逻辑实例
func NewGetQRCodeKeyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetQRCodeKeyLogic {
	return &GetQRCodeKeyLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// QRCodeKeyResp 二维码 Key 响应结构体
type QRCodeKeyResp struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
	Data    struct {
		Key string `json:"key"` // 二维码唯一标识，用于后续轮询登录状态
		URL string `json:"url"` // 二维码图片地址，前端可直接展示
	} `json:"data"`
}

// GetQRCodeKey 获取二维码 Key
// 生成唯一的二维码 key 和对应的 URL
// 前端可以使用 URL 展示二维码图片
// 后续使用 key 轮询登录状态
func (l *GetQRCodeKeyLogic) GetQRCodeKey() (*QRCodeKeyResp, error) {
	logx.Infof("[Netease Auth] 📱 开始获取二维码 Key...")

	key := generateQRCodeKey()

	qrURL := fmt.Sprintf("https://music.163.com/login?qrkey=%s&timestamp=%d", key, time.Now().Unix())

	logx.Infof("[Netease Auth] ✅ 二维码 Key 生成成功 | Key: %s | URL: %s", key, qrURL)

	RegisterQRCodeSession(key)

	rawDataMap := map[string]interface{}{
		"qrcode_key": key,
		"qr_url":     qrURL,
		"created_at": time.Now().Format(time.RFC3339),
		"status":     "waiting",
	}
	rawDataBytes, _ := json.Marshal(rawDataMap)

	account := &model.ThirdPartyAccount{
		UserID:    0,
		Provider:  model.ProviderNetease,
		OpenID:    "",
		Nickname:  "",
		Avatar:    "",
		RawData:   sql.NullString{String: string(rawDataBytes), Valid: true},
		Status:    model.ThirdPartyAccountStatusNormal,
		CreatedAt: time.Now(),
	}

	err := l.svcCtx.ThirdPartyAccountRepo.Create(l.ctx, account)
	if err != nil {
		logx.Errorf("[Netease Auth] ⚠️ 预写入数据库失败（不影响使用）: %v", err)
	} else {
		logx.Infof("[Netease Auth] 💾 二维码会话已预写入数据库 | AccountID: %d | Key: %s", account.ID, key)
	}

	resp := &QRCodeKeyResp{
		Code:    200,
		Message: "success",
		Data: struct {
			Key string `json:"key"`
			URL string `json:"url"`
		}{
			Key: key,
			URL: qrURL,
		},
	}

	return resp, nil
}

// generateQRCodeKey 生成唯一的二维码 Key
// 使用 crypto/rand 生成安全的随机字符串
func generateQRCodeKey() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		logx.Errorf("[Netease Auth] ❌ 生成随机 Key 失败: %v", err)
		return fallbackGenerateKey()
	}
	return hex.EncodeToString(bytes)
}

// fallbackGenerateKey 备用方案：使用时间戳+随机数生成 Key
func fallbackGenerateKey() string {
	return fmt.Sprintf("qr_%d_%s", time.Now().UnixNano(), generateRandomString(8))
}

// generateRandomString 生成指定长度的随机字符串
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	charsetLen := len(charset)
	b := make([]byte, length)
	for i := range b {
		n, _ := rand.Prime(rand.Reader, charsetLen)
		idx := int(n.Int64()) % charsetLen
		b[i] = charset[idx]
	}
	return string(b)
}
