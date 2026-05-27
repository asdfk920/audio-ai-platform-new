package logic

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/model"
	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/svc"
)

// SimulateQRCodeLoginReq 模拟扫码登录请求
type SimulateQRCodeLoginReq struct {
	Key         string `json:"key"`                // 二维码 Key（必填）
	UserID      int64  `json:"user_id"`            // 系统用户 ID（必填）
	OpenID      string `json:"open_id"`            // 第三方平台用户 ID（必填，如网易云音乐的 userId）
	Nickname    string `json:"nickname"`           // 昵称（必填）
	Avatar      string `json:"avatar"`             // 头像 URL
	Gender      int16  `json:"gender,omitempty"`   // 性别：0-未知，1-男，2-女
	Country     string `json:"country,omitempty"`  // 国家/地区代码
	Province    string `json:"province,omitempty"` // 省份
	City        string `json:"city,omitempty"`     // 城市
	AccessToken string `json:"access_token"`       // 访问令牌（模拟数据）
}

// SimulateQRCodeLoginResp 模拟扫码登录响应
type SimulateQRCodeLoginResp struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
	Data    struct {
		Key      string `json:"key"`
		Status   int    `json:"status"`
		UserID   int64  `json:"user_id"`
		Nickname string `json:"nickname"`
		Avatar   string `json:"avatar"`
		Message  string `json:"message"`
	} `json:"data"`
}

// SimulateQRCodeLoginLogic 模拟扫码登录的业务逻辑
type SimulateQRCodeLoginLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewSimulateQRCodeLoginLogic 创建模拟扫码登录逻辑实例
func NewSimulateQRCodeLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SimulateQRCodeLoginLogic {
	return &SimulateQRCodeLoginLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// SimulateQRCodeLogin 模拟扫码登录成功
// 1. 验证二维码 key 是否存在且未过期
// 2. 将第三方账号信息写入数据库
// 3. 更新内存中的会话状态为已确认（803）
func (l *SimulateQRCodeLoginLogic) SimulateQRCodeLogin(req *SimulateQRCodeLoginReq) (*SimulateQRCodeLoginResp, error) {
	logx.Infof("[Netease Auth] 🎭 开始模拟扫码登录 | Key: %s", req.Key)

	if req.Key == "" {
		return nil, fmt.Errorf("参数错误: key 不能为空")
	}
	if req.UserID <= 0 {
		return nil, fmt.Errorf("参数错误: user_id 必须大于 0")
	}
	if req.OpenID == "" {
		return nil, fmt.Errorf("参数错误: open_id 不能为空")
	}
	if req.Nickname == "" {
		return nil, fmt.Errorf("参数错误: nickname 不能为空")
	}

	qrcodeMutex.RLock()
	session, exists := qrcodeSessions[req.Key]
	qrcodeMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("二维码不存在或已失效")
	}

	now := time.Now()
	expiredDuration := now.Sub(session.CreatedAt)
	if expiredDuration > time.Duration(QRCodeExpireSeconds)*time.Second {
		return nil, fmt.Errorf("二维码已过期（%.1f 秒）", expiredDuration.Seconds())
	}

	rawDataMap := map[string]interface{}{
		"qrcode_key": req.Key,
		"open_id":    req.OpenID,
		"login_time": now.Format(time.RFC3339),
		"simulated":  true,
		"provider":   model.ProviderNetease,
	}

	rawDataBytes, err := json.Marshal(rawDataMap)
	if err != nil {
		return nil, fmt.Errorf("序列化原始数据失败: %w", err)
	}

	account := &model.ThirdPartyAccount{
		UserID:      req.UserID,
		Provider:    model.ProviderNetease,
		OpenID:      req.OpenID,
		Nickname:    req.Nickname,
		Avatar:      req.Avatar,
		Gender:      req.Gender,
		CountryCode: req.Country,
		Province:    req.Province,
		City:        req.City,
		AccessToken: req.AccessToken,
		RawData:     sql.NullString{String: string(rawDataBytes), Valid: true},
		Status:      model.ThirdPartyAccountStatusNormal,
		LastLoginAt: now,
	}

	err = l.svcCtx.ThirdPartyAccountRepo.Create(l.ctx, account)
	if err != nil {
		logx.Errorf("[Netease Auth] ❌ 写入第三方账号失败: %v", err)
		return nil, fmt.Errorf("保存账号信息失败: %w", err)
	}

	logx.Infof("[Netease Auth] ✅ 第三方账号已写入数据库 | AccountID: %d | UserID: %d | OpenID: %s",
		account.ID, account.UserID, account.OpenID)

	UpdateQRCodeStatus(req.Key, QRCodeStatusConfirmed, req.UserID, req.Nickname, req.Avatar)

	logx.Infof("\n====================================")
	logx.Infof("🎉 [Netease Auth] ✅✅✅ 模拟扫码登录成功! ✅✅✅")
	logx.Infof("====================================")
	logx.Infof("📋 账号信息:")
	logx.Infof("   🆔 Account ID: %d", account.ID)
	logx.Infof("   👤 User ID: %d", req.UserID)
	logx.Infof("   🔑 Open ID: %s", req.OpenID)
	logx.Infof("   😊 Nickname: %s", req.Nickname)
	logx.Infof("   🖼️  Avatar: %s", req.Avatar)
	logx.Infof("   📌 QRCode Key: %s", req.Key)
	logx.Infof("====================================")

	resp := &SimulateQRCodeLoginResp{
		Code:    200,
		Message: "success",
	}
	resp.Data.Key = req.Key
	resp.Data.Status = QRCodeStatusConfirmed
	resp.Data.UserID = req.UserID
	resp.Data.Nickname = req.Nickname
	resp.Data.Avatar = req.Avatar
	resp.Data.Message = "模拟登录成功，账号信息已保存到数据库"

	return resp, nil
}
