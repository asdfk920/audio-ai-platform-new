package logic

import (
	"context"
	"fmt"
	"regexp"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/pkg/passwd"
	"github.com/jacklau/audio-ai-platform/pkg/redisx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/repo/dao"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/jacklau/audio-ai-platform/services/user/internal/userdomain/profile/userinfo"
	"github.com/zeromicro/go-zero/core/logx"
)

var (
	regEmailRegexReset  = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	regMobileRegexReset = regexp.MustCompile(`^1[3-9]\d{9}$`)
)

type ResetPasswordLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewResetPasswordLogic 忘记旧密码：邮箱/手机号 + 验证码 重置密码
func NewResetPasswordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResetPasswordLogic {
	return &ResetPasswordLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ResetPasswordLogic) ResetPassword(req *types.ResetPasswordReq) (resp *types.UserInfo, err error) {
	target, by := l.normalizeTarget(req)
	if target == "" {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "请使用邮箱或手机号其中一种方式重置密码")
	}
	if by == "email" && !regEmailRegexReset.MatchString(req.Email) {
		return nil, errorx.NewDefaultError(errorx.CodeInvalidEmail)
	}
	if by == "mobile" && !regMobileRegexReset.MatchString(req.Mobile) {
		return nil, errorx.NewDefaultError(errorx.CodeInvalidMobile)
	}
	if req.VerifyCode == "" {
		return nil, errorx.NewDefaultError(errorx.CodeVerifyCodeInvalid)
	}
	if req.NewPassword == "" || len(req.NewPassword) < 6 {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "新密码不能少于6位")
	}
	if req.NewPasswordConfirm == "" {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "请再次输入新密码")
	}
	if req.NewPassword != req.NewPasswordConfirm {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "两次新密码不一致")
	}

	// 验证验证码
	// 为什么重置密码必须校验验证码：
	// - 忘记旧密码时无法再做“旧密码校验”，验证码就承担了身份验证的职责。
	// - 验证码 key 按 target（邮箱/手机号）区分，因此“发送验证码的方式”和“重置密码填写的方式”必须一致。
	codeKey := fmt.Sprintf("user:verify_code:%s", target)
	stored, err := redisx.Get(l.ctx, codeKey)
	if err != nil || stored == "" {
		return nil, errorx.NewDefaultError(errorx.CodeVerifyCodeInvalid)
	}
	if stored != req.VerifyCode {
		return nil, errorx.NewDefaultError(errorx.CodeVerifyCodeInvalid)
	}

	repo := l.svcCtx.UserRepo
	var u *dao.UserWithPassword
	if by == "email" {
		u, err = repo.FindByEmailForLogin(l.ctx, req.Email)
	} else {
		u, err = repo.FindByMobileForLogin(l.ctx, req.Mobile)
	}
	if err != nil {
		l.Logger.Errorf("find user: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, err.Error())
	}
	if u == nil {
		return nil, errorx.NewDefaultError(errorx.CodeUserNotFound)
	}

	// 新密码不能与旧密码一样（若旧密码存在）
	// 设计原因：避免用户“重置后仍是旧密码”，并减少弱口令重复使用风险。
	if u.Password != nil && u.Salt != nil {
		if passwd.VerifyPassword(*u.Salt, req.NewPassword, *u.Password) {
			return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "新密码不能与旧密码一致")
		}
	}

	newSalt, newHash, err := passwd.HashPasswordWithNewSalt(req.NewPassword)
	if err != nil {
		l.Logger.Errorf("HashPasswordWithNewSalt: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeSystemError, "密码加密失败")
	}

	if err := repo.UpdatePassword(l.ctx, u.Id, newHash, newSalt); err != nil {
		l.Logger.Errorf("UpdatePassword: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, err.Error())
	}

	_ = redisx.Del(l.ctx, codeKey)

	uu, err := repo.FindByID(l.ctx, u.Id)
	if err != nil {
		l.Logger.Errorf("FindByID: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, err.Error())
	}
	if uu == nil {
		return nil, errorx.NewDefaultError(errorx.CodeUserNotFound)
	}

	info := userinfo.FromDAO(uu)
	return &info, nil
}

func (l *ResetPasswordLogic) normalizeTarget(req *types.ResetPasswordReq) (target, by string) {
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
