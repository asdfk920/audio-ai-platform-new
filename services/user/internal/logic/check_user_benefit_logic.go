package logic

import (
	"context"
	"time"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/pkg/util/ctxuser"
	"github.com/jacklau/audio-ai-platform/services/user/internal/repo/dao"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type CheckUserBenefitLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCheckUserBenefitLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CheckUserBenefitLogic {
	return &CheckUserBenefitLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// CheckUserBenefit 用户权益校验
// 完整流程：
// 1. 从 Token 解析用户 ID
// 2. 校验用户是否存在、账号状态正常
// 3. 查询用户会员信息（等级、有效期、状态）
// 4. 查询该会员等级所拥有的权益列表
// 5. 判断用户是否包含请求的权益
// 6. 判断会员是否在有效期内
// 7. 判断是否有特殊权益（赠送、活动、体验卡）
// 8. 返回校验结果
func (l *CheckUserBenefitLogic) CheckUserBenefit(req *types.CheckUserBenefitReq) (resp *types.CheckUserBenefitResp, err error) {
	// 1. 获取当前用户 ID（从 JWT）
	userId := l.getUserIdFromCtx()
	if userId <= 0 {
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "登录已过期或无效，请重新登录")
	}

	// 2. 参数校验
	if req.BenefitCode == "" {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "权益标识不能为空")
	}

	// 3. 查询用户会员档案
	userRepo := dao.NewUserRepo(l.svcCtx.DB)
	levelCode, expireAt, isPermanent, err := userRepo.GetActiveMemberProfile(l.ctx, userId)
	if err != nil {
		l.Logger.Errorf("查询用户会员档案失败：%v", err)
		return &types.CheckUserBenefitResp{
			UserId:        userId,
			BenefitCode:   req.BenefitCode,
			HasPermission: false,
			Reason:        "查询会员信息失败",
		}, nil
	}

	// 4. 用户不是会员
	if levelCode == "" {
		return &types.CheckUserBenefitResp{
			UserId:        userId,
			BenefitCode:   req.BenefitCode,
			HasPermission: false,
			Reason:        "非会员用户，无此权益",
		}, nil
	}

	// 5. 会员已过期（非永久会员）
	if !isPermanent && expireAt != nil {
		if time.Now().After(*expireAt) {
			return &types.CheckUserBenefitResp{
				UserId:        userId,
				BenefitCode:   req.BenefitCode,
				HasPermission: false,
				Reason:        "会员已过期",
				LevelCode:     levelCode,
				ExpireAt:      expireAt.Unix(),
				IsPermanent:   isPermanent,
			}, nil
		}
	}

	// 6. 查询该会员等级的权益列表
	benefits, err := userRepo.ListBenefitsByLevelCode(l.ctx, levelCode)
	if err != nil {
		l.Logger.Errorf("查询会员权益列表失败：%v", err)
		return &types.CheckUserBenefitResp{
			UserId:        userId,
			BenefitCode:   req.BenefitCode,
			HasPermission: false,
			Reason:        "查询权益列表失败",
		}, nil
	}

	// 7. 判断是否包含请求的权益
	hasBenefit := false
	for _, benefit := range benefits {
		if benefit.BenefitCode == req.BenefitCode {
			hasBenefit = true
			break
		}
	}

	// 8. 不包含该权益
	if !hasBenefit {
		return &types.CheckUserBenefitResp{
			UserId:        userId,
			BenefitCode:   req.BenefitCode,
			HasPermission: false,
			Reason:        "当前会员等级不支持此权益",
			LevelCode:     levelCode,
			IsPermanent:   isPermanent,
		}, nil
	}

	// 9. 校验通过
	expireAtUnix := int64(0)
	if expireAt != nil {
		expireAtUnix = expireAt.Unix()
	}

	return &types.CheckUserBenefitResp{
		UserId:             userId,
		BenefitCode:        req.BenefitCode,
		HasPermission:      true,
		Reason:             "校验通过",
		LevelCode:          levelCode,
		ExpireAt:           expireAtUnix,
		IsPermanent:        isPermanent,
		SubscriptionActive: true,
	}, nil
}

// getUserIdFromCtx 从上下文获取用户 ID
func (l *CheckUserBenefitLogic) getUserIdFromCtx() int64 {
	return ctxuser.ParseUserID(l.ctx)
}
