package logic

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

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

func (l *CreateDeviceShareLogic) CreateDeviceShare(req *types.DeviceShareCreateReq) (resp *types.DeviceShareItem, err error) {
	defer func() {
		if r := recover(); r != nil {
			l.Logger.Errorf("CreateDeviceShare: 发生异常, panic=%v, stack=\n%s", r, debug.Stack())
			err = errorx.NewCodeError(errorx.CodeInternalError, "系统繁忙，请稍后重试")
		}
	}()

	userId := l.getUserIdFromCtx()
	if userId == 0 {
		l.Logger.Errorf("CreateDeviceShare: 获取用户ID失败 - 用户未登录或Token已过期")
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "用户未登录")
	}

	sn := strings.TrimSpace(req.Sn)
	if sn == "" {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "设备SN不能为空")
	}

	if len(sn) < 6 || len(sn) > 64 {
		return nil, errorx.NewCodeError(errorx.CodeDeviceSnInvalid, "设备SN格式错误")
	}

	targetAccount := strings.TrimSpace(req.ShareTo)
	if targetAccount == "" {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "请输入被共享人的手机号或邮箱")
	}

	expireDays := req.ExpireDays
	if expireDays <= 0 {
		expireDays = 7
	}
	if expireDays > 365 {
		expireDays = 365
	}

	tx, err := l.svcCtx.DB.BeginTx(l.ctx, nil)
	if err != nil {
		l.Logger.Errorf("CreateDeviceShare: 开启事务失败, err=%v", err)
		return nil, errorx.NewCodeError(errorx.CodeInternalError, "系统繁忙，请稍后重试")
	}
	defer func() { _ = tx.Rollback() }()

	bind, err := dao.FindActiveBindByUserAndSN(l.ctx, tx, userId, sn)
	if err != nil {
		l.Logger.Errorf("CreateDeviceShare: 查询绑定关系失败, userId=%d, sn=%s, err=%v", userId, sn, err)
		return nil, errorx.NewCodeError(errorx.CodeInternalError, "系统繁忙，请稍后重试")
	}
	if bind == nil {
		l.Logger.Errorf("CreateDeviceShare: 用户未绑定该设备, userId=%d, sn=%s", userId, sn)
		return nil, errorx.NewCodeError(errorx.CodeDeviceNotBound, "您未绑定此设备，无法共享")
	}

	targetUser, findErr := dao.FindUserByAccount(l.ctx, tx, targetAccount, 0)
	if findErr != nil {
		l.Logger.Errorf("CreateDeviceShare: 查找目标用户失败, account=%s, err=%v", targetAccount, findErr)
		return nil, errorx.NewCodeError(errorx.CodeInternalError, "系统繁忙，请稍后重试")
	}
	if targetUser == nil {
		l.Logger.Errorf("CreateDeviceShare: 目标用户不存在, account=%s", targetAccount)
		return nil, errorx.NewCodeError(errorx.CodeUserNotFound, "目标用户不存在，请检查账号是否正确")
	}

	if targetUser.ID == userId {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "不能将设备共享给自己")
	}

	existing, err := dao.FindActiveShareForDeviceUser(l.ctx, tx, bind.DeviceID, targetUser.ID)
	if err != nil {
		l.Logger.Errorf("CreateDeviceShare: 检查重复共享失败, err=%v", err)
		return nil, errorx.NewCodeError(errorx.CodeInternalError, "系统繁忙，请稍后重试")
	}
	if existing != nil {
		l.Logger.Infof("CreateDeviceShare: 已存在共享记录, shareId=%d", existing.ID)
		return nil, errorx.NewCodeError(errorx.CodeDeviceShareExists, "已向该用户发起过设备共享，无需重复操作")
	}

	endAt := time.Now().AddDate(0, 0, expireDays)

	inviteCode, codeErr := generateInviteCode()
	if codeErr != nil {
		l.Logger.Errorf("CreateDeviceShare: 生成邀请码失败, err=%v", codeErr)
		return nil, errorx.NewCodeError(errorx.CodeInternalError, "系统繁忙，请稍后重试")
	}

	shareRow, insertErr := dao.InsertDeviceShare(l.ctx, tx, dao.DeviceShareRow{
		FamilyID:        1,
		DeviceID:        bind.DeviceID,
		DeviceSN:        sn,
		DeviceName:      bind.DeviceName,
		OwnerUserID:     userId,
		SharedUserID:    targetUser.ID,
		TargetAccount:   firstNonEmpty(targetUser.Email.String, targetUser.Mobile.String),
		InviteCode:      inviteCode,
		ShareType:       "permanent",
		PermissionLevel: "view_only",
		PermissionRaw:   []byte(`{}`),
		EndAt:           sql.NullTime{Time: endAt, Valid: true},
		Status:          dao.DeviceShareStatusPending,
		CreatedBy:       userId,
	})
	if insertErr != nil {
		l.Logger.Errorf("CreateDeviceShare: 创建共享记录失败, err=%v", insertErr)
		return nil, errorx.NewCodeError(errorx.CodeInternalError, "系统繁忙，请稍后重试")
	}

	if commitErr := tx.Commit(); commitErr != nil {
		l.Logger.Errorf("CreateDeviceShare: 提交事务失败, err=%v", commitErr)
		return nil, errorx.NewCodeError(errorx.CodeInternalError, "系统繁忙，请稍后重试")
	}

	l.Logger.Infof("CreateDeviceShare: 共享邀请创建成功, userId=%d, sn=%s, targetUserId=%d, shareId=%d, expireDays=%d",
		userId, sn, targetUser.ID, shareRow.ID, expireDays)

	return &types.DeviceShareItem{
		ShareId:    shareRow.ID,
		Sn:         shareRow.DeviceSN,
		FromUserId: shareRow.OwnerUserID,
		ToUserId:   shareRow.SharedUserID,
		ToAccount:  shareRow.TargetAccount,
		Status:     statusToInt(shareRow.Status),
		CreatedAt:  shareRow.CreatedAt.Format("2006-01-02 15:04:05"),
		EndAt:      formatNullTime(shareRow.EndAt),
	}, nil
}

func (l *CreateDeviceShareLogic) getUserIdFromCtx() int64 {
	v := l.ctx.Value("userId")
	if v == nil {
		l.Logger.Errorf("CreateDeviceShare: Context中未找到userId, ctx keys: 检查中间件是否正确设置")
		return 0
	}

	l.Logger.Infof("CreateDeviceShare: 从Context获取到userId, type=%T, value=%v", v, v)

	switch id := v.(type) {
	case int64:
		return id
	case int:
		return int64(id)
	case float64:
		return int64(id)
	case json.Number:
		n, err := id.Int64()
		if err != nil {
			l.Logger.Errorf("CreateDeviceShare: json.Number转换失败, value=%s, err=%v", string(id), err)
			return 0
		}
		return n
	case string:
		n, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			l.Logger.Errorf("CreateDeviceShare: string转换失败, value=%s, err=%v", id, err)
			return 0
		}
		return n
	default:
		l.Logger.Errorf("CreateDeviceShare: userId类型不支持, type=%T, value=%v", v, v)
		return 0
	}
}

func generateInviteCode() (string, error) {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "SHR" + strings.ToUpper(hex.EncodeToString(buf)), nil
}
