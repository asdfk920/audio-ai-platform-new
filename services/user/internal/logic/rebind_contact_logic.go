package logic

import (
	"context"
	"fmt"
	"regexp"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/pkg/redisx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/pkg/util/ctxuser"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

var (
	regEmailRegexRebind  = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	regMobileRegexRebind = regexp.MustCompile(`^1[3-9]\d{9}$`)
)

type RebindContactLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRebindContactLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RebindContactLogic {
	return &RebindContactLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RebindContactLogic) RebindContact(req *types.RebindContactReq) (resp *types.RebindContactResp, err error) {
	userId := ctxuser.ParseUserID(l.ctx)
	if userId <= 0 {
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "登录已过期或无效，请重新登录")
	}
	if req == nil {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "参数错误：请求体不能为空")
	}

	oldTarget, newTarget, by := l.normalizeTargets(req)
	if by == "" {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "请按同一种方式换绑（邮箱→邮箱 或 手机号→手机号）")
	}
	if oldTarget == "" || newTarget == "" {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "请填写旧的和新的联系方式")
	}
	if req.OldVerifyCode == "" {
		return nil, errorx.NewCodeError(errorx.CodeVerifyCodeInvalid, "请输入旧联系方式的验证码")
	}

	u, err := l.svcCtx.UserRepo.FindByID(l.ctx, userId)
	if err != nil {
		l.Logger.Errorf("FindByID: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "系统繁忙，请稍后重试")
	}
	if u == nil {
		return nil, errorx.NewDefaultError(errorx.CodeUserNotFound)
	}

	switch by {
	case "email":
		if u.Email == nil || *u.Email != oldTarget {
			return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "旧邮箱与当前绑定的邮箱不一致")
		}
		if !regEmailRegexRebind.MatchString(newTarget) {
			return nil, errorx.NewDefaultError(errorx.CodeInvalidEmail)
		}
	case "mobile":
		if u.Mobile == nil || *u.Mobile != oldTarget {
			return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "旧手机号与当前绑定的手机号不一致")
		}
		if !regMobileRegexRebind.MatchString(newTarget) {
			return nil, errorx.NewDefaultError(errorx.CodeInvalidMobile)
		}
	}

	if oldTarget == newTarget {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "新旧联系方式不能相同")
	}

	if err := l.verifyOldCode(oldTarget, req.OldVerifyCode); err != nil {
		return nil, err
	}

	conflictUser, err := l.checkConflict(newTarget, by, userId)
	if err != nil {
		return nil, err
	}
	if conflictUser != nil {
		msg := fmt.Sprintf("该%s已被其他用户绑定", byLabel(by))
		return nil, errorx.NewCodeError(errorx.CodeUserExists, msg)
	}

	var affected int64
	switch by {
	case "email":
		affected, err = l.svcCtx.UserRepo.RebindEmail(l.ctx, userId, oldTarget, newTarget)
	default:
		affected, err = l.svcCtx.UserRepo.RebindMobile(l.ctx, userId, oldTarget, newTarget)
	}
	if err != nil {
		l.Logger.Errorf("rebind %s: %v", by, err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "换绑失败，请稍后重试")
	}
	if affected == 0 {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "换绑失败：数据状态异常或已被修改")
	}

	_ = redisx.Del(l.ctx, l.verifyCodeKey(oldTarget))

	return &types.RebindContactResp{
		Message: "换绑成功",
	}, nil
}

func (l *RebindContactLogic) verifyOldCode(target, code string) error {
	codeKey := l.verifyCodeKey(target)
	storedCode, err := redisx.Get(l.ctx, codeKey)
	if err != nil {
		l.Logger.Errorf("redis get %s: %v", codeKey, err)
		exists, existsErr := redisx.Exists(l.ctx, codeKey)
		if existsErr != nil {
			return errorx.NewCodeError(errorx.CodeRedisError, "系统繁忙，请稍后重试")
		}
		if exists == 0 {
			return errorx.NewCodeError(errorx.CodeVerifyCodeInvalid, "验证码已过期，请重新获取验证码")
		}
		return errorx.NewCodeError(errorx.CodeVerifyCodeInvalid, "验证码无效，请重新发送验证码")
	}
	if storedCode == "" {
		exists, _ := redisx.Exists(l.ctx, codeKey)
		if exists == 0 {
			return errorx.NewCodeError(errorx.CodeVerifyCodeInvalid, "验证码已过期，请重新获取验证码")
		}
		return errorx.NewCodeError(errorx.CodeVerifyCodeInvalid, "验证码无效，请重新发送验证码")
	}
	if storedCode != code {
		return errorx.NewCodeError(errorx.CodeVerifyCodeInvalid, "验证码错误，请检查后重新输入")
	}
	return nil
}

func (l *RebindContactLogic) checkConflict(target string, by string, currentUserId int64) (*interface{}, error) {
	switch by {
	case "email":
		exist, err := l.svcCtx.UserRepo.FindByEmail(l.ctx, target)
		if err != nil {
			l.Logger.Errorf("FindByEmail: %v", err)
			return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "系统繁忙，请稍后重试")
		}
		if exist != nil && exist.Id != currentUserId {
			var result interface{} = exist
			return &result, nil
		}
	default:
		exist, err := l.svcCtx.UserRepo.FindByMobile(l.ctx, target)
		if err != nil {
			l.Logger.Errorf("FindByMobile: %v", err)
			return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "系统繁忙，请稍后重试")
		}
		if exist != nil && exist.Id != currentUserId {
			var result interface{} = exist
			return &result, nil
		}
	}
	return nil, nil
}

func (l *RebindContactLogic) normalizeTargets(req *types.RebindContactReq) (oldTarget, newTarget, by string) {
	hasOldEmail := req.OldEmail != ""
	hasOldMobile := req.OldMobile != ""
	hasNewEmail := req.NewEmail != ""
	hasNewMobile := req.NewMobile != ""

	if hasOldEmail && !hasOldMobile && hasNewEmail && !hasNewMobile {
		return req.OldEmail, req.NewEmail, "email"
	}
	if hasOldMobile && !hasOldEmail && hasNewMobile && !hasNewEmail {
		return req.OldMobile, req.NewMobile, "mobile"
	}
	return "", "", ""
}

func (l *RebindContactLogic) verifyCodeKey(target string) string {
	return fmt.Sprintf("user:verify_code:%s", target)
}

func byLabel(by string) string {
	if by == "email" {
		return "邮箱"
	}
	return "手机号"
}
