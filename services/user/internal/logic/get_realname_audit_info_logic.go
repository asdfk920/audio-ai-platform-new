package logic

import (
	"context"
	"database/sql"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/pkg/util/ctxuser"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/jacklau/audio-ai-platform/services/user/internal/userdomain/profile/realname"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetRealNameAuditInfoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetRealNameAuditInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetRealNameAuditInfoLogic {
	return &GetRealNameAuditInfoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetRealNameAuditInfoLogic) Get() (resp *types.RealNameAuditInfoResp, err error) {
	userID := ctxuser.ParseUserID(l.ctx)
	if userID == 0 {
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "未登录")
	}

	auth, err := l.svcCtx.UserRepo.RealNameFindLatestAuthByUser(l.ctx, userID)
	if err != nil {
		logx.Errorf("find latest auth failed: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "系统繁忙")
	}
	if auth == nil {
		return nil, errorx.NewCodeError(errorx.CodeRealNameRecordNotFound, "暂无实名认证记录")
	}

	user, err := l.svcCtx.UserRepo.RealNameTxLoadUserForUpdate(l.ctx, nil, userID)
	if err != nil {
		logx.Errorf("load user failed: %v", err)
	}

	var realNameMasked string
	if user != nil && user.RealNameStatus == realname.UserRealNameVerified {
		if user.Nickname != nil {
			realNameMasked = realname.MaskRealName(*user.Nickname)
		} else {
			realNameMasked = auth.RealNameMasked
		}
	} else {
		realNameMasked = auth.RealNameMasked
	}

	var reviewedAt int64
	if auth.ReviewedAt.Valid {
		reviewedAt = auth.ReviewedAt.Time.Unix()
	}

	var failReason string
	if auth.FailReason.Valid {
		failReason = auth.FailReason.String
	}

	var reviewerNote string
	if auth.ReviewerNote.Valid {
		reviewerNote = auth.ReviewerNote.String
	}

	var thirdPartyFlowNo string
	if auth.ThirdPartyFlowNo.Valid {
		thirdPartyFlowNo = auth.ThirdPartyFlowNo.String
	}

	return &types.RealNameAuditInfoResp{
		AuthRecordId:     auth.ID,
		AuthStatus:       int32(auth.AuthStatus),
		RealNameMasked:   realNameMasked,
		IdNumberLast4:    auth.IdNumberLast4,
		CertType:         int32(auth.CertType),
		IdCardFrontRef:   l.refToURL(auth.IdCardFrontRef),
		IdCardBackRef:    l.refToURL(auth.IdCardBackRef),
		FaceDataRef:      l.refToURL(auth.FaceDataRef),
		FailReason:       failReason,
		ReviewerNote:     reviewerNote,
		ThirdPartyFlowNo: thirdPartyFlowNo,
		CreatedAt:        auth.CreatedAt.Unix(),
		ReviewedAt:       reviewedAt,
	}, nil
}

func (l *GetRealNameAuditInfoLogic) refToURL(ref sql.NullString) string {
	if !ref.Valid || ref.String == "" {
		return ""
	}
	if len(ref.String) > 10 && ref.String[:10] == "oss://real" {
		return "https://cdn.example.com/" + ref.String[15:]
	}
	return ref.String
}
