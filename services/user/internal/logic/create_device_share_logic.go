package logic

import (
	"context"
	"encoding/json"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/devicesharesvc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/pkg/util/ctxuser"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type CreateDeviceShareLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateDeviceShareLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateDeviceShareLogic {
	return &CreateDeviceShareLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *CreateDeviceShareLogic) CreateDeviceShare(req *types.DeviceShareCreateReq) (*types.DeviceShareItem, error) {
	userID := ctxuser.ParseUserID(l.ctx)
	if userID <= 0 {
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "")
	}
	var permission map[string]any
	if req.Permission != "" {
		if err := json.Unmarshal([]byte(req.Permission), &permission); err != nil {
			return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "permission 必须是合法 JSON")
		}
	}
	view, err := devicesharesvc.New(l.svcCtx).CreateShareInvite(l.ctx, devicesharesvc.CreateShareInviteInput{
		OperatorUserID:  userID,
		DeviceSN:        req.DeviceSn,
		TargetUserID:    req.TargetUserId,
		TargetAccount:   req.TargetAccount,
		ShareType:       req.ShareType,
		PermissionLevel: req.PermissionLevel,
		Permission:      permission,
		StartAt:         unixPtr(req.StartAt),
		EndAt:           unixPtr(req.EndAt),
		Remark:          req.Remark,
	})
	if err != nil {
		return nil, err
	}
	return toDeviceShareItem(view), nil
}
