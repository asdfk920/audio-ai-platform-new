package logic

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/svc"
)

// QRCodeCallbackReq 二维码扫码回调请求（第三方平台调用）
type QRCodeCallbackReq struct {
	Key         string `json:"key"`                    // 二维码 Key（必填）
	UserID      int64  `json:"user_id"`                // 第三方平台用户 ID（必填）
	OpenID      string `json:"open_id"`                // OpenID/UnionID（必填）
	Nickname    string `json:"nickname"`               // 用户昵称（必填）
	Avatar      string `json:"avatar"`                 // 用户头像 URL
	Gender      int16  `json:"gender,omitempty"`       // 性别：0-未知，1-男，2-女
	Country     string `json:"country,omitempty"`      // 国家代码
	Province    string `json:"province,omitempty"`     // 省份
	City        string `json:"city,omitempty"`         // 城市
	AccessToken string `json:"access_token,omitempty"` // 访问令牌
}

// QRCodeCallbackResp 二维码扫码回调响应
type QRCodeCallbackResp struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
	Data    struct {
		AccountID int64  `json:"account_id"` // 数据库记录 ID
		Key       string `json:"key"`        // 二维码 Key
		UserID    int64  `json:"user_id"`    // 用户 ID
		Nickname  string `json:"nickname"`   // 昵称
		Status    string `json:"status"`     // 处理结果
	} `json:"data"`
}

// QRCodeCallbackLogic 二维码回调业务逻辑
type QRCodeCallbackLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewQRCodeCallbackLogic 创建回调逻辑实例
func NewQRCodeCallbackLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QRCodeCallbackLogic {
	return &QRCodeCallbackLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// QRCodeCallback 处理第三方平台回调
// 1. 验证参数
// 2. 根据 qrcode_key 查找数据库记录
// 3. 更新用户信息到数据库
// 4. 更新内存会话状态为已确认（803）
func (l *QRCodeCallbackLogic) QRCodeCallback(req *QRCodeCallbackReq) (*QRCodeCallbackResp, error) {
	logx.Infof("\n====================================")
	logx.Infof("📩 [Netease Auth] 收到第三方回调 | Key: %s", req.Key)
	logx.Infof("====================================")

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

	account, err := l.svcCtx.ThirdPartyAccountRepo.FindByQRCodeKey(l.ctx, req.Key)
	if err != nil {
		logx.Errorf("[Netease Auth] ❌ 查询二维码会话失败: %v", err)
		return nil, fmt.Errorf("查询二维码会话失败: %w", err)
	}

	if account == nil {
		logx.Errorf("[Netease Auth] ⚠️ 二维码 Key 不存在或已过期 | Key: %s", req.Key)
		return nil, fmt.Errorf("二维码不存在或已过期")
	}

	now := time.Now()

	rawDataMap := map[string]interface{}{
		"qrcode_key": req.Key,
		"open_id":    req.OpenID,
		"user_id":    req.UserID,
		"nickname":   req.Nickname,
		"avatar":     req.Avatar,
		"login_time": now.Format(time.RFC3339),
		"status":     "confirmed",
		"provider":   "netease",
	}
	rawDataBytes, _ := json.Marshal(rawDataMap)

	err = l.svcCtx.ThirdPartyAccountRepo.UpdateAccountInfo(
		l.ctx,
		account.ID,
		req.UserID,
		req.OpenID,
		req.Nickname,
		req.Avatar,
		req.Gender,
		req.Country,
		req.Province,
		req.City,
		req.AccessToken,
		sql.NullString{String: string(rawDataBytes), Valid: true},
	)

	if err != nil {
		logx.Errorf("[Netease Auth] ❌ 更新账号信息失败: %v", err)
		return nil, fmt.Errorf("更新账号信息失败: %w", err)
	}

	UpdateQRCodeStatus(req.Key, QRCodeStatusConfirmed, req.UserID, req.Nickname, req.Avatar)

	logx.Infof("\n✅✅✅ [Netease Auth] 回调处理成功! ✅✅✅")
	logx.Infof("====================================")
	logx.Infof("📋 账号信息:")
	logx.Infof("   🆔 Account ID: %d", account.ID)
	logx.Infof("   👤 User ID: %d", req.UserID)
	logx.Infof("   🔑 Open ID: %s", req.OpenID)
	logx.Infof("   😊 Nickname: %s", req.Nickname)
	logx.Infof("   🖼️  Avatar: %s", req.Avatar)
	logx.Infof("   📌 QRCode Key: %s", req.Key)
	logx.Infof("====================================\n")

	resp := &QRCodeCallbackResp{
		Code:    200,
		Message: "success",
	}
	resp.Data.AccountID = account.ID
	resp.Data.Key = req.Key
	resp.Data.UserID = req.UserID
	resp.Data.Nickname = req.Nickname
	resp.Data.Status = "confirmed"

	return resp, nil
}
