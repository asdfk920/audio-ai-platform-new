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

// GetAccountCancellationStatusLogic 查询注销状态逻辑处理
type GetAccountCancellationStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewGetAccountCancellationStatusLogic 创建查询注销状态逻辑处理实例
func NewGetAccountCancellationStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAccountCancellationStatusLogic {
	return &GetAccountCancellationStatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Get 查询注销状态
// 返回用户当前的注销状态，包括三种状态：
//   - normal: 正常状态（未申请注销）
//   - cooling_off: 冷静期中（已申请注销，等待冷静期结束）
//   - cancelled: 已注销（账号已永久删除）
//
// 返回说明：
//   - resp: 注销状态响应（包含状态码、冷静期结束时间）
//   - err: 错误信息
func (l *GetAccountCancellationStatusLogic) Get() (resp *types.GetAccountCancellationStatusResp, err error) {
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

	// 3. 判断注销状态
	// 根据用户表中的注销相关字段判断当前状态
	var phase string
	var coolingEndAt int64

	if user.AccountCancelledAt.Valid {
		// 已注销：account_cancelled_at 字段有值，表示账号已永久删除
		phase = "cancelled"
		coolingEndAt = 0
	} else if user.CancellationCoolingUntil.Valid && user.CancellationCoolingUntil.Time.After(time.Now()) {
		// 冷静期中：cancellation_cooling_until 字段有值且未过期
		phase = "cooling_off"
		coolingEndAt = user.CancellationCoolingUntil.Time.Unix() // 返回冷静期结束时间戳
	} else {
		// 正常状态：未申请注销或已撤销注销申请
		phase = "normal"
		coolingEndAt = 0
	}

	// 4. 返回状态响应
	return &types.GetAccountCancellationStatusResp{
		Phase:        phase,        // 状态标识
		CoolingEndAt: coolingEndAt, // 冷静期结束时间戳（秒级）
	}, nil
}
