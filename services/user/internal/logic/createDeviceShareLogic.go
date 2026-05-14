package logic

import (
	"context"
	"database/sql"
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
	userId := l.getUserIdFromCtx()
	if userId == 0 {
		l.Logger.Errorf("CreateDeviceShare: 获取用户ID失败")
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "请重新登录")
	}

	if req.Sn == "" {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "设备SN不能为空")
	}
	if req.ToUserId == 0 && req.ShareTo == "" {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "被共享人不能为空")
	}

	tx, err := l.svcCtx.DB.BeginTx(l.ctx, nil)
	if err != nil {
		l.Logger.Errorf("CreateDeviceShare: 开启事务失败, err=%v", err)
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	bind, err := dao.FindActiveBindByUserAndSN(l.ctx, tx, userId, req.Sn)
	if err != nil {
		l.Logger.Errorf("CreateDeviceShare: 查询绑定关系失败, err=%v", err)
		return nil, err
	}
	if bind == nil {
		return nil, errorx.NewCodeError(errorx.CodeDeviceNotBound, "无权限共享此设备")
	}

	var targetUserId int64 = req.ToUserId
	var targetAccount string
	if targetUserId == 0 && req.ShareTo != "" {
		targetUser, findErr := dao.FindUserByAccount(l.ctx, tx, req.ShareTo, 0)
		if findErr != nil {
			l.Logger.Errorf("CreateDeviceShare: 查找目标用户失败, account=%s, err=%v", req.ShareTo, findErr)
			return nil, findErr
		}
		if targetUser == nil {
			return nil, errorx.NewCodeError(errorx.CodeUserNotFound, "目标用户不存在")
		}
		targetUserId = targetUser.ID
		targetAccount = firstNonEmpty(targetUser.Email.String, targetUser.Mobile.String)
	} else if targetUserId > 0 {
		targetUser, findErr := dao.FindUserByID(l.ctx, tx, targetUserId)
		if findErr != nil {
			l.Logger.Errorf("CreateDeviceShare: 查找目标用户失败, userId=%d, err=%v", targetUserId, findErr)
			return nil, findErr
		}
		if targetUser != nil {
			targetAccount = firstNonEmpty(targetUser.Email.String, targetUser.Mobile.String)
		}
	}

	if targetUserId == userId {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "不能分享给自己")
	}

	existing, err := dao.FindActiveShareForDeviceUser(l.ctx, tx, bind.DeviceID, targetUserId)
	if err != nil {
		l.Logger.Errorf("CreateDeviceShare: 检查重复共享失败, err=%v", err)
		return nil, err
	}
	if existing != nil {
		return nil, errorx.NewCodeError(errorx.CodeDeviceShareExists, "已共享给该用户，无需重复发起")
	}

	endAt := time.Now().AddDate(0, 0, 7)

	shareRow, err := dao.InsertDeviceShare(l.ctx, tx, dao.DeviceShareRow{
		DeviceID:      bind.DeviceID,
		DeviceSN:      req.Sn,
		DeviceName:    bind.DeviceName,
		OwnerUserID:   userId,
		SharedUserID:  targetUserId,
		TargetAccount: targetAccount,
		Status:        dao.DeviceShareStatusPending,
		EndAt:         sql.NullTime{Time: endAt, Valid: true},
		CreatedBy:     userId,
	})
	if err != nil {
		l.Logger.Errorf("CreateDeviceShare: 创建共享记录失败, err=%v", err)
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		l.Logger.Errorf("CreateDeviceShare: 提交事务失败, err=%v", err)
		return nil, err
	}

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
		return 0
	}
	switch id := v.(type) {
	case int64:
		return id
	case int:
		return int64(id)
	case float64:
		return int64(id)
	default:
		return 0
	}
}
