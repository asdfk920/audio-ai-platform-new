package logic

import (
	"context"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/common/validate"
	"github.com/jacklau/audio-ai-platform/pkg/passwd"
	"github.com/jacklau/audio-ai-platform/services/user/internal/pkg/util/ctxuser"
	"github.com/jacklau/audio-ai-platform/services/user/internal/repo/dao"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type ChangePasswordLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewChangePasswordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChangePasswordLogic {
	return &ChangePasswordLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ChangePasswordLogic) ChangePassword(req *types.ChangePasswordReq) (resp *types.ChangePasswordResp, err error) {
	userId := ctxuser.ParseUserID(l.ctx)
	if userId <= 0 {
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "登录已过期或无效，请重新登录")
	}

	oldPwd := req.OldPassword
	newPwd := req.NewPassword
	newPwdConfirm := req.NewPasswordConfirm

	if oldPwd == "" {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "请输入旧密码")
	}
	if newPwd == "" {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "请输入新密码")
	}
	if newPwdConfirm == "" {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "请再次输入新密码")
	}

	if err := validate.CheckPasswordMin(newPwd, 6); err != nil {
		return nil, err
	}

	if newPwd != newPwdConfirm {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "两次输入的新密码不一致")
	}

	if oldPwd == newPwd {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "新密码不能与旧密码相同")
	}

	u, err := l.svcCtx.UserRepo.FindByID(l.ctx, userId)
	if err != nil {
		l.Logger.Errorf("FindByID: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "系统繁忙，请稍后重试")
	}
	if u == nil {
		return nil, errorx.NewDefaultError(errorx.CodeUserNotFound)
	}

	var userWithPass *dao.UserWithPassword
	userWithPass, err = l.svcCtx.UserRepo.FindByIDForLogin(l.ctx, userId)
	if err != nil {
		l.Logger.Errorf("FindByIDForLogin: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "系统繁忙，请稍后重试")
	}
	if userWithPass == nil || userWithPass.Password == nil || userWithPass.Salt == nil {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "用户密码信息异常，请联系管理员")
	}

	if !passwd.VerifyPassword(*userWithPass.Salt, oldPwd, *userWithPass.Password) {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "旧密码错误，请重新输入")
	}

	newSalt, newHash, err := passwd.HashPasswordWithNewSalt(newPwd)
	if err != nil {
		l.Logger.Errorf("HashPasswordWithNewSalt: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeSystemError, "密码加密失败，请稍后重试")
	}

	if err := l.svcCtx.UserRepo.UpdatePassword(l.ctx, userId, newHash, newSalt); err != nil {
		l.Logger.Errorf("UpdatePassword: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "密码修改失败，请稍后重试")
	}

	return &types.ChangePasswordResp{
		Message: "密码修改成功",
	}, nil
}
