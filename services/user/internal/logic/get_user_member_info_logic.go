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

// GetUserMemberInfoLogic 获取用户会员信息逻辑处理
type GetUserMemberInfoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewGetUserMemberInfoLogic 创建获取用户会员信息逻辑处理实例
func NewGetUserMemberInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserMemberInfoLogic {
	return &GetUserMemberInfoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// GetUserMemberInfo 获取用户会员信息
// 参数说明：
//   - req: 查询请求（空）
//
// 返回说明：
//   - resp: 会员信息响应（包含会员等级、有效期、权益等）
//   - err: 错误信息
func (l *GetUserMemberInfoLogic) GetUserMemberInfo(req *types.GetUserMemberInfoReq) (resp *types.GetUserMemberInfoResp, err error) {
	// 1. 获取当前用户 ID（从 JWT Token 中解析）
	userID := ctxuser.ParseUserID(l.ctx)
	if userID == 0 {
		// 未登录或 Token 无效
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "未登录")
	}

	// 2. 查询用户会员数据
	member, err := l.svcCtx.MemberOrder.GetUserMemberInfo(l.ctx, userID)
	if err != nil {
		logx.Errorf("get user member info failed: user_id=%d, err=%v", userID, err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "查询会员信息失败")
	}

	// 3. 封装返回结果
	resp = &types.GetUserMemberInfoResp{
		UserID: userID,
	}

	// 4. 处理会员信息
	if member == nil {
		// 无会员记录，按非会员处理
		resp.MemberLevel = 0
		resp.LevelName = "非会员"
		resp.Status = 0
		resp.ExpireTime = 0
		resp.IsExpire = true
		resp.BenefitList = []string{}
		resp.CreateTime = 0
		resp.RemainingDays = 0
		return resp, nil
	}

	// 5. 有会员记录，填充详细信息
	resp.MemberLevel = l.parseLevelCode(member.LevelCode)
	resp.LevelName = l.getLevelName(member.LevelCode)
	resp.Status = int64(member.Status)
	resp.ExpireTime = member.ExpireAt.Unix()
	resp.CreateTime = member.CreatedAt.Unix()

	now := time.Now()
	if member.IsPermanent == 1 {
		resp.IsExpire = false
		resp.RemainingDays = -1
	} else {
		resp.IsExpire = member.ExpireAt.Before(now)
		if resp.IsExpire {
			resp.RemainingDays = 0
		} else {
			resp.RemainingDays = int64(member.ExpireAt.Sub(now).Hours() / 24)
		}
	}

	resp.BenefitList = l.getBenefitList(member.LevelCode)
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

	return resp, nil
}

// parseLevelCode 解析会员等级编码为数字等级
func (l *GetUserMemberInfoLogic) parseLevelCode(levelCode string) int64 {
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
func (l *GetUserMemberInfoLogic) getLevelName(levelCode string) string {
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
func (l *GetUserMemberInfoLogic) getBenefitList(levelCode string) []string {
	// TODO: 从数据库或配置中心读取权益配置
	// 这里根据等级返回预设的权益列表
	benefits := map[string][]string{
		"vip_monthly": {
			"免广告",
			"标准音质",
			"最多 2 个设备",
			"基础内容库",
		},
		"vip_quarterly": {
			"免广告",
			"高音质",
			"最多 3 个设备",
			"基础内容库",
			"专属客服",
		},
		"vip_yearly": {
			"免广告",
			"无损音质",
			"最多 5 个设备",
			"全部内容库",
			"专属客服",
			"优先体验新功能",
		},
	}

	if benefits, ok := benefits[levelCode]; ok {
		return benefits
	}
	return []string{}
}

// GetMemberStatusName 获取会员状态名称
func (l *GetUserMemberInfoLogic) GetMemberStatusName(status int64) string {
	statusNames := map[int64]string{
		0: "未开通",
		1: "正常",
		2: "已过期",
		3: "已冻结",
	}

	if name, ok := statusNames[status]; ok {
		return name
	}
	return "未知"
}
