package dao

import (
	"database/sql"
	"time"
)

// User 基本信息（不含密码）。
type User struct {
	Id       int64
	Email    *string
	Mobile   *string
	Nickname *string
	Avatar   *string
	Status   int16
	// 实名：real_name_status 0 未认证 1 已通过 2 审核中 3 失败
	RealNameStatus   int16
	RealNameAt       sql.NullTime
	RealNameCertType sql.NullInt16
	// 注销：冷静期未结束前禁止登录；执行后 account_cancelled_at 表示逻辑销户完成
	CancellationCoolingUntil sql.NullTime
	AccountCancelledAt     sql.NullTime

	// 基础画像（来自 migrations/001_init.sql 中的 birthday/gender）
	Birthday sql.NullTime
	Gender   sql.NullInt16

	// 资料扩展（迁移 025）
	Constellation        sql.NullString
	Age                  sql.NullInt16
	Signature            sql.NullString
	Bio                  sql.NullString
	BirthdayVisibility   sql.NullInt16 // 0=仅自己 1=好友 2=公开
	GenderVisibility     sql.NullInt16
	ProfileComplete      int16 // 0/1，表 NOT NULL DEFAULT 0
	ProfileCompleteScore int16 // 0-100
	Hobbies              sql.NullString
	Location             sql.NullString
}

// UserWithPassword 登录校验用。
type UserWithPassword struct {
	Id                 int64
	Email              *string
	Mobile             *string
	Nickname           *string
	Avatar             *string
	Status             int16
	Password           *string
	Salt               *string
	AccountLockedUntil sql.NullTime
	LoginFailCount     int32
	CancellationCoolingUntil sql.NullTime
	AccountCancelledAt       sql.NullTime
}

// UserProfileFromWithPassword 提取公开展示字段（供重置密码等场景避免二次查询）。
func UserProfileFromWithPassword(u *UserWithPassword) *User {
	if u == nil {
		return nil
	}
	return &User{
		Id:       u.Id,
		Email:    u.Email,
		Mobile:   u.Mobile,
		Nickname: u.Nickname,
		Avatar:   u.Avatar,
		Status:   u.Status,
	}
}

// UserInsertParams 插入 users（仅 SQL 参数聚合）。
type UserInsertParams struct {
	Email, Mobile, Password, Salt, Nickname, Avatar *string
	Status                                          int
	RegisterIP, RegisterChannel                     *string
	PasswordChangedAt                               sql.NullTime
	InviteCode, DeviceID                            *string
	Birthday                                        sql.NullTime
	Gender                                          *int16
}

// RegisterMeta 注册时 logic 层组装的审计字段（不入库默认值由表 DEFAULT 承担）。
type RegisterMeta struct {
	ClientIP        string
	RegisterChannel string // 可覆盖为 web/app 等；空则使用账号通道 email/mobile
	DeviceID        string
	InviteCode      string
}

// ProfileUpdate 更新用户资料可选字段（nil 表示不修改该列）。
type ProfileUpdate struct {
	Nickname             *string
	Avatar               *string
	Birthday             *time.Time
	Gender               *int16
	Constellation        *string
	Age                  *int16
	Signature            *string
	Bio                  *string
	BirthdayVisibility   *int16
	GenderVisibility     *int16
	ProfileComplete      *int16
	ProfileCompleteScore *int16
	Hobbies              *string
	Location             *string
}

// Any 是否至少有一项要更新。
func (p *ProfileUpdate) Any() bool {
	if p == nil {
		return false
	}
	return p.Nickname != nil || p.Avatar != nil ||
		p.Birthday != nil || p.Gender != nil ||
		p.Constellation != nil || p.Age != nil || p.Signature != nil || p.Bio != nil ||
		p.BirthdayVisibility != nil || p.GenderVisibility != nil ||
		p.ProfileComplete != nil || p.ProfileCompleteScore != nil ||
		p.Hobbies != nil || p.Location != nil
}

