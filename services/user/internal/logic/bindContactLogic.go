package logic

import (
	"context"
	"fmt"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/pkg/redisx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/pkg/util/ctxuser"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/jacklau/audio-ai-platform/services/user/internal/userdomain/profile/userinfo"
	"github.com/zeromicro/go-zero/core/logx"
)

type BindContactLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewBindContactLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BindContactLogic {
	return &BindContactLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *BindContactLogic) BindContact(req *types.BindContactReq) (resp *types.UserInfo, err error) {
	userId := ctxuser.ParseUserID(l.ctx)
	if userId <= 0 {
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "登录已过期或无效，请重新登录")
	}

	target, by := l.normalizeTarget(req)
	if target == "" || req.VerifyCode == "" {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "参数错误：请填写邮箱或手机号，并输入验证码")
	}

	if by == "email" && !regEmailRegex.MatchString(target) {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "邮箱格式不正确")
	}
	if by == "mobile" && !regMobileRegex.MatchString(target) {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "手机号格式不正确")
	}

	codeKey := fmt.Sprintf("user:verify_code:%s", target)
	storedCode, err := redisx.Get(l.ctx, codeKey)
	if err != nil {
		l.Logger.Errorf("redis get %s: %v", codeKey, err)
		exists, existsErr := redisx.Exists(l.ctx, codeKey)
		if existsErr != nil {
			return nil, errorx.NewCodeError(errorx.CodeRedisError, "系统繁忙，请稍后重试")
		}
		if exists == 0 {
			return nil, errorx.NewCodeError(errorx.CodeVerifyCodeInvalid, "验证码已过期，请重新获取验证码")
		}
		return nil, errorx.NewCodeError(errorx.CodeVerifyCodeInvalid, "验证码无效，请重新发送验证码")
	}
	if storedCode == "" {
		exists, _ := redisx.Exists(l.ctx, codeKey)
		if exists == 0 {
			return nil, errorx.NewCodeError(errorx.CodeVerifyCodeInvalid, "验证码已过期，请重新获取验证码")
		}
		return nil, errorx.NewCodeError(errorx.CodeVerifyCodeInvalid, "验证码无效，请重新发送验证码")
	}
	if storedCode != req.VerifyCode {
		return nil, errorx.NewCodeError(errorx.CodeVerifyCodeInvalid, "验证码错误，请检查后重新输入")
	}

	u, err := l.svcCtx.UserRepo.FindByID(l.ctx, userId)
	if err != nil {
		l.Logger.Errorf("FindByID: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "系统繁忙，请稍后重试")
	}
	if u == nil {
		return nil, errorx.NewDefaultError(errorx.CodeUserNotFound)
	}

	if by == "email" {
		if u.Email != nil && *u.Email != "" && *u.Email != target {
			return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "您已绑定邮箱，如需更换请使用换绑功能")
		}
		exist, err := l.svcCtx.UserRepo.FindByEmail(l.ctx, target)
		if err != nil {
			l.Logger.Errorf("FindByEmail: %v", err)
			return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "系统繁忙，请稍后重试")
		}
		if exist != nil && exist.Id != userId {
			return nil, errorx.NewCodeError(errorx.CodeUserExists, "该邮箱已被其他用户绑定")
		}
	} else {
		if u.Mobile != nil && *u.Mobile != "" && *u.Mobile != target {
			return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "您已绑定手机号，如需更换请使用换绑功能")
		}
		exist, err := l.svcCtx.UserRepo.FindByMobile(l.ctx, target)
		if err != nil {
			l.Logger.Errorf("FindByMobile: %v", err)
			return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "系统繁忙，请稍后重试")
		}
		if exist != nil && exist.Id != userId {
			return nil, errorx.NewCodeError(errorx.CodeUserExists, "该手机号已被其他用户绑定")
		}
	}

	var affected int64
	if by == "email" {
		affected, err = l.svcCtx.UserRepo.BindEmail(l.ctx, userId, target)
	} else {
		affected, err = l.svcCtx.UserRepo.BindMobile(l.ctx, userId, target)
	}
	if err != nil {
		l.Logger.Errorf("bind contact: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "绑定失败，请稍后重试")
	}
	if affected == 0 {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "绑定失败：账号已被绑定或数据异常")
	}

	_ = redisx.Del(l.ctx, codeKey)

	uu, err := l.svcCtx.UserRepo.FindByID(l.ctx, userId)
	if err != nil {
		l.Logger.Errorf("FindByID after bind: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "查询用户信息失败")
	}
	if uu == nil {
		return nil, errorx.NewDefaultError(errorx.CodeUserNotFound)
	}

	info := userinfo.FromDAO(uu)
	return &info, nil
}

func (l *BindContactLogic) normalizeTarget(req *types.BindContactReq) (target, by string) {
	hasEmail := req.Email != ""
	hasMobile := req.Mobile != ""
	if hasEmail && !hasMobile {
		return req.Email, "email"
	}
	if hasMobile && !hasEmail {
		return req.Mobile, "mobile"
	}
	return "", ""
}
