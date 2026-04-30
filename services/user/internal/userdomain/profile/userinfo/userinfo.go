// Package userinfo 将 users 主表行转为 API 类型 UserInfo（无密码字段）。
package userinfo

import (
	"github.com/jacklau/audio-ai-platform/services/user/internal/repo/dao"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
)

// FromDAO 将主表用户行转为对外 UserInfo（供更新资料、换绑、改密、重置密码等接口响应复用）。
func FromDAO(u *dao.User) types.UserInfo {
	if u == nil {
		return types.UserInfo{}
	}
	info := types.UserInfo{
		UserId:               u.Id,
		Status:               int32(u.Status),
		RealNameStatus:       int32(u.RealNameStatus),
		ProfileComplete:      int32(u.ProfileComplete),
		ProfileCompleteScore: int32(u.ProfileCompleteScore),
	}
	if u.Email != nil {
		info.Email = *u.Email
	}
	if u.Mobile != nil {
		info.Mobile = *u.Mobile
	}
	if u.Nickname != nil {
		info.Nickname = *u.Nickname
	}
	if u.Avatar != nil {
		info.Avatar = *u.Avatar
	}
	if u.RealNameCertType.Valid {
		info.RealNameCertType = int32(u.RealNameCertType.Int16)
	}
	if u.RealNameAt.Valid {
		info.RealNameAt = u.RealNameAt.Time.Unix()
	}
	if u.Constellation.Valid {
		info.Constellation = u.Constellation.String
	}
	if u.Age.Valid {
		info.Age = int32(u.Age.Int16)
	}
	if u.Signature.Valid {
		info.Signature = u.Signature.String
	}
	if u.Bio.Valid {
		info.Bio = u.Bio.String
	}
	if u.BirthdayVisibility.Valid {
		info.BirthdayVisibility = int32(u.BirthdayVisibility.Int16)
	}
	if u.GenderVisibility.Valid {
		info.GenderVisibility = int32(u.GenderVisibility.Int16)
	}
	if u.Hobbies.Valid {
		info.Hobbies = u.Hobbies.String
	}
	if u.Location.Valid {
		info.Location = u.Location.String
	}
	return info
}
