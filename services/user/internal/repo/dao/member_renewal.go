package dao

import (
	"database/sql"
	"time"
)

// BizScene 订单业务场景（与 order_master.biz_scene / CHECK 约束一致）。
const (
	BizSceneNew          = "new"
	BizSceneRenewActive  = "renew_active"
	BizSceneRenewExpired = "renew_expired"
)

// UserMemberRenewalState 计算续费时 user_member 快照（可与 DB 行对应）。
type UserMemberRenewalState struct {
	HasRow      bool
	Status      int16
	IsPermanent bool
	ExpireAt    time.Time
	ExpireValid bool
}

// BuildRenewalState 由 SELECT expire_at, is_permanent, status 一行构造。
func BuildRenewalState(exp sql.NullTime, perm sql.NullInt16, st sql.NullInt16) UserMemberRenewalState {
	s := UserMemberRenewalState{HasRow: true, Status: 0}
	if st.Valid {
		s.Status = st.Int16
	}
	if perm.Valid && perm.Int16 == 1 {
		s.IsPermanent = true
	}
	if exp.Valid {
		s.ExpireValid = true
		s.ExpireAt = exp.Time
	}
	return s
}

// ComputeMemberRenewal 与 FulfillPaidOrderTx / PreviewNewExpireAt 共用：计算新到期日与场景。
// 永久会员续费：不在日历上延长，预览为当前档案到期日（若无则当前时间）。
func ComputeMemberRenewal(now time.Time, s UserMemberRenewalState, durationDays int) (newExpire time.Time, bizScene string) {
	if !s.HasRow {
		return now.AddDate(0, 0, durationDays), BizSceneNew
	}
	if s.Status != 1 {
		return now.AddDate(0, 0, durationDays), BizSceneRenewExpired
	}
	if s.IsPermanent {
		if s.ExpireValid {
			return s.ExpireAt, BizSceneRenewActive
		}
		return now, BizSceneRenewActive
	}
	if s.ExpireValid && s.ExpireAt.After(now) {
		return s.ExpireAt.AddDate(0, 0, durationDays), BizSceneRenewActive
	}
	return now.AddDate(0, 0, durationDays), BizSceneRenewExpired
}

// ShouldSkipMembershipExtend 永久有效会员：支付仅记账，不累加 expire_at。
func ShouldSkipMembershipExtend(s UserMemberRenewalState) bool {
	return s.HasRow && s.Status == 1 && s.IsPermanent
}
