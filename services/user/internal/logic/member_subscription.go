package logic

import (
	"time"

	"github.com/jacklau/audio-ai-platform/services/user/internal/repo/dao"
)

// subscriptionPhase 返回 none | active | cancel_pending | expired（无会员档为 none）。
func subscriptionPhase(m *dao.UserMemberRow, now time.Time) string {
	if m == nil {
		return "none"
	}
	if m.IsPermanent == 1 {
		return "active"
	}
	if m.Status != 1 {
		return "expired"
	}
	if !m.ExpireAt.After(now) {
		return "expired"
	}
	if m.CancelPending == 1 {
		return "cancel_pending"
	}
	return "active"
}
