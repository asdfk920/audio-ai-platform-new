package logic

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/svc"
)

// QRCodeStatus 二维码登录状态常量
const (
	QRCodeStatusExpired   = 800 // 二维码过期
	QRCodeStatusWaiting   = 801 // 等待扫码
	QRCodeStatusScanned   = 802 // 已扫码，待确认
	QRCodeStatusConfirmed = 803 // 授权登录成功
	QRCodeStatusCancelled = 805 // 已取消授权

	// QRCodeExpireSeconds 二维码有效期（秒）
	QRCodeExpireSeconds = 300 // 5 分钟
)

// QRCodeSession 二维码会话信息
type QRCodeSession struct {
	Key       string    `json:"key"`        // 二维码 Key
	Status    int       `json:"status"`     // 当前状态
	CreatedAt time.Time `json:"created_at"` // 创建时间
	UpdatedAt time.Time `json:"updated_at"` // 最后更新时间
	UserID    int64     `json:"user_id"`    // 用户 ID（仅 status=803）
	Nickname  string    `json:"nickname"`   // 用户昵称（仅 status=803）
	Avatar    string    `json:"avatar"`     // 用户头像（仅 status=803）
}

// qrcodeStore 全局二维码会话存储（内存）
var (
	qrcodeSessions = make(map[string]*QRCodeSession)
	qrcodeMutex    sync.RWMutex
)

// RegisterQRCodeSession 注册新的二维码会话
func RegisterQRCodeSession(key string) {
	qrcodeMutex.Lock()
	defer qrcodeMutex.Unlock()

	qrcodeSessions[key] = &QRCodeSession{
		Key:       key,
		Status:    QRCodeStatusWaiting,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	logx.Infof("[Netease Auth] 📝 注册二维码会话 | Key: %s", key)
}

// UpdateQRCodeStatus 更新二维码状态（用于测试或外部触发）
func UpdateQRCodeStatus(key string, status int, userID int64, nickname, avatar string) {
	qrcodeMutex.Lock()
	defer qrcodeMutex.Unlock()

	if session, ok := qrcodeSessions[key]; ok {
		session.Status = status
		session.UpdatedAt = time.Now()
		if status == QRCodeStatusConfirmed {
			session.UserID = userID
			session.Nickname = nickname
			session.Avatar = avatar
		}
		logx.Infof("[Netease Auth] 🔄 更新二维码状态 | Key: %s | Status: %d", key, status)
	}
}

// CheckQRCodeStatusLogic 检查二维码状态的业务逻辑
type CheckQRCodeStatusLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewCheckQRCodeStatusLogic 创建检查二维码状态逻辑实例
func NewCheckQRCodeStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CheckQRCodeStatusLogic {
	return &CheckQRCodeStatusLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// QRCodeStatusResp 二维码状态响应结构体
type QRCodeStatusResp struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
	Data    struct {
		Status      int    `json:"status"`       // 登录状态：800/801/802/803/805
		UserID      int64  `json:"user_id"`      // 用户 ID（status=803）
		Nickname    string `json:"nickname"`     // 用户昵称（status=803）
		Avatar      string `json:"avatar"`       // 用户头像（status=803）
		AccessToken string `json:"access_token"` // 访问令牌（status=803，核心字段）
	} `json:"data"`
}

// CheckQRCodeStatus 检查二维码登录状态
// 根据传入的 key 返回当前二维码的扫描状态
// 如果配置了网易云音乐客户端，会调用真实 API 查询状态
func (l *CheckQRCodeStatusLogic) CheckQRCodeStatus(key string) (*QRCodeStatusResp, error) {
	logx.Infof("[Netease Auth] 🔍 查询二维码状态 | Key: %s", key)

	qrcodeMutex.RLock()
	session, exists := qrcodeSessions[key]
	qrcodeMutex.RUnlock()

	resp := &QRCodeStatusResp{}

	if !exists {
		logx.Errorf("[Netease Auth] ⚠️ 二维码 Key 不存在 | Key: %s", key)
		resp.Code = 400
		resp.Message = "二维码不存在或已失效"
		resp.Data.Status = QRCodeStatusExpired
		return resp, nil
	}

	now := time.Now()
	expiredDuration := now.Sub(session.CreatedAt)

	if expiredDuration > time.Duration(QRCodeExpireSeconds)*time.Second {
		logx.Errorf("[Netease Auth] ⏰ 二维码已过期 | Key: %s | 已存在: %.1f 秒", key, expiredDuration.Seconds())

		qrcodeMutex.Lock()
		delete(qrcodeSessions, key)
		qrcodeMutex.Unlock()

		resp.Code = 200
		resp.Message = "success"
		resp.Data.Status = QRCodeStatusExpired
		return resp, nil
	}

	logx.Infof("[Netease Auth] ✅ 二维码状态查询成功 | Key: %s | 内存状态: %d | 已存在: %.1f 秒",
		key, session.Status, expiredDuration.Seconds())

	resp.Code = 200
	resp.Message = "success"

	// 优先调用网易云音乐真实 API 查询状态
	if l.svcCtx.NeteaseClient != nil {
		logx.Infof("[Netease Auth] 🔍 调用网易云音乐 API 查询真实状态 | Key: %s", key)

		neteaseResp, err := l.svcCtx.NeteaseClient.CheckQRCodeStatus(key)
		if err != nil {
			logx.Errorf("[Netease Auth] ❌ 调用网易云音乐 API 失败: %v", err)
			resp.Data.Status = session.Status
			return resp, nil
		}

		logx.Infof("[Netease Auth] 📊 网易云音乐返回真实状态 | Status: %d", neteaseResp.Data.Status)

		// 用户已授权登录成功
		if neteaseResp.Data.Status == QRCodeStatusConfirmed {
			logx.Infof("\n====================================")
			logx.Infof("🎉 [Netease Auth] ✅✅✅ 登录成功! ✅✅✅")
			logx.Infof("====================================")
			logx.Infof("📋 网易云音乐返回的完整用户信息:")
			logx.Infof("   👤 User ID: %d", neteaseResp.Data.UserID)
			logx.Infof("   😊 Nickname: %s", neteaseResp.Data.Nickname)
			logx.Infof("   🖼️  Avatar: %s", neteaseResp.Data.Avatar)
			logx.Infof("   🔑 AccessToken: %s...", neteaseResp.Data.AccessToken[:min(20, len(neteaseResp.Data.AccessToken))])
			logx.Infof("   🔄 RefreshToken: %s...", neteaseResp.Data.RefreshToken[:min(20, len(neteaseResp.Data.RefreshToken))])
			logx.Infof("   ⏱️  ExpiresIn: %d 秒", neteaseResp.Data.ExpiresIn)
			logx.Infof("====================================\n")

			// 查找数据库中的预写入记录
			account, _ := l.svcCtx.ThirdPartyAccountRepo.FindByQRCodeKey(l.ctx, key)

			var accountID int64
			if account != nil {
				accountID = account.ID

				// 构建完整的用户信息 JSON（包含所有 token 信息）
				rawDataMap := map[string]interface{}{
					"qrcode_key":    key,
					"open_id":       fmt.Sprintf("%d", neteaseResp.Data.UserID),
					"user_id":       neteaseResp.Data.UserID,
					"nickname":      neteaseResp.Data.Nickname,
					"avatar":        neteaseResp.Data.Avatar,
					"access_token":  neteaseResp.Data.AccessToken,
					"refresh_token": neteaseResp.Data.RefreshToken,
					"expires_in":    neteaseResp.Data.ExpiresIn,
					"login_time":    time.Now().Format(time.RFC3339),
					"status":        "confirmed",
					"provider":      "netease",
					"source":        "real_api_polling",
				}
				rawDataBytes, _ := json.Marshal(rawDataMap)

				// 更新数据库记录
				err := l.svcCtx.ThirdPartyAccountRepo.UpdateAccountInfo(
					l.ctx,
					account.ID,
					neteaseResp.Data.UserID,
					fmt.Sprintf("%d", neteaseResp.Data.UserID),
					neteaseResp.Data.Nickname,
					neteaseResp.Data.Avatar,
					0, "", "", "",
					neteaseResp.Data.AccessToken, // 保存 access_token 到数据库
					sql.NullString{String: string(rawDataBytes), Valid: true},
				)
				if err != nil {
					logx.Errorf("[Netease Auth] ⚠️ 更新数据库失败（不影响返回）: %v", err)
				} else {
					logx.Infof("[Netease Auth] 💾 用户信息已保存到数据库 | AccountID: %d", accountID)
				}

				// 返回数据库中的完整信息（包括 access_token）
				resp.Data.Status = QRCodeStatusConfirmed
				resp.Data.UserID = account.UserID
				resp.Data.Nickname = account.Nickname
				resp.Data.Avatar = account.Avatar
				resp.Data.AccessToken = account.AccessToken

				// 同步内存缓存
				session.Status = QRCodeStatusConfirmed
				session.UserID = account.UserID
				session.Nickname = account.Nickname
				session.Avatar = account.Avatar

				return resp, nil
			} else {
				// 数据库中没有预写入的记录，直接使用网易云音乐返回的信息
				logx.Errorf("[Netease Auth] ⚠️ 数据库中未找到预写入的记录，仅返回网易云音乐数据")

				resp.Data.Status = QRCodeStatusConfirmed
				resp.Data.UserID = neteaseResp.Data.UserID
				resp.Data.Nickname = neteaseResp.Data.Nickname
				resp.Data.Avatar = neteaseResp.Data.Avatar

				session.Status = QRCodeStatusConfirmed
				session.UserID = neteaseResp.Data.UserID
				session.Nickname = neteaseResp.Data.Nickname
				session.Avatar = neteaseResp.Data.Avatar

				return resp, nil
			}
		}

		// 其他状态的日志和返回
		if neteaseResp.Data.Status == QRCodeStatusScanned {
			logx.Infof("[Netease Auth] 👀 用户已扫描二维码，等待确认...")
		} else if neteaseResp.Data.Status == QRCodeStatusWaiting {
			logx.Infof("[Netease Auth] ⏳ 等待用户扫描二维码...")
		} else if neteaseResp.Data.Status == QRCodeStatusCancelled {
			logx.Infof("[Netease Auth] ❌ 用户取消了授权")
		} else if neteaseResp.Data.Status == QRCodeStatusExpired {
			logx.Infof("[Netease Auth] ⏰ 网易云音乐端二维码已过期")

			qrcodeMutex.Lock()
			delete(qrcodeSessions, key)
			qrcodeMutex.Unlock()

			resp.Data.Status = QRCodeStatusExpired
			return resp, nil
		}

		// 返回网易云音乐的真实状态
		resp.Data.Status = neteaseResp.Data.Status
		return resp, nil
	}

	// 未配置网易云音乐客户端，使用本地数据库缓存
	logx.Errorf("[Netease Auth] ⚠️ 未配置网易云音乐客户端，使用本地缓存状态")

	account, err := l.svcCtx.ThirdPartyAccountRepo.FindByQRCodeKey(l.ctx, key)
	if err != nil {
		logx.Errorf("[Netease Auth] ❌ 查询数据库失败，使用内存状态: %v", err)
		resp.Data.Status = session.Status
		if session.Status == QRCodeStatusConfirmed {
			resp.Data.UserID = session.UserID
			resp.Data.Nickname = session.Nickname
			resp.Data.Avatar = session.Avatar
		}
		return resp, nil
	}

	if account != nil && account.UserID > 0 {
		tokenPreview := account.AccessToken
		if len(tokenPreview) > 20 {
			tokenPreview = tokenPreview[:20] + "..."
		}
		logx.Infof("[Netease Auth] 📋 从数据库获取到账号信息 | UserID: %d | Nickname: %s | OpenID: %s | Token: %s",
			account.UserID, account.Nickname, account.OpenID, tokenPreview)

		resp.Data.Status = QRCodeStatusConfirmed
		resp.Data.UserID = account.UserID
		resp.Data.Nickname = account.Nickname
		resp.Data.Avatar = account.Avatar
		resp.Data.AccessToken = account.AccessToken

		session.Status = QRCodeStatusConfirmed
		session.UserID = account.UserID
		session.Nickname = account.Nickname
		session.Avatar = account.Avatar

		err = l.svcCtx.ThirdPartyAccountRepo.UpdateLastLogin(l.ctx, account.ID)
		if err != nil {
			logx.Errorf("[Netease Auth] ⚠️ 更新最后登录时间失败: %v", err)
		}
	} else {
		resp.Data.Status = session.Status
		if session.Status == QRCodeStatusConfirmed {
			resp.Data.UserID = session.UserID
			resp.Data.Nickname = session.Nickname
			resp.Data.Avatar = session.Avatar
		}
	}

	return resp, nil
}
