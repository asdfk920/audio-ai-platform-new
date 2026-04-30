package logic

import (
	"context"
	"fmt"
	"regexp"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/pkg/passwd"
	"github.com/jacklau/audio-ai-platform/pkg/redisx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

var (
	regEmailRegex  = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	regMobileRegex = regexp.MustCompile(`^1[3-9]\d{9}$`)
)

type RegisterLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 用户注册（需先发验证码；按邮箱/手机判断是否已注册；密码加盐存储）
func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RegisterLogic) Register(req *types.RegisterReq) (resp *types.RegisterResp, err error) {
	target, by := l.normalizeTarget(req)
	if target == "" {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "请使用邮箱或手机号其中一种方式注册")
	}
	if req.Email != "" && req.Mobile != "" {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "邮箱与手机号只能填其中一项")
	}
	if by == "email" && !regEmailRegex.MatchString(req.Email) {
		return nil, errorx.NewDefaultError(errorx.CodeInvalidEmail)
	}
	if by == "mobile" && !regMobileRegex.MatchString(req.Mobile) {
		return nil, errorx.NewDefaultError(errorx.CodeInvalidMobile)
	}
	if req.Password == "" || len(req.Password) < 6 {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "密码不能少于6位")
	}

	codeKey := fmt.Sprintf("user:verify_code:%s", target)
	stored, err := redisx.Get(l.ctx, codeKey)
	if err != nil || stored == "" {
		return nil, errorx.NewDefaultError(errorx.CodeVerifyCodeInvalid)
	}
	if stored != req.VerifyCode {
		return nil, errorx.NewDefaultError(errorx.CodeVerifyCodeInvalid)
	}

	// 注册前根据注册方式判断该账号是否已注册
	repo := l.svcCtx.UserRepo
	if by == "email" {
		exist, _ := repo.FindByEmail(l.ctx, req.Email)
		if exist != nil {
			return nil, errorx.NewDefaultError(errorx.CodeUserExists)
		}
	} else {
		exist, _ := repo.FindByMobile(l.ctx, req.Mobile)
		if exist != nil {
			return nil, errorx.NewDefaultError(errorx.CodeUserExists)
		}
	}

	salt, hashed, err := passwd.HashPasswordWithNewSalt(req.Password)
	if err != nil {
		l.Logger.Errorf("hash password: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeSystemError, err.Error())
	}

	var email, mobile *string
	if by == "email" {
		email = &req.Email
	} else {
		mobile = &req.Mobile
	}
	nickname := &req.Nickname
	if req.Nickname == "" {
		nickname = nil
	}

	userID, err := repo.Create(l.ctx, email, mobile, &hashed, &salt, nickname, nil, 1)
	if err != nil {
		l.Logger.Errorf("create user: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, err.Error())
	}

	_ = redisx.Del(l.ctx, codeKey)

	return &types.RegisterResp{UserId: userID}, nil
}

func (l *RegisterLogic) normalizeTarget(req *types.RegisterReq) (target, by string) {
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
