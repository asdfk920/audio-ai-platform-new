package logic

import (
	"encoding/json"
	"time"

	"github.com/jacklau/audio-ai-platform/services/user/internal/devicesharesvc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
)

func unixPtr(t int64) *time.Time {
	if t == 0 {
		return nil
	}
	u := time.Unix(t, 0)
	return &u
}

func toDeviceShareItem(v *devicesharesvc.ShareView) *types.DeviceShareItem {
	if v == nil {
		return nil
	}
	var permStr string
	if v.Permission != nil {
		b, _ := json.Marshal(v.Permission)
		permStr = string(b)
	}
	var startAt, endAt, confirmedAt int64
	if v.StartAt != nil {
		startAt = v.StartAt.Unix()
	}
	if v.EndAt != nil {
		endAt = v.EndAt.Unix()
	}
	if v.ConfirmedAt != nil {
		confirmedAt = v.ConfirmedAt.Unix()
	}
	var createdAt int64
	if !v.CreatedAt.IsZero() {
		createdAt = v.CreatedAt.Unix()
	}
	return &types.DeviceShareItem{
		ShareId:         v.ID,
		DeviceId:        v.DeviceID,
		DeviceSn:        v.DeviceSN,
		DeviceName:      v.DeviceName,
		OwnerUserId:     v.OwnerUserID,
		OwnerNickname:   v.OwnerNickname,
		SharedUserId:    v.SharedUserID,
		SharedNickname:  v.SharedNickname,
		TargetAccount:   v.TargetAccount,
		InviteCode:      v.InviteCode,
		ShareType:       v.ShareType,
		PermissionLevel: v.PermissionLevel,
		Permission:      permStr,
		Status:          v.Status,
		StartAt:         startAt,
		EndAt:           endAt,
		CreatedAt:       createdAt,
		ConfirmedAt:     confirmedAt,
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
