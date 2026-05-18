package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/repo/dao"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CancelDeviceShareLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCancelDeviceShareLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CancelDeviceShareLogic {
	return &CancelDeviceShareLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CancelDeviceShareLogic) CancelDeviceShare(req *types.DeviceShareCancelReq) error {

	fmt.Println("========== [撤销设备共享] 开始 ==========")
	sn := strings.TrimSpace(req.Sn)
	fmt.Printf("请求参数: shareId=%d, sn=%s\n", req.ShareId, sn)

	userId := l.getUserIdFromCtx()
	fmt.Printf("从Context获取userId: %d\n", userId)
	if userId == 0 {
		fmt.Println("❌ 错误: 用户未登录 (userId=0)")
		return errorx.NewCodeError(errorx.CodeTokenInvalid, "登录已过期")
	}
	fmt.Println("✅ 步骤1: 鉴权通过")

	if req.ShareId == 0 && sn == "" {
		fmt.Println("❌ 错误: share_id 与 sn 至少填一项")
		return errorx.NewCodeError(errorx.CodeInvalidParam, "请提供 share_id 或设备 sn")
	}
	fmt.Println("✅ 步骤2: 参数校验通过")

	tx, err := l.svcCtx.DB.BeginTx(l.ctx, nil)
	if err != nil {
		fmt.Printf("❌ 错误: 开启事务失败 - %v\n", err)
		return errorx.NewCodeError(errorx.CodeInternalError, "系统繁忙，请稍后重试")
	}
	defer func() { _ = tx.Rollback() }()
	fmt.Println("✅ 步骤3: 事务开启成功")

	if sn != "" && req.ShareId == 0 {
		n, revokeErr := dao.RevokeActiveSharesByOwnerAndSN(l.ctx, tx, userId, sn)
		if revokeErr != nil {
			fmt.Printf("❌ 错误: 按 SN 撤销共享失败 - %v\n", revokeErr)
			return errorx.NewCodeError(errorx.CodeInternalError, "系统繁忙，请稍后重试")
		}
		if n == 0 {
			fmt.Printf("❌ 错误: 无匹配的可撤销共享 (owner=%d, sn=%s)\n", userId, sn)
			return errorx.NewCodeError(errorx.CodeDeviceShareNotFound, "该设备暂无有效共享记录可撤销")
		}
		fmt.Printf("✅ 已撤销 %d 条共享记录（按设备 SN）\n", n)
		if commitErr := tx.Commit(); commitErr != nil {
			fmt.Printf("❌ 错误: 提交事务失败 - %v\n", commitErr)
			return errorx.NewCodeError(errorx.CodeInternalError, "系统繁忙，请稍后重试")
		}
		fmt.Println("========== [撤销设备共享] 成功 ==========")
		return nil
	}

	share, findErr := dao.FindDeviceShareByID(l.ctx, tx, req.ShareId)
	if findErr != nil {
		fmt.Printf("❌ 错误: 查询共享记录失败 - %v\n", findErr)
		return errorx.NewCodeError(errorx.CodeInternalError, "系统错误")
	}
	if share == nil {
		fmt.Println("❌ 错误: 共享记录不存在")
		return errorx.NewCodeError(errorx.CodeDeviceShareNotFound, "共享记录不存在")
	}

	fmt.Printf("✅ 查询到共享记录:\n")
	fmt.Printf("  - ShareID: %d\n", share.ID)
	fmt.Printf("  - DeviceSN: %s\n", share.DeviceSN)
	fmt.Printf("  - OwnerUserID: %d (设备所有者)\n", share.OwnerUserID)
	fmt.Printf("  - SharerUserID: %d (发起共享)\n", share.SharerUserID)
	fmt.Printf("  - TargetAccount: %s\n", share.TargetAccount)
	fmt.Printf("  - Status: %s\n", share.Status)

	if share.OwnerUserID != userId && share.SharerUserID != userId {
		fmt.Printf("❌ 错误: 无权限撤销此共享 (当前用户=%d, 所有者=%d, 分享者=%d)\n",
			userId, share.OwnerUserID, share.SharerUserID)
		return errorx.NewCodeError(errorx.CodeDeviceShareNoCancelPermission, "无权限撤销此共享")
	}
	fmt.Println("✅ 步骤4: 权限验证通过（是设备所有者或分享者）")

	if share.Status == dao.DeviceShareStatusRevoked {
		fmt.Println("❌ 错误: 该共享已被取消")
		return errorx.NewCodeError(errorx.CodeDeviceShareAlreadyCanceled, "该共享已取消")
	}

	if share.Status == dao.DeviceShareStatusExpired {
		fmt.Println("❌ 错误: 该共享已过期")
		return errorx.NewCodeError(errorx.CodeDeviceShareAlreadyCanceled, "该共享已过期")
	}

	if share.Status == dao.DeviceShareStatusRejected {
		fmt.Println("❌ 错误: 该共享已被拒绝")
		return errorx.NewCodeError(errorx.CodeDeviceShareAlreadyCanceled, "该共享已被拒绝")
	}

	if share.Status == dao.DeviceShareStatusQuit {
		fmt.Println("❌ 错误: 对方已退出共享")
		return errorx.NewCodeError(errorx.CodeDeviceShareAlreadyCanceled, "对方已退出共享")
	}

	if share.Status != dao.DeviceShareStatusActive && share.Status != dao.DeviceShareStatusPending {
		fmt.Printf("❌ 错误: 该共享记录状态异常，无法撤销 (status=%s)\n", share.Status)
		return errorx.NewCodeError(errorx.CodeDeviceShareAlreadyCanceled, "该共享记录无法撤销")
	}
	fmt.Println("✅ 步骤5: 状态检查通过（可以撤销）")

	updateErr := dao.UpdateDeviceShareStatus(l.ctx, tx, req.ShareId, dao.DeviceShareStatusRevoked)
	if updateErr != nil {
		fmt.Printf("❌ 错误: 更新状态失败 - %v\n", updateErr)
		return errorx.NewCodeError(errorx.CodeInternalError, "系统错误")
	}
	fmt.Println("✅ 步骤6: 状态更新为已撤销（revoked）")

	if commitErr := tx.Commit(); commitErr != nil {
		fmt.Printf("❌ 错误: 提交事务失败 - %v\n", commitErr)
		return errorx.NewCodeError(errorx.CodeInternalError, "系统繁忙，请稍后重试")
	}
	fmt.Println("✅ 步骤7: 事务提交成功")

	fmt.Println("========== [撤销设备共享] 成功 ==========")
	return nil
}

func (l *CancelDeviceShareLogic) getUserIdFromCtx() int64 {
	v := l.ctx.Value("userId")
	if v == nil {
		return 0
	}
	switch id := v.(type) {
	case int64:
		return id
	case int:
		return int64(id)
	case float64:
		return int64(id)
	case json.Number:
		n, _ := id.Int64()
		return n
	case string:
		n, _ := strconv.ParseInt(id, 10, 64)
		return n
	default:
		return 0
	}
}
