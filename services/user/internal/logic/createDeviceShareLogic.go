package logic

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/repo/dao"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateDeviceShareLogic struct {
	Logger logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateDeviceShareLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateDeviceShareLogic {
	return &CreateDeviceShareLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateDeviceShareLogic) CreateDeviceShare(req *types.DeviceShareCreateReq) error {

	sn := strings.TrimSpace(req.Sn)
	shareTo := strings.TrimSpace(req.ShareTo)
	expireDays := req.ExpireDays

	fmt.Println("========== [设备共享创建] 开始 ==========")
	fmt.Printf("请求参数: sn=%s, shareTo=%s, expireDays=%d\n", sn, shareTo, expireDays)

	userId := l.getUserIdFromCtx()
	fmt.Printf("从Context获取userId: %d\n", userId)
	if userId == 0 {
		fmt.Println("❌ 错误: 用户未登录 (userId=0)")
		return errorx.NewCodeError(errorx.CodeTokenInvalid, "用户未登录")
	}
	fmt.Println("✅ 步骤1: 鉴权通过")

	if sn == "" {
		fmt.Println("❌ 错误: 设备SN为空")
		return errorx.NewCodeError(errorx.CodeInvalidParam, "设备SN不能为空")
	}

	if shareTo == "" {
		fmt.Println("❌ 错误: 被分享人账号为空")
		return errorx.NewCodeError(errorx.CodeInvalidParam, "请输入被分享人的邮箱")
	}
	fmt.Println("✅ 步骤2: 参数校验通过")

	tx, err := l.svcCtx.DB.BeginTx(l.ctx, nil)
	if err != nil {
		fmt.Printf("❌ 错误: 开启事务失败 - %v\n", err)
		return errorx.NewCodeError(errorx.CodeInternalError, "系统繁忙，请稍后重试")
	}
	defer func() { _ = tx.Rollback() }()
	fmt.Println("✅ 步骤3: 事务开启成功")

	bind, err := dao.FindActiveBindByUserAndSN(l.ctx, tx, userId, sn)
	if err != nil {
		fmt.Printf("❌ 错误: 查询设备绑定失败 - %v\n", err)
		return errorx.NewCodeError(errorx.CodeInternalError, "系统繁忙，请稍后重试")
	}
	if bind == nil {
		fmt.Println("❌ 错误: 设备不存在或无权限操作")
		return errorx.NewCodeError(errorx.CodeDeviceNotBound, "设备不存在或无权限操作")
	}
	fmt.Printf("✅ 步骤4: 设备检查通过 (deviceID=%d, deviceName=%s)\n", bind.DeviceID, bind.DeviceName)

	targetUser, err := dao.FindUserByAccount(l.ctx, tx, shareTo, 0)
	if err != nil {
		fmt.Printf("❌ 错误: 查询目标用户失败 - %v\n", err)
		return errorx.NewCodeError(errorx.CodeInternalError, "系统繁忙，请稍后重试")
	}
	if targetUser == nil {
		fmt.Printf("❌ 错误: 目标用户不存在 (%s)\n", shareTo)
		return errorx.NewCodeError(errorx.CodeUserNotFound, "目标用户不存在")
	}
	fmt.Printf("✅ 步骤5: 目标用户存在 (targetUserID=%d)\n", targetUser.ID)

	if targetUser.ID == userId {
		fmt.Println("❌ 错误: 不能共享给自己")
		return errorx.NewCodeError(errorx.CodeInvalidParam, "不能将设备共享给自己")
	}

	existing, err := dao.FindActiveShareForDeviceUser(l.ctx, tx, bind.DeviceID, targetUser.ID)
	if err != nil {
		fmt.Printf("❌ 错误: 查询重复共享失败 - %v\n", err)
		return errorx.NewCodeError(errorx.CodeInternalError, "系统繁忙，请稍后重试")
	}
	if existing != nil {
		fmt.Println("❌ 错误: 已经共享过该用户")
		return errorx.NewCodeError(errorx.CodeDeviceShareExists, "不能重复分享")
	}
	fmt.Println("✅ 步骤6: 重复共享检查通过")

	var endAt sql.NullTime
	if expireDays > 0 {
		endAt = sql.NullTime{Time: time.Now().AddDate(0, 0, expireDays), Valid: true}
	}

	targetAccount := firstNonEmpty(targetUser.Email.String, targetUser.Mobile.String)
	inviteCode := generateInviteCode()

	fmt.Printf("准备插入共享记录:\n")
	fmt.Printf("  - FamilyID: 1\n")
	fmt.Printf("  - DeviceID: %d\n", bind.DeviceID)
	fmt.Printf("  - DeviceSN: %s\n", sn)
	fmt.Printf("  - DeviceName: %s\n", bind.DeviceName)
	fmt.Printf("  - OwnerUserID: %d\n", userId)
	fmt.Printf("  - SharerUserID: %d (发起共享的用户)\n", userId)
	fmt.Printf("  - SharedUserID: %d\n", targetUser.ID)
	fmt.Printf("  - TargetAccount: %s\n", targetAccount)
	fmt.Printf("  - InviteCode: %s\n", inviteCode)
	fmt.Printf("  - ShareType: 0 (永久)\n")
	fmt.Printf("  - PermissionLevel: view_only\n")
	fmt.Printf("  - EndAt: %+v\n", endAt)
	fmt.Printf("  - Status: %s (待接收)\n", dao.DeviceShareStatusPending)

	shareRow, err := dao.InsertDeviceShare(l.ctx, tx, dao.DeviceShareRow{
		FamilyID:        1,
		DeviceID:        bind.DeviceID,
		DeviceSN:        sn,
		DeviceName:      bind.DeviceName,
		OwnerUserID:     userId,
		SharerUserID:    userId,
		SharedUserID:    targetUser.ID,
		TargetAccount:   targetAccount,
		InviteCode:      inviteCode,
		ShareType:       "0",
		PermissionLevel: "view_only",
		PermissionRaw:   []byte(`{}`),
		EndAt:           endAt,
		Status:          dao.DeviceShareStatusPending,
		CreatedBy:       userId,
	})
	if err != nil {
		fmt.Printf("❌ 错误: 插入共享记录失败 - %v\n", err)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return errorx.NewCodeError(errorx.CodeDeviceShareExists, "不能重复分享")
		}
		return errorx.NewCodeError(errorx.CodeInternalError, "系统繁忙，请稍后重试")
	}
	fmt.Printf("✅ 步骤7: 共享记录插入成功 (shareID=%d)\n", shareRow.ID)

	if err := tx.Commit(); err != nil {
		fmt.Printf("❌ 错误: 提交事务失败 - %v\n", err)
		return errorx.NewCodeError(errorx.CodeInternalError, "系统繁忙，请稍后重试")
	}
	fmt.Println("✅ 步骤8: 事务提交成功")

	fmt.Println("========== [设备共享创建] 成功 ==========")
	return nil
}

func (l *CreateDeviceShareLogic) getUserIdFromCtx() int64 {
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

func generateInviteCode() string {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return "SHR" + strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return "SHR" + strings.ToUpper(hex.EncodeToString(buf))
}
