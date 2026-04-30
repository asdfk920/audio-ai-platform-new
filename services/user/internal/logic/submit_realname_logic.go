package logic

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/pkg/util/ctxuser"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/jacklau/audio-ai-platform/services/user/internal/userdomain/profile/realname"
	"github.com/zeromicro/go-zero/core/logx"
)

type SubmitRealNameLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSubmitRealNameLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SubmitRealNameLogic {
	return &SubmitRealNameLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SubmitRealNameLogic) Submit(req *types.RealNameSubmitReq, clientIP, userAgent string) (resp *types.RealNameSubmitResp, err error) {
	if req.CertType != realname.CertTypePersonal && req.CertType != realname.CertTypeEnterprise {
		return nil, errorx.NewCodeError(errorx.CodeRealNameInvalidPayload, "证件类型不支持")
	}

	if err := realname.ValidateSubmit(
		req.CertType,
		req.RealName,
		req.IdNumber,
		req.IdPhotoBase64,
		req.IdCardFrontBase64,
		req.IdCardBackBase64,
		req.FaceImageBase64,
		false,
	); err != nil {
		return nil, err
	}

	userID := ctxuser.ParseUserID(l.ctx)
	if userID == 0 {
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "未登录")
	}

	user, err := l.svcCtx.UserRepo.RealNameTxLoadUserForUpdate(l.ctx, nil, userID)
	if err != nil {
		logx.Errorf("load user for update failed: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "系统繁忙")
	}
	if user == nil {
		return nil, errorx.NewCodeError(errorx.CodeUserNotFound, "用户不存在")
	}

	if user.RealNameStatus == realname.UserRealNameVerified {
		return &types.RealNameSubmitResp{
			RealNameStatus: int32(realname.UserRealNameVerified),
			Message:        "您已完成实名认证，无需重复提交",
		}, nil
	}

	if user.RealNameStatus == realname.UserRealNameInProgress {
		latestAuth, err := l.svcCtx.UserRepo.RealNameFindLatestAuthByUser(l.ctx, userID)
		if err != nil {
			logx.Errorf("find latest auth failed: %v", err)
		} else if latestAuth != nil && latestAuth.AuthStatus == realname.AuthPendingThirdParty {
			return &types.RealNameSubmitResp{
				RealNameStatus: int32(realname.UserRealNameInProgress),
				Message:        "您的实名申请正在审核中，请勿重复提交",
			}, nil
		}
	}

	tx, err := l.svcCtx.UserRepo.BeginTx(l.ctx)
	if err != nil {
		logx.Errorf("begin tx failed: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "系统繁忙")
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	user, err = l.svcCtx.UserRepo.RealNameTxLoadUserForUpdate(l.ctx, tx, userID)
	if err != nil {
		logx.Errorf("load user in tx failed: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "系统繁忙")
	}
	if user == nil {
		return nil, errorx.NewCodeError(errorx.CodeUserNotFound, "用户不存在")
	}

	if user.RealNameStatus == realname.UserRealNameVerified {
		_ = tx.Rollback()
		return &types.RealNameSubmitResp{
			RealNameStatus: int32(realname.UserRealNameVerified),
			Message:        "您已完成实名认证，无需重复提交",
		}, nil
	}

	aesKey, err := l.svcCtx.Config.ResolveIDNumberKey()
	if err != nil {
		logx.Errorf("resolve AES key failed: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeSystemError, "系统繁忙")
	}

	idNumberEncrypted, err := realname.EncryptAESGCM(
		strings.ToUpper(strings.TrimSpace(req.IdNumber)),
		aesKey,
	)
	if err != nil {
		logx.Errorf("encrypt id number failed: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "系统繁忙")
	}

	realNameMasked := realname.MaskRealName(strings.TrimSpace(req.RealName))
	idNumberLast4 := realname.IDLast4(strings.ToUpper(req.IdNumber))

	frontB64 := strings.TrimSpace(req.IdCardFrontBase64)
	if frontB64 == "" {
		frontB64 = strings.TrimSpace(req.IdPhotoBase64)
	}
	backB64 := strings.TrimSpace(req.IdCardBackBase64)
	faceB64 := strings.TrimSpace(req.FaceImageBase64)

	var frontRef, backRef, faceRef *string
	if frontB64 != "" {
		ref := l.saveImageRef(frontB64)
		frontRef = &ref
	}
	if backB64 != "" {
		ref := l.saveImageRef(backB64)
		backRef = &ref
	}
	if faceB64 != "" {
		ref := l.saveImageRef(faceB64)
		faceRef = &ref
	}

	authID, err := l.svcCtx.UserRepo.RealNameTxInsertAuth(
		l.ctx,
		tx,
		userID,
		req.CertType,
		realNameMasked,
		idNumberEncrypted,
		idNumberLast4,
		frontRef,
		frontRef,
		backRef,
		faceRef,
		userAgent,
	)
	if err != nil {
		logx.Errorf("insert auth failed: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "系统繁忙")
	}

	err = l.svcCtx.UserRepo.RealNameTxMarkUserInProgress(l.ctx, tx, userID, req.CertType)
	if err != nil {
		logx.Errorf("mark user in progress failed: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "系统繁忙")
	}

	err = tx.Commit()
	if err != nil {
		logx.Errorf("commit tx failed: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "系统繁忙")
	}

	go l.asyncThirdPartyVerify(authID, userID, req)

	return &types.RealNameSubmitResp{
		RealNameStatus: int32(realname.UserRealNameInProgress),
		Message:        "实名信息已提交，等待审核",
		AuthRecordId:   authID,
	}, nil
}

func (l *SubmitRealNameLogic) saveImageRef(b64 string) string {
	if len(b64) > 10<<20 {
		return ""
	}
	hash := sha256.Sum256([]byte(b64))
	return "oss://realname/" + hex.EncodeToString(hash[:16])
}

func (l *SubmitRealNameLogic) asyncThirdPartyVerify(authID int64, userID int64, req *types.RealNameSubmitReq) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	flowNo := "MOCK" + time.Now().Format("20060102150405")
	channel := "mock"
	rawResp := `{"code":"0","message":"success"}`

	var authStatus int16 = realname.AuthThirdPartyPass
	var failReason *string

	_ = l.svcCtx.UserRepo.RealNameUpdateAuthAfterThirdParty(ctx, authID, authStatus, flowNo, channel, rawResp, failReason)
	if authStatus == realname.AuthThirdPartyPass {
		_ = l.svcCtx.UserRepo.RealNameMarkPendingManual(ctx, authID, flowNo, channel, rawResp)
	}
}
