package logic

import (
	"context"
	"fmt"
	"time"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/pkg/util/ctxuser"
	"github.com/jacklau/audio-ai-platform/services/user/internal/repo/dao"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/jacklau/audio-ai-platform/services/user/internal/userdomain/member/memberpay"
	"github.com/zeromicro/go-zero/core/logx"
)

// CreateMemberOrderLogic 创建会员订单逻辑处理
type CreateMemberOrderLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewCreateMemberOrderLogic 创建创建会员订单逻辑处理实例
func NewCreateMemberOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateMemberOrderLogic {
	return &CreateMemberOrderLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// CreateMemberOrder 创建会员订单
// 参数说明：
//   - req: 创建订单请求（包含套餐编码、支付方式）
//
// 返回说明：
//   - resp: 订单响应（包含订单号、金额、支付信息等）
//   - err: 错误信息
func (l *CreateMemberOrderLogic) CreateMemberOrder(req *types.CreateMemberOrderReq) (resp *types.CreateMemberOrderResp, err error) {
	// 1. 获取当前用户 ID（从 JWT Token 中解析）
	userID := ctxuser.ParseUserID(l.ctx)
	if userID == 0 {
		// 未登录或 Token 无效
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "未登录")
	}

	// 2. 参数校验
	// 校验支付方式是否合法（1 微信 2 支付宝 3 余额）
	if req.PayType < 1 || req.PayType > 3 {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "支付方式不合法（1 微信 2 支付宝 3 余额）")
	}

	// 3. 查询会员套餐信息
	// 从数据库查询套餐是否存在、是否上架、价格等信息
	pkg, err := l.svcCtx.MemberOrder.GetMemberPackageByCode(l.ctx, req.PackageCode)
	if err != nil {
		logx.Errorf("get member package failed: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "查询套餐失败")
	}
	if pkg == nil {
		// 套餐不存在或已下架
		return nil, errorx.NewCodeError(errorx.CodeMemberPackageNotFound, "会员套餐不存在或已下架")
	}

	// 4. 校验用户状态
	// 检查用户是否可购买（账号状态正常、未被禁用等）
	user, err := l.svcCtx.UserRepo.FindByID(l.ctx, userID)
	if err != nil {
		logx.Errorf("load user failed: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "系统繁忙")
	}
	if user == nil {
		return nil, errorx.NewCodeError(errorx.CodeUserNotFound, "用户不存在")
	}

	// 5. 生成唯一订单号
	// 格式：M + 毫秒时间戳 + 8 位随机数
	orderNo := memberpay.NewOrderNo()

	// 6. 续费规则与场景（与履约一致）
	now := time.Now()
	renewSt, err := l.svcCtx.MemberOrder.GetUserMemberRenewalState(l.ctx, userID)
	if err != nil {
		logx.Errorf("renewal state failed: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "系统繁忙")
	}
	previewExp, bizScene := dao.ComputeMemberRenewal(now, renewSt, pkg.DurationDays)

	origCent := pkg.ListPriceCent
	if origCent <= 0 {
		origCent = pkg.PriceCent
	}
	discountCent := origCent - pkg.PriceCent
	if discountCent < 0 {
		discountCent = 0
		origCent = pkg.PriceCent
	}

	// 7. 创建订单（待支付）
	err = l.svcCtx.MemberOrder.InsertMemberOrder(l.ctx, orderNo, userID, pkg, int16(req.PayType), origCent, discountCent, bizScene)
	if err != nil {
		logx.Errorf("insert member order failed: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "创建订单失败")
	}

	// 8. 返回订单信息
	return &types.CreateMemberOrderResp{
		OrderNo:            orderNo,
		AmountCent:         pkg.PriceCent,
		AmountYuan:         fmt.Sprintf("%.2f", float64(pkg.PriceCent)/100),
		OriginalAmountCent: origCent,
		DiscountCent:       discountCent,
		BizScene:           bizScene,
		PackageName:        pkg.PackageName,
		PackageCode:        pkg.PackageCode,
		DurationDays:       int64(pkg.DurationDays),
		PreviewExpireAt:    previewExp.Unix(),
		PayType:            req.PayType,
		PayParams:          nil,
	}, nil
}
