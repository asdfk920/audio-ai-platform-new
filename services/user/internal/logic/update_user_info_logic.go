package logic

import (
	"context"
	"net/http"
	"time"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/pkg/upload"
	"github.com/jacklau/audio-ai-platform/services/user/internal/pkg/util/ctxuser"
	"github.com/jacklau/audio-ai-platform/services/user/internal/repo/dao"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	userinfo "github.com/jacklau/audio-ai-platform/services/user/internal/userdomain/profile/userinfo"
	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateUserInfoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateUserInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateUserInfoLogic {
	return &UpdateUserInfoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// UpdateUserInfoWithFile 修改用户信息（支持头像上传）
func (l *UpdateUserInfoLogic) UpdateUserInfoWithFile(req *types.UpdateUserInfoReq, r *http.Request) (resp *types.UserInfoResp, err error) {
	userId := ctxuser.ParseUserID(l.ctx)
	if userId <= 0 {
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "登录已过期或无效，请重新登录")
	}

	user, findErr := l.svcCtx.UserRepo.FindByID(l.ctx, userId)
	if findErr != nil {
		l.Logger.Errorf("find user: %v", findErr)
		return nil, errorx.NewDefaultError(errorx.CodeSystemError)
	}
	if user == nil {
		return nil, errorx.NewCodeError(errorx.CodeUserNotFound, "用户不存在")
	}

	update := &dao.ProfileUpdate{}

	// 处理文本字段
	if req.Username != "" {
		username := req.Username
		update.Username = &username
	}

	if req.Nickname != "" {
		nickname := req.Nickname
		update.Nickname = &nickname
	}

	if req.Signature != "" {
		signature := req.Signature
		update.Signature = &signature
	}

	if req.Bio != "" {
		bio := req.Bio
		update.Bio = &bio
	}

	if req.Constellation != "" {
		constellation := req.Constellation
		update.Constellation = &constellation
	}

	if req.Hobbies != "" {
		hobbies := req.Hobbies
		update.Hobbies = &hobbies
	}

	if req.Location != "" {
		location := req.Location
		update.Location = &location
	}

	if req.Language != "" {
		language := req.Language
		update.Language = &language
	}

	if req.Timezone != "" {
		timezone := req.Timezone
		update.Timezone = &timezone
	}

	if req.RealName != "" {
		realName := req.RealName
		update.RealName = &realName
	}

	// 处理整数字段
	if req.Gender != 0 {
		gender := int16(req.Gender)
		update.Gender = &gender
	}

	if req.Age != 0 {
		age := int16(req.Age)
		update.Age = &age
	}

	if req.BirthdayVisibility != 0 {
		birthdayVis := int16(req.BirthdayVisibility)
		update.BirthdayVisibility = &birthdayVis
	}

	if req.GenderVisibility != 0 {
		genderVis := int16(req.GenderVisibility)
		update.GenderVisibility = &genderVis
	}

	if req.ProfileComplete != 0 {
		pc := int16(req.ProfileComplete)
		update.ProfileComplete = &pc
	}

	if req.ProfileCompleteScore != 0 {
		pcs := int16(req.ProfileCompleteScore)
		update.ProfileCompleteScore = &pcs
	}

	// 处理生日字段
	if req.Birthday != "" {
		birthday, parseErr := time.Parse("2006-01-02", req.Birthday)
		if parseErr != nil {
			return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "生日格式错误，应为 YYYY-MM-DD")
		}
		update.Birthday = &birthday
	}

	// 处理头像上传
	_, fileHeader, fileErr := r.FormFile("avatar")
	if fileErr == nil && fileHeader != nil {
		cfg := &upload.Config{
			MaxFileSize:       l.svcCtx.Config.Upload.MaxFileSize * 1024 * 1024,
			SavePath:          l.svcCtx.Config.Upload.SavePath,
			AllowedExtensions: l.svcCtx.Config.Upload.AllowedExtensions,
		}
		uploadResult, uploadErr := upload.SaveAvatar(fileHeader, cfg)
		if uploadErr != nil {
			l.Logger.Errorf("upload avatar: %v", uploadErr)
			return nil, uploadErr
		}
		if uploadResult != nil {
			update.Avatar = &uploadResult.URL
		}
	} else if req.Avatar != "" {
		avatar := req.Avatar
		update.Avatar = &avatar
	}

	if !update.Any() {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "无更新内容")
	}

	if updateErr := l.svcCtx.UserRepo.UpdateProfile(l.ctx, userId, update); updateErr != nil {
		l.Logger.Errorf("update profile: %v", updateErr)
		return nil, updateErr
	}

	updatedUser, getErr := l.svcCtx.UserRepo.FindByID(l.ctx, userId)
	if getErr != nil {
		l.Logger.Errorf("find updated user: %v", getErr)
		return nil, errorx.NewDefaultError(errorx.CodeSystemError)
	}

	userInfo := userinfo.FromDAO(updatedUser)

	return &types.UserInfoResp{
		UserInfo: userInfo,
	}, nil
}
