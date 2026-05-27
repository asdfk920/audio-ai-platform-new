package netease

import "time"

// ============================================
// IOT 公共参数（请求头）
// ============================================

// IOTCommonParams IOT 公共参数结构
type IOTCommonParams struct {
	AppID     string `json:"appId"`     // 应用 ID
	Timestamp string `json:"timestamp"` // 时间戳（毫秒）
	Nonce     string `json:"nonce"`     // 随机字符串
	Signature string `json:"signature"` // 签名
	Version   string `json:"version"`   // API 版本号
}

// ============================================
// 获取二维码接口
// ============================================

// GetQRCodeKeyReq 获取二维码请求参数
type GetQRCodeKeyReq struct {
	Type       int `json:"type"`       // 二维码类型：2（扫码登录）
	ExpiredKey int `json:"expiredKey"` // 有效期（秒）：300（5分钟）
}

// GetQRCodeKeyResp 获取二维码响应
type GetQRCodeKeyResp struct {
	Code    int    `json:"code"` // 状态码：200 成功
	Message string `json:"msg"`  // 消息
	Data    struct {
		UniKey string `json:"uniKey"` // 唯一标识 Key（用于轮询）
		URL    string `json:"url"`    // 二维码 URL 或内容
	} `json:"data"`
}

// ============================================
// 轮询二维码状态接口
// ============================================

// CheckQRCodeStatusReq 轮询状态请求参数
type CheckQRCodeStatusReq struct {
	Key      string `json:"key"`      // 二维码唯一标识（uniKey）
	ClientID string `json:"clientId"` // 客户端 ID（AppID）
}

// CheckQRCodeStatusResp 轮询状态响应
type CheckQRCodeStatusResp struct {
	Code    int    `json:"code"` // 状态码：200 成功
	Message string `json:"msg"`  // 消息
	Data    struct {
		Status       int    `json:"status"`       // 登录状态码
		UserID       int64  `json:"userId"`       // 用户 ID（status=803 时返回）
		Nickname     string `json:"nickname"`     // 用户昵称（status=803 时返回）
		Avatar       string `json:"avatar"`       // 用户头像 URL（status=803 时返回）
		AccessToken  string `json:"accessToken"`  // 访问令牌（status=803 时返回）
		RefreshToken string `json:"refreshToken"` // 刷新令牌（status=803 时返回）
		ExpiresIn    int64  `json:"expiresIn"`    // 令牌过期时间（秒）（status=803 时返回）
	} `json:"data"`
}

// ============================================
// 二维码状态常量
// ============================================

const (
	QRCodeStatusWaiting   = 801 // 等待扫码
	QRCodeStatusScanned   = 802 // 已扫码，待确认
	QRCodeStatusConfirmed = 803 // 授权登录成功
	QRCodeStatusExpired   = 800 // 二维码过期
	QRCodeStatusCancelled = 805 // 已取消授权
)

// QRCodeStatusText 状态码对应的文本描述
var QRCodeStatusText = map[int]string{
	QRCodeStatusWaiting:   "等待扫码",
	QRCodeStatusScanned:   "已扫码，待确认",
	QRCodeStatusConfirmed: "授权登录成功",
	QRCodeStatusExpired:   "二维码已过期",
	QRCodeStatusCancelled: "已取消授权",
}

// ============================================
// 用户信息模型
// ============================================

// UserInfo 用户信息
type UserInfo struct {
	UserID       int64     `json:"userId"`
	Nickname     string    `json:"nickname"`
	Avatar       string    `json:"avatar"`
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	ExpiresIn    int64     `json:"expiresIn"`
	LoginTime    time.Time `json:"loginTime"`
}

// ============================================
// 错误响应
// ============================================

// APIError API 错误响应
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
}

func (e *APIError) Error() string {
	return e.Message
}
