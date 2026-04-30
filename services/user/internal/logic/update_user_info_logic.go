package logic

import (
	"context"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/pkg/util/ctxuser"
	"github.com/jacklau/audio-ai-platform/services/user/internal/repo/dao"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/jacklau/audio-ai-platform/services/user/internal/userdomain/profile/userinfo"
	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateUserInfoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewUpdateUserInfoLogic 更新当前登录用户资料（身份从 JWT 解析，body 只传要修改的字段）。
func NewUpdateUserInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateUserInfoLogic {
	return &UpdateUserInfoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateUserInfoLogic) UpdateUserInfo(req *types.UpdateUserInfoReq) (resp *types.UserInfo, err error) {
	uid := ctxuser.ParseUserID(l.ctx)
	if uid <= 0 {
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "登录已过期或无效，请重新登录")
	}
	if req == nil {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "请求体不能为空")
	}
	p := profileUpdateFromReq(req)
	if p == nil || !p.Any() {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "请至少修改一项资料")
	}
	u, _, _, err := l.svcCtx.UserRepo.UpdateUserProfileTransactional(l.ctx, uid, p)
	if err != nil {
		l.Logger.Errorf("UpdateUserProfileTransactional: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, err.Error())
	}
	if u == nil {
		return nil, errorx.NewDefaultError(errorx.CodeUserNotFound)
	}
	info := userinfo.FromDAO(u)
	return &info, nil
}

func profileUpdateFromReq(req *types.UpdateUserInfoReq) *dao.ProfileUpdate {
	p := &dao.ProfileUpdate{}
	if s := strings.TrimSpace(req.Nickname); s != "" {
		p.Nickname = &s
	}
	if s := strings.TrimSpace(req.Avatar); s != "" {
		p.Avatar = &s
	}
	if s := strings.TrimSpace(req.Birthday); s != "" {
		t, err := time.ParseInLocation("2006-01-02", s, time.Local)
		if err == nil {
			p.Birthday = &t
		}
	}
	if req.Gender != nil {
		p.Gender = req.Gender
	}
	if s := strings.TrimSpace(req.Constellation); s != "" {
		p.Constellation = &s
	}
	if req.Age != nil {
		p.Age = req.Age
	}
	if s := strings.TrimSpace(req.Signature); s != "" {
		p.Signature = &s
	}
	if s := strings.TrimSpace(req.Bio); s != "" {
		p.Bio = &s
	}
	if req.BirthdayVisibility != nil {
		p.BirthdayVisibility = req.BirthdayVisibility
	}
	if req.GenderVisibility != nil {
		p.GenderVisibility = req.GenderVisibility
	}
	if req.ProfileComplete != nil {
		p.ProfileComplete = req.ProfileComplete
	}
	if req.ProfileCompleteScore != nil {
		p.ProfileCompleteScore = req.ProfileCompleteScore
	}
	if s := strings.TrimSpace(req.Hobbies); s != "" {
		p.Hobbies = &s
	}
	if s := strings.TrimSpace(req.Location); s != "" {
		p.Location = &s
	}
	return p
}
