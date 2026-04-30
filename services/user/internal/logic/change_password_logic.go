package logic

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/pkg/passwd"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/jacklau/audio-ai-platform/services/user/internal/userdomain/profile/userinfo"
	"github.com/zeromicro/go-zero/core/logx"
)

type ChangePasswordLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewChangePasswordLogic 修改密码（已登录，验证旧密码）
func NewChangePasswordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChangePasswordLogic {
	return &ChangePasswordLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ChangePasswordLogic) ChangePassword(req *types.ChangePasswordReq) (resp *types.UserInfo, err error) {
	userId := l.getUserIdFromCtx()
	if userId <= 0 {
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "登录已过期或无效，请重新登录")
	}

	// 为什么“修改密码”必须要求登录态：
	// - 该链路依赖旧密码校验，本质是“已认证用户”的敏感操作。
	// - 未登录场景走“验证码重置密码”，避免把两条安全链路混在一起。
	if req == nil {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "参数错误")
	}
	if req.OldPassword == "" {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "请输入旧密码")
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

	u, err := l.svcCtx.UserRepo.FindByIDForLogin(l.ctx, userId)
	if err != nil {
		l.Logger.Errorf("FindByIDForLogin: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, err.Error())
	}
	if u == nil {
		return nil, errorx.NewDefaultError(errorx.CodeUserNotFound)
	}
	if u.Password == nil || u.Salt == nil {
		return nil, errorx.NewCodeError(errorx.CodePasswordError, "该账号未设置密码，请使用第三方登录")
	}

	// 校验旧密码
	if !passwd.VerifyPassword(*u.Salt, req.OldPassword, *u.Password) {
		return nil, errorx.NewCodeError(errorx.CodePasswordError, "旧密码不正确")
	}

	// 新密码不能与旧密码一样
	// 设计原因：避免用户误操作“改了但没改”，也减少弱口令反复使用带来的风险。
	if passwd.VerifyPassword(*u.Salt, req.NewPassword, *u.Password) {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "新密码不能与旧密码一致")
	}

	newSalt, newHash, err := passwd.HashPasswordWithNewSalt(req.NewPassword)
	if err != nil {
		l.Logger.Errorf("HashPasswordWithNewSalt: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeSystemError, "密码加密失败")
	}

	if err := l.svcCtx.UserRepo.UpdatePassword(l.ctx, userId, newHash, newSalt); err != nil {
		l.Logger.Errorf("UpdatePassword: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, err.Error())
	}

	uu, err := l.svcCtx.UserRepo.FindByID(l.ctx, userId)
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

func (l *ChangePasswordLogic) getUserIdFromCtx() int64 {
	v := l.ctx.Value("userId")
	if v == nil {
		return 0
	}
	switch id := v.(type) {
	case json.Number:
		n, err := id.Int64()
		if err != nil {
			return 0
		}
		return n
	case float64:
		return int64(id)
	case int64:
		return id
	case int:
		return int64(id)
	case string:
		n, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			return 0
		}
		return n
	default:
		return 0
	}
}
