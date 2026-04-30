package dao

import (
	"testing"
	"time"
)

func TestComputeMemberRenewal(t *testing.T) {
	day := func(d int) time.Time {
		return time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC).AddDate(0, 0, d)
	}
	now := day(0)

	tcases := []struct {
		name   string
		st     UserMemberRenewalState
		days   int
		want   time.Time
		scene  string
	}{
		{
			name:  "no_row",
			st:    UserMemberRenewalState{HasRow: false},
			days:  30,
			want:  now.AddDate(0, 0, 30),
			scene: BizSceneNew,
		},
		{
			name: "active_stack",
			st: UserMemberRenewalState{
				HasRow: true, Status: 1, IsPermanent: false,
				ExpireValid: true, ExpireAt: day(10),
			},
			days:  30,
			want:  day(10).AddDate(0, 0, 30),
			scene: BizSceneRenewActive,
		},
		{
			name: "expired_buy",
			st: UserMemberRenewalState{
				HasRow: true, Status: 1, IsPermanent: false,
				ExpireValid: true, ExpireAt: day(-5),
			},
			days:  30,
			want:  now.AddDate(0, 0, 30),
			scene: BizSceneRenewExpired,
		},
		{
			name: "permanent_no_extend",
			st: UserMemberRenewalState{
				HasRow: true, Status: 1, IsPermanent: true,
				ExpireValid: true, ExpireAt: day(100),
			},
			days:  30,
			want:  day(100),
			scene: BizSceneRenewActive,
		},
	}
	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			got, scene := ComputeMemberRenewal(now, tc.st, tc.days)
			if !got.Equal(tc.want) {
				t.Fatalf("expire got=%s want=%s", got.UTC().Format(time.RFC3339), tc.want.UTC().Format(time.RFC3339))
			}
			if scene != tc.scene {
				t.Fatalf("scene got=%s want=%s", scene, tc.scene)
			}
		})
	}
}
