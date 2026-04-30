package rebindcontact

import (
	"strings"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/jacklau/audio-ai-platform/services/user/internal/userdomain/auth/verifycode"
)

// ParsedRebind 换绑旧新目标（已 Trim、已校验通道一致与格式）。
type ParsedRebind struct {
	OldTarget string
	NewTarget string
	Channel   string
	OldCode   string
	NewCode   string
}

// ParseAndValidate 入参校验（不含库表、不含 Redis 验证码比对）。
func ParseAndValidate(req *types.RebindContactReq) (*ParsedRebind, error) {
	if req == nil {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "参数错误")
	}
	oldE := strings.TrimSpace(req.OldEmail)
	oldM := strings.TrimSpace(req.OldMobile)
	newE := strings.TrimSpace(req.NewEmail)
	newM := strings.TrimSpace(req.NewMobile)

	hasOldE := oldE != ""
	hasOldM := oldM != ""
	hasNewE := newE != ""
	hasNewM := newM != ""

	var oldTarget, newTarget, channel string
	if hasOldE && !hasOldM && hasNewE && !hasNewM {
		channel = verifycode.ChannelEmail
		oldTarget, newTarget = oldE, newE
	} else if hasOldM && !hasOldE && hasNewM && !hasNewE {
		channel = verifycode.ChannelMobile
		oldTarget, newTarget = oldM, newM
	} else {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "请按同一种方式换绑（邮箱->邮箱 或 手机号->手机号）")
	}

	if err := verifycode.ValidateFormat(channel, oldTarget); err != nil {
		return nil, err
	}
	if err := verifycode.ValidateFormat(channel, newTarget); err != nil {
		return nil, err
	}
	if oldTarget == newTarget {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "新旧账号不能相同")
	}

	oldCode := strings.TrimSpace(req.OldVerifyCode)
	newCode := strings.TrimSpace(req.NewVerifyCode)
	if oldCode == "" || newCode == "" {
		return nil, errorx.NewDefaultError(errorx.CodeVerifyCodeInvalid)
	}

	return &ParsedRebind{
		OldTarget: oldTarget,
		NewTarget: newTarget,
		Channel:   channel,
		OldCode:   oldCode,
		NewCode:   newCode,
	}, nil
}
