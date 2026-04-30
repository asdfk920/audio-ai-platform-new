package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/pkg/redisx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/jacklau/audio-ai-platform/services/user/internal/userdomain/profile/userinfo"
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

// NewRebindContactLogic 换绑手机号/邮箱（已登录：旧验证码 + 新验证码）
func NewRebindContactLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RebindContactLogic {
	return &RebindContactLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RebindContactLogic) RebindContact(req *types.RebindContactReq) (resp *types.UserInfo, err error) {
	userId := l.getUserIdFromCtx()
	if userId <= 0 {
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "登录已过期或无效，请重新登录")
	}
	if req == nil {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "参数错误")
	}

	oldTarget, newTarget, by := l.normalizeTargets(req)
	if by == "" {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "请按同一种方式换绑（邮箱->邮箱 或 手机号->手机号）")
	}
	if req.OldVerifyCode == "" || req.NewVerifyCode == "" {
		return nil, errorx.NewDefaultError(errorx.CodeVerifyCodeInvalid)
	}

	// 绑定前校验当前用户是否确实绑定了 oldTarget
	// 为什么要校验 oldTarget 属于当前用户：
	// - 换绑的安全前提是“证明你拥有旧账号”（旧验证码校验）并且“旧账号确实是你当前绑定的”。
	// - 防止用户拿到别人的旧验证码后越权换绑（oldTarget guard 是关键）。
	u, err := l.svcCtx.UserRepo.FindByID(l.ctx, userId)
	if err != nil {
		l.Logger.Errorf("FindByID: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, err.Error())
	}
	if u == nil {
		return nil, errorx.NewDefaultError(errorx.CodeUserNotFound)
	}
	if by == "email" {
		if u.Email == nil || *u.Email != oldTarget {
			return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "旧邮箱与当前绑定不一致")
		}
		if !regEmailRegexRebind.MatchString(newTarget) {
			return nil, errorx.NewDefaultError(errorx.CodeInvalidEmail)
		}
	} else {
		if u.Mobile == nil || *u.Mobile != oldTarget {
			return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "旧手机号与当前绑定不一致")
		}
		if !regMobileRegexRebind.MatchString(newTarget) {
			return nil, errorx.NewDefaultError(errorx.CodeInvalidMobile)
		}
	}
	if oldTarget == newTarget {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "新旧账号不能相同")
	}

	// 校验旧账号验证码
	// 说明：验证码 key 按目标账号区分，所以必须分别对 oldTarget/newTarget 做校验。
	if err := l.verifyCode(oldTarget, req.OldVerifyCode); err != nil {
		return nil, err
	}
	// 校验新账号验证码
	if err := l.verifyCode(newTarget, req.NewVerifyCode); err != nil {
		return nil, err
	}

	// 绑定前先查库：新账号是否已被其他用户绑定
	if by == "email" {
		exist, err := l.svcCtx.UserRepo.FindByEmail(l.ctx, newTarget)
		if err != nil {
			l.Logger.Errorf("FindByEmail: %v", err)
			return nil, errorx.NewCodeError(errorx.CodeDatabaseError, err.Error())
		}
		if exist != nil && exist.Id != userId {
			return nil, errorx.NewCodeError(errorx.CodeUserExists, "该账号已被其他用户绑定")
		}
	} else {
		exist, err := l.svcCtx.UserRepo.FindByMobile(l.ctx, newTarget)
		if err != nil {
			l.Logger.Errorf("FindByMobile: %v", err)
			return nil, errorx.NewCodeError(errorx.CodeDatabaseError, err.Error())
		}
		if exist != nil && exist.Id != userId {
			return nil, errorx.NewCodeError(errorx.CodeUserExists, "该账号已被其他用户绑定")
		}
	}

	// 执行换绑（带旧值 guard + 唯一性兜底）
	// 这里的 RebindEmail/RebindMobile 带 WHERE email/mobile = oldTarget：
	// - 能抵御并发/状态变化（例如用户在另一端已换绑）
	// - 也让“旧账号不匹配”时不会误更新到新账号
	var affected int64
	if by == "email" {
		affected, err = l.svcCtx.UserRepo.RebindEmail(l.ctx, userId, oldTarget, newTarget)
	} else {
		affected, err = l.svcCtx.UserRepo.RebindMobile(l.ctx, userId, oldTarget, newTarget)
	}
	if err != nil {
		l.Logger.Errorf("rebind: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, err.Error())
	}
	if affected == 0 {
		return nil, errorx.NewCodeError(errorx.CodeUserExists, "换绑失败：账号已被绑定或旧账号不匹配")
	}

	// 删除两个验证码
	_ = redisx.Del(l.ctx, l.verifyCodeKey(oldTarget))
	_ = redisx.Del(l.ctx, l.verifyCodeKey(newTarget))

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

func (l *RebindContactLogic) verifyCode(target, code string) error {
	stored, err := redisx.Get(l.ctx, l.verifyCodeKey(target))
	if err != nil || stored == "" {
		return errorx.NewDefaultError(errorx.CodeVerifyCodeInvalid)
	}
	if stored != code {
		return errorx.NewDefaultError(errorx.CodeVerifyCodeInvalid)
	}
	return nil
}

func (l *RebindContactLogic) verifyCodeKey(target string) string {
	return fmt.Sprintf("user:verify_code:%s", target)
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

func (l *RebindContactLogic) getUserIdFromCtx() int64 {
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
