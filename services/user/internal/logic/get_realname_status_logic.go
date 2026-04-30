package logic

import (
	"context"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/pkg/util/ctxuser"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/jacklau/audio-ai-platform/services/user/internal/userdomain/profile/realname"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetRealNameStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetRealNameStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetRealNameStatusLogic {
	return &GetRealNameStatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetRealNameStatusLogic) Get() (resp *types.RealNameStatusResp, err error) {
	userID := ctxuser.ParseUserID(l.ctx)
	if userID == 0 {
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "未登录")
	}

	user, err := l.svcCtx.UserRepo.RealNameTxLoadUserForUpdate(l.ctx, nil, userID)
	if err != nil {
		logx.Errorf("load user failed: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "系统繁忙")
	}
	if user == nil {
		return nil, errorx.NewCodeError(errorx.CodeUserNotFound, "用户不存在")
	}

	auth, err := l.svcCtx.UserRepo.RealNameFindLatestAuthByUser(l.ctx, userID)
	if err != nil {
		logx.Errorf("find latest auth failed: %v", err)
	}

	var subStatus string
	var message string
	var failReason string
	var updatedAt int64
	var realNameMasked string
	var idNumberMasked string
	var submittedAt int64
	var auditedAt int64

	if auth != nil {
		updatedAt = auth.CreatedAt.Unix()
		submittedAt = auth.CreatedAt.Unix()

		if auth.ReviewedAt.Valid {
			auditedAt = auth.ReviewedAt.Time.Unix()
		}

		switch auth.AuthStatus {
		case realname.AuthPendingThirdParty:
			subStatus = "third_party_verifying"
			message = "实名认证信息已提交，等待第三方核验"
		case realname.AuthThirdPartyPass:
			subStatus = "pending_manual"
			message = "第三方核验通过，等待人工审核"
		case realname.AuthThirdPartyFail:
			subStatus = "third_party_failed"
			message = "第三方核验失败"
			if auth.FailReason.Valid {
				failReason = auth.FailReason.String
			}
		case realname.AuthPendingManual:
			subStatus = "manual_reviewing"
			message = "人工审核中"
		case realname.AuthManualPass:
			subStatus = "approved"
			message = "实名认证已通过"
		case realname.AuthManualReject:
			subStatus = "rejected"
			message = "实名认证被驳回"
			if auth.ReviewerNote.Valid {
				failReason = auth.ReviewerNote.String
			}
		default:
			subStatus = "unknown"
			message = "实名认证状态未知"
		}
	} else {
		if user.RealNameAt.Valid {
			updatedAt = user.RealNameAt.Time.Unix()
		} else {
			updatedAt = 0
		}
	}

	switch user.RealNameStatus {
	case realname.UserRealNameNone:
		if message == "" {
			message = "未进行实名认证"
		}
	case realname.UserRealNameVerified:
		message = "实名认证已通过"
		subStatus = "verified"
	case realname.UserRealNameInProgress:
		if message == "" {
			message = "实名认证进行中"
		}
	case realname.UserRealNameFailed:
		message = "实名认证失败"
		subStatus = "failed"
	}

	return &types.RealNameStatusResp{
		RealNameStatus: int32(user.RealNameStatus),
		CertType:       int32(user.RealNameCertType.Int16),
		SubStatus:      subStatus,
		Message:        message,
		FailReason:     failReason,
		UpdatedAt:      updatedAt,
		RealNameMasked: realNameMasked,
		IdNumberMasked: idNumberMasked,
		SubmittedAt:    submittedAt,
		AuditedAt:      auditedAt,
	}, nil
}
