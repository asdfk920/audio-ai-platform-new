package logic

import (
	"encoding/json"
	"time"

	"github.com/jacklau/audio-ai-platform/services/user/internal/devicesharesvc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/familysvc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
)

func toFamilyInfoResp(v *familysvc.FamilyView) *types.FamilyInfoResp {
	if v == nil {
		return nil
	}
	return &types.FamilyInfoResp{
		FamilyId:    v.ID,
		Name:        v.Name,
		OwnerUserId: v.OwnerUserID,
		CurrentRole: v.CurrentRole,
		MemberCount: v.MemberCount,
		CreatedAt:   v.CreatedAt.Unix(),
		UpdatedAt:   v.UpdatedAt.Unix(),
	}
}

func toFamilyInviteResp(v *familysvc.FamilyInviteView) *types.FamilyMemberInviteResp {
	if v == nil {
		return nil
	}
	resp := &types.FamilyMemberInviteResp{
		FamilyId:      v.FamilyID,
		InviteCode:    v.InviteCode,
		TargetUserId:  v.TargetUserID,
		TargetAccount: v.TargetAccount,
		Role:          v.Role,
		Status:        v.Status,
	}
	if v.ExpiresAt != nil {
		resp.ExpireAt = v.ExpiresAt.Unix()
	}
	return resp
}

func toFamilyMemberItem(v familysvc.MemberView) types.FamilyMemberItem {
	return types.FamilyMemberItem{
		UserId:    v.UserID,
		Nickname:  v.Nickname,
		Email:     v.Email,
		Mobile:    v.Mobile,
		Role:      v.Role,
		Status:    v.Status,
		JoinedAt:  v.JoinedAt.Unix(),
		InvitedBy: v.InvitedBy,
	}
}

func toDeviceShareItem(v *devicesharesvc.ShareView) *types.DeviceShareItem {
	if v == nil {
		return nil
	}
	resp := &types.DeviceShareItem{
		ShareId:         v.ID,
		FamilyId:        v.FamilyID,
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
		Permission:      marshalJSON(v.Permission),
		Status:          v.Status,
		FamilyName:      v.FamilyName,
		CreatedAt:       v.CreatedAt.Unix(),
	}
	if v.StartAt != nil {
		resp.StartAt = v.StartAt.Unix()
	}
	if v.EndAt != nil {
		resp.EndAt = v.EndAt.Unix()
	}
	if v.ConfirmedAt != nil {
		resp.ConfirmedAt = v.ConfirmedAt.Unix()
	}
	return resp
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

func unixPtr(v int64) *time.Time {
	if v <= 0 {
		return nil
	}
	t := time.Unix(v, 0)
	return &t
}

func marshalJSON(v any) string {
	if v == nil {
		return ""
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}
