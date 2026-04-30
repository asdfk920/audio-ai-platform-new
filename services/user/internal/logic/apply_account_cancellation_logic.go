package logic

import (
	"context"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/pkg/util/ctxuser"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/jacklau/audio-ai-platform/services/user/internal/userdomain/profile/accountcancel"
	"github.com/zeromicro/go-zero/core/logx"
)

// ApplyAccountCancellationLogic 申请注销账号逻辑处理
type ApplyAccountCancellationLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewApplyAccountCancellationLogic 创建申请注销账号逻辑处理实例
func NewApplyAccountCancellationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ApplyAccountCancellationLogic {
	return &ApplyAccountCancellationLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Apply 申请注销账号
// 参数说明：
//   - req: 注销申请请求参数（包含密码、原因、协议确认）
//   - clientIP: 客户端 IP 地址（用于日志记录）
//   - userAgent: 客户端设备信息（用于日志记录）
//
// 返回说明：
//   - resp: 注销申请响应（包含冷静期结束时间）
//   - err: 错误信息
func (l *ApplyAccountCancellationLogic) Apply(req *types.ApplyAccountCancellationReq, clientIP, userAgent string) (resp *types.ApplyAccountCancellationResp, err error) {
	// 1. 获取当前用户 ID（从 JWT Token 中解析）
	userID := ctxuser.ParseUserID(l.ctx)
	if userID == 0 {
		// 未登录或 Token 无效
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "未登录")
	}

	// 2. 加载用户信息（从数据库查询用户基本信息）
	user, err := l.svcCtx.UserRepo.FindByID(l.ctx, userID)
	if err != nil {
		// 数据库查询失败，记录错误日志
		logx.Errorf("load user failed: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "系统繁忙")
	}
	if user == nil {
		// 用户不存在
		return nil, errorx.NewCodeError(errorx.CodeUserNotFound, "用户不存在")
	}

	// 3. 校验账号状态
	// 检查账号是否已注销或处于冷静期（冷静期内禁止重复申请）
	if err := accountcancel.ErrIfClosedOrCooling(user); err != nil {
		// 账号已注销或冷静期中，返回相应错误
		return nil, err
	}

	// 4. 校验密码（可选，安全增强）：尚未接入密码校验，传入 password 时直接拒绝以免误传即放行
	if strings.TrimSpace(req.Password) != "" {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "当前版本注销申请不支持密码校验，请勿填写 password")
	}

	// 5. 校验注销前置条件
	// 检查用户是否满足注销条件（无未完成订单、无余额、无未到期会员等）
	if err := l.checkCancellationPrerequisites(userID); err != nil {
		// 前置条件不满足，返回错误提示
		return nil, err
	}

	// 6. 计算冷静期结束时间
	// 根据配置文件中的冷静期天数设置（默认 7 天，范围 7-30 天）
	coolingDays := l.svcCtx.Config.Cancellation.EffectiveCoolingDays()
	if coolingDays < 7 {
		// 低于最小值，强制设置为 7 天（符合《网络安全法》要求）
		coolingDays = 7
	}
	if coolingDays > 30 {
		// 超过最大值，限制为 30 天
		coolingDays = 30
	}
	// 计算冷静期结束时间（当前时间 + 冷静期天数）
	coolingEndAt := time.Now().Add(time.Duration(coolingDays) * 24 * time.Hour)

	// 7. 提交注销申请
	// 将注销原因转换为指针类型（空字符串转为 nil）
	reason := &req.Reason
	if req.Reason == "" {
		reason = nil
	}

	// 调用数据仓库层方法，在事务中完成注销申请
	// 包括：插入注销日志、更新用户冷静期结束时间
	err = l.svcCtx.UserRepo.ApplyAccountCancellationTx(l.ctx, userID, reason, &clientIP, &userAgent, coolingEndAt)
	if err != nil {
		// 申请失败，记录错误日志
		logx.Errorf("apply cancellation failed: %v", err)
		return nil, err
	}

	// 8. 返回成功响应
	// 返回冷静期结束时间戳（秒级 Unix 时间戳）
	return &types.ApplyAccountCancellationResp{
		CoolingEndAt: coolingEndAt.Unix(),
	}, nil
}

// checkCancellationPrerequisites 校验注销前置条件
// 确保用户账号满足注销条件，避免注销后产生业务纠纷
func (l *ApplyAccountCancellationLogic) checkCancellationPrerequisites(userID int64) error {
	// TODO: 校验未完成订单
	// 检查用户是否有待支付、待发货、待收货的订单

	// TODO: 校验会员权益
	// 检查用户是否有未到期的会员服务

	// TODO: 校验余额和收益
	// 检查用户账户是否有余额、未提现收益

	// TODO: 校验设备绑定
	// 检查用户是否已解绑所有设备

	// TODO: 校验未到期优惠券
	// 检查用户是否有未使用的优惠券

	// 暂时返回 nil，后续根据业务需求补充具体校验逻辑
	return nil
}
