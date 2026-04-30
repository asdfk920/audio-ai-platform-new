package logic

import (
	"testing"
	"time"

	"github.com/jacklau/audio-ai-platform/services/user/internal/repo/dao"
)

func TestSubscriptionPhaseCancelPending(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	exp := now.AddDate(0, 0, 30)
	m := &dao.UserMemberRow{
		Status:        1,
		IsPermanent:   0,
		ExpireAt:      exp,
		CancelPending: 1,
	}
	if got := subscriptionPhase(m, now); got != "cancel_pending" {
		t.Fatalf("got %q", got)
	}
}
