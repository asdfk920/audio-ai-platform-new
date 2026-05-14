package logic

import (
	"database/sql"

	"github.com/jacklau/audio-ai-platform/services/user/internal/devicesharesvc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
)

func statusToInt(s string) int16 {
	switch s {
	case "pending":
		return 0
	case "active":
		return 1
	case "rejected":
		return 2
	case "revoked":
		return 3
	case "expired":
		return 4
	case "quit":
		return 5
	default:
		return 0
	}
}

func toDeviceShareItem(v *devicesharesvc.ShareView) *types.DeviceShareItem {
	if v == nil {
		return nil
	}
	var createdAt, endAt string
	if !v.CreatedAt.IsZero() {
		createdAt = v.CreatedAt.Format("2006-01-02 15:04:05")
	}
	if v.EndAt != nil {
		endAt = v.EndAt.Format("2006-01-02 15:04:05")
	}
	return &types.DeviceShareItem{
		ShareId:   v.ID,
		Sn:        v.DeviceSN,
		ToUserId:  v.SharedUserID,
		ToAccount: v.TargetAccount,
		Status:    statusToInt(v.Status),
		CreatedAt: createdAt,
		EndAt:     endAt,
	}
}

func toDeviceShareListResp(list []devicesharesvc.ShareView) *types.DeviceShareListResp {
	items := make([]types.DeviceShareItem, 0, len(list))
	for i := range list {
		item := toDeviceShareItem(&list[i])
		if item != nil {
			items = append(items, *item)
		}
	}
	return &types.DeviceShareListResp{List: items}
}

func formatNullTime(t sql.NullTime) string {
	if t.Valid {
		return t.Time.Format("2006-01-02 15:04:05")
	}
	return ""
}

func firstNonEmpty(strs ...string) string {
	for _, s := range strs {
		if s != "" {
			return s
		}
	}
	return ""
}
