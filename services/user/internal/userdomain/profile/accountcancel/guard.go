package accountcancel

import (
	"time"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/repo/dao"
)

// ErrIfAccountClosed 仅已执行逻辑注销的账号（冷静期中仍可使用注销相关接口）。
func ErrIfAccountClosed(u *dao.User) error {
	if u != nil && u.AccountCancelledAt.Valid {
		return errorx.NewDefaultError(errorx.CodeAccountCancelled)
	}
	return nil
}

// ErrIfClosedOrCooling 一般业务接口：冷静期与已注销均不可使用。
func ErrIfClosedOrCooling(u *dao.User) error {
	if u == nil {
		return nil
	}
	if u.AccountCancelledAt.Valid {
		return errorx.NewDefaultError(errorx.CodeAccountCancelled)
	}
	if u.CancellationCoolingUntil.Valid && u.CancellationCoolingUntil.Time.After(time.Now()) {
		return errorx.NewDefaultError(errorx.CodeCancellationCooling)
	}
	return nil
}

// ErrIfClosedOrCoolingLogin 密码登录前校验。
func ErrIfClosedOrCoolingLogin(u *dao.UserWithPassword) error {
	if u == nil {
		return nil
	}
	if u.AccountCancelledAt.Valid {
		return errorx.NewDefaultError(errorx.CodeAccountCancelled)
	}
	if u.CancellationCoolingUntil.Valid && u.CancellationCoolingUntil.Time.After(time.Now()) {
		return errorx.NewDefaultError(errorx.CodeCancellationCooling)
	}
	return nil
}
