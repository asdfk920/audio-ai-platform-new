package logic

import (
	"context"
	"time"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/pkg/util/ctxuser"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// GetUserMemberBenefitsLogic 获取用户会员权益逻辑处理
type GetUserMemberBenefitsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewGetUserMemberBenefitsLogic 创建获取用户会员权益逻辑处理实例
func NewGetUserMemberBenefitsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserMemberBenefitsLogic {
	return &GetUserMemberBenefitsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// GetUserMemberBenefits 获取用户会员权益
// 参数说明：
//   - req: 查询请求（空）
//
// 返回说明：
//   - resp: 会员权益响应（包含会员等级、权益列表等）
//   - err: 错误信息
func (l *GetUserMemberBenefitsLogic) GetUserMemberBenefits(req *types.GetUserMemberBenefitsReq) (resp *types.GetUserMemberBenefitsResp, err error) {
	// 1. 获取当前用户 ID（从 JWT Token 中解析）
	userID := ctxuser.ParseUserID(l.ctx)
	if userID == 0 {
		// 未登录或 Token 无效
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "未登录")
	}

	// 2. 查询用户会员信息
	member, err := l.svcCtx.MemberOrder.GetUserMemberInfo(l.ctx, userID)
	if err != nil {
		logx.Errorf("get user member info failed: user_id=%d, err=%v", userID, err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "查询会员信息失败")
	}

	// 3. 判断会员状态
	now := time.Now()
	var memberLevel int64
	var levelName string
	var status int64
	var isValid bool

	if member == nil {
		memberLevel = 0
		levelName = "非会员"
		status = 0
		isValid = false
	} else {
		memberLevel = l.parseLevelCode(member.LevelCode)
		levelName = l.getLevelName(member.LevelCode)
		status = int64(member.Status)
		isValid = member.Status == 1 && (member.IsPermanent == 1 || member.ExpireAt.After(now))
	}

	// 4. 获取对应等级的权益列表
	benefitList := l.getBenefitList(memberLevel, isValid)

	// 5. 封装返回结果
	resp = &types.GetUserMemberBenefitsResp{
		MemberLevel:       memberLevel,
		LevelName:         levelName,
		Status:            status,
		IsValid:           isValid,
		SubscriptionPhase: "none",
		BenefitList:       benefitList,
	}
	if member != nil {
		resp.SubscriptionPhase = subscriptionPhase(member, now)
		resp.CancelPending = member.CancelPending == 1
		resp.AutoRenew = member.AutoRenew == 1
		resp.AutoRenewPackageCode = member.AutoRenewPackageCode
		if member.AutoRenewPayType > 0 {
			resp.AutoRenewPayType = int64(member.AutoRenewPayType)
		}
		if member.AutoRenewUpdatedAt.Valid {
			resp.AutoRenewUpdatedAt = member.AutoRenewUpdatedAt.Time.Unix()
		}
	}

	return resp, nil
}

// parseLevelCode 解析会员等级编码为数字等级
func (l *GetUserMemberBenefitsLogic) parseLevelCode(levelCode string) int64 {
	switch levelCode {
	case "vip_monthly":
		return 1 // 月度会员
	case "vip_quarterly":
		return 2 // 季度会员
	case "vip_yearly":
		return 3 // 年度会员
	default:
		return 0 // 非会员
	}
}

// getLevelName 根据等级编码返回等级名称
func (l *GetUserMemberBenefitsLogic) getLevelName(levelCode string) string {
	switch levelCode {
	case "vip_monthly":
		return "月度会员"
	case "vip_quarterly":
		return "季度会员"
	case "vip_yearly":
		return "年度会员"
	case "", "none":
		return "非会员"
	default:
		return "会员"
	}
}

// getBenefitList 根据会员等级获取权益列表
// 参数：
//   - memberLevel: 会员等级（0 非会员 1 月度 2 季度 3 年度）
//   - isValid: 会员是否有效
//
// 返回：
//   - benefitList: 权益列表
func (l *GetUserMemberBenefitsLogic) getBenefitList(memberLevel int64, isValid bool) []*types.MemberBenefit {
	// 定义所有可能的权益
	allBenefits := map[int64][]*types.MemberBenefit{
		0: { // 非会员
			{
				BenefitCode: "standard_quality",
				BenefitName: "标准音质",
				Description: "128kbps 标准音质播放",
			},
			{
				BenefitCode: "basic_content",
				BenefitName: "基础内容",
				Description: "免费内容库收听",
			},
		},
		1: { // 月度会员
			{
				BenefitCode: "no_ads",
				BenefitName: "免广告",
				Description: "收听全程无广告干扰",
			},
			{
				BenefitCode: "high_quality",
				BenefitName: "标准音质",
				Description: "320kbps 高品质音质播放",
			},
			{
				BenefitCode: "multi_device",
				BenefitName: "多设备支持",
				Description: "支持多个设备同时使用（最多 2 个设备）",
			},
			{
				BenefitCode: "basic_content_library",
				BenefitName: "基础内容库",
				Description: "解锁全部基础付费内容",
			},
		},
		2: { // 季度会员
			{
				BenefitCode: "no_ads",
				BenefitName: "免广告",
				Description: "收听全程无广告干扰",
			},
			{
				BenefitCode: "high_quality",
				BenefitName: "高音质",
				Description: "320kbps 高品质音质播放",
			},
			{
				BenefitCode: "multi_device",
				BenefitName: "多设备支持",
				Description: "支持多个设备同时使用（最多 3 个设备）",
			},
			{
				BenefitCode: "basic_content_library",
				BenefitName: "基础内容库",
				Description: "解锁全部基础付费内容",
			},
			{
				BenefitCode: "vip_support",
				BenefitName: "专属客服",
				Description: "优先响应的专属客户服务",
			},
		},
		3: { // 年度会员
			{
				BenefitCode: "no_ads",
				BenefitName: "免广告",
				Description: "收听全程无广告干扰",
			},
			{
				BenefitCode: "lossless_quality",
				BenefitName: "无损音质",
				Description: "FLAC 无损音质播放",
			},
			{
				BenefitCode: "spatial_audio",
				BenefitName: "空间音频",
				Description: "沉浸式空间音频体验",
			},
			{
				BenefitCode: "multi_device",
				BenefitName: "多设备支持",
				Description: "支持多个设备同时使用（最多 5 个设备）",
			},
			{
				BenefitCode: "basic_content_library",
				BenefitName: "基础内容库",
				Description: "解锁全部基础付费内容",
			},
			{
				BenefitCode: "premium_content",
				BenefitName: "专属内容库",
				Description: "解锁年度会员专属内容",
			},
			{
				BenefitCode: "vip_support",
				BenefitName: "专属客服",
				Description: "优先响应的专属客户服务",
			},
			{
				BenefitCode: "priority_download",
				BenefitName: "优先下载",
				Description: "内容下载优先队列",
			},
			{
				BenefitCode: "priority_access",
				BenefitName: "优先体验",
				Description: "新功能优先体验权限",
			},
		},
	}

	// 根据等级返回对应的权益列表
	if benefits, ok := allBenefits[memberLevel]; ok {
		return benefits
	}

	// 默认返回非会员权益
	return allBenefits[0]
}
