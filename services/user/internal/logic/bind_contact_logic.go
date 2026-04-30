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
	regEmailRegexBind  = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	regMobileRegexBind = regexp.MustCompile(`^1[3-9]\d{9}$`)
)

type BindContactLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewBindContactLogic 绑定手机号/邮箱（已登录 + 验证码）
func NewBindContactLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BindContactLogic {
	return &BindContactLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *BindContactLogic) BindContact(req *types.BindContactReq) (resp *types.UserInfo, err error) {
	userId := l.getUserIdFromCtx()
	if userId <= 0 {
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "登录已过期或无效，请重新登录")
	}

	target, by := l.normalizeTarget(req)
	if target == "" {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "请填写邮箱或手机号其中一项进行绑定")
	}
	if req.VerifyCode == "" {
		return nil, errorx.NewDefaultError(errorx.CodeVerifyCodeInvalid)
	}
	if by == "email" && !regEmailRegexBind.MatchString(req.Email) {
		return nil, errorx.NewDefaultError(errorx.CodeInvalidEmail)
	}
	if by == "mobile" && !regMobileRegexBind.MatchString(req.Mobile) {
		return nil, errorx.NewDefaultError(errorx.CodeInvalidMobile)
	}

	// 校验验证码
	codeKey := fmt.Sprintf("user:verify_code:%s", target)
	stored, err := redisx.Get(l.ctx, codeKey)
	if err != nil || stored == "" {
		return nil, errorx.NewDefaultError(errorx.CodeVerifyCodeInvalid)
	}
	if stored != req.VerifyCode {
		return nil, errorx.NewDefaultError(errorx.CodeVerifyCodeInvalid)
	}

	// 绑定前先查库：是否已被其他用户绑定
	// 为什么“先查再绑定”：
	// - 让用户获得明确错误提示（该账号已被其他用户绑定），而不是依赖 UPDATE 影响行数的“模糊失败”。
	// - 仍保留后续 DB 条件作为并发兜底（两人同时绑定同一号码时，只有一个会成功）。
	if by == "email" {
		exist, err := l.svcCtx.UserRepo.FindByEmail(l.ctx, req.Email)
		if err != nil {
			l.Logger.Errorf("FindByEmail: %v", err)
			return nil, errorx.NewCodeError(errorx.CodeDatabaseError, err.Error())
		}
		if exist != nil && exist.Id != userId {
			return nil, errorx.NewCodeError(errorx.CodeUserExists, "该账号已被其他用户绑定")
		}
	} else {
		exist, err := l.svcCtx.UserRepo.FindByMobile(l.ctx, req.Mobile)
		if err != nil {
			l.Logger.Errorf("FindByMobile: %v", err)
			return nil, errorx.NewCodeError(errorx.CodeDatabaseError, err.Error())
		}
		if exist != nil && exist.Id != userId {
			return nil, errorx.NewCodeError(errorx.CodeUserExists, "该账号已被其他用户绑定")
		}
	}

	// 绑定（再次依赖 DB 唯一性条件兜底）
	// 注意：这里的 BindEmail/BindMobile 内部带了 NOT EXISTS 检查，避免并发下重复绑定。
	var affected int64
	if by == "email" {
		affected, err = l.svcCtx.UserRepo.BindEmail(l.ctx, userId, req.Email)
	} else {
		affected, err = l.svcCtx.UserRepo.BindMobile(l.ctx, userId, req.Mobile)
	}
	if err != nil {
		l.Logger.Errorf("bind contact: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, err.Error())
	}
	if affected == 0 {
		// 已绑定了不同值，或竞态下被其他用户先绑定
		return nil, errorx.NewCodeError(errorx.CodeUserExists, "该账号已被其他用户绑定")
	}

	_ = redisx.Del(l.ctx, codeKey)

	u, err := l.svcCtx.UserRepo.FindByID(l.ctx, userId)
	if err != nil {
		l.Logger.Errorf("FindByID: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, err.Error())
	}
	if u == nil {
		return nil, errorx.NewDefaultError(errorx.CodeUserNotFound)
	}

	info := userinfo.FromDAO(u)
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

func (l *BindContactLogic) getUserIdFromCtx() int64 {
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
