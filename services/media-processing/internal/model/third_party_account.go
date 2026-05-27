package model

import (
	"database/sql"
	"time"
)

// ThirdPartyAccount 第三方账号数据模型
// 对应数据库中的 third_party_account 表
type ThirdPartyAccount struct {
	ID            int64          `db:"id"`
	UserID        int64          `db:"user_id"`         // 系统用户 ID
	Provider      string         `db:"provider"`        // 第三方平台：netease/wechat/google
	OpenID        string         `db:"open_id"`         // 第三方平台的用户唯一标识
	UnionID       string         `db:"union_id"`        // 统一用户标识（可选）
	Nickname      string         `db:"nickname"`        // 昵称
	Avatar        string         `db:"avatar"`          // 头像 URL
	Gender        int16          `db:"gender"`          // 性别：0-未知，1-男，2-女
	CountryCode   string         `db:"country_code"`    // 国家/地区代码
	Province      string         `db:"province"`        // 省份
	City          string         `db:"city"`            // 城市
	AccessToken   string         `db:"access_token"`    // 访问令牌
	RefreshToken  string         `db:"refresh_token"`   // 刷新令牌
	TokenExpiresAt *time.Time    `db:"token_expires_at"` // 令牌过期时间
	RawData       sql.NullString `db:"raw_data"`        // 原始 JSON 数据
	Status        int16          `db:"status"`          // 状态：1-正常，0-禁用，-1-已解绑
	LastLoginAt   time.Time      `db:"last_login_at"`   // 最后登录时间
	CreatedAt     time.Time      `db:"created_at"`
	UpdatedAt     time.Time      `db:"updated_at"`
	DeletedAt     *time.Time     `db:"deleted_at"`
}

// ThirdPartyAccountStatus 第三方账号状态常量
const (
	ThirdPartyAccountStatusNormal   = 1  // 正常（已绑定）
	ThirdPartyAccountStatusDisabled = 0  // 禁用
	ThirdPartyAccountStatusUnbound  = -1 // 已解绑
)

// ThirdPartyProvider 第三方平台常量
const (
	ProviderNetease = "netease" // 网易云音乐
	ProviderWechat  = "wechat"  // 微信
	ProviderGoogle  = "google"  // Google
)
