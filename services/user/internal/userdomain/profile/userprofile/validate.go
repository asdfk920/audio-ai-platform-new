package userprofile

import (
	"strings"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/common/validate"
	regutil "github.com/jacklau/audio-ai-platform/services/user/internal/userdomain/auth/register"
)

// containsEmojiRune 拦截常见 Emoji / 绘文字区间（昵称禁止）。
func containsEmojiRune(s string) bool {
	for _, r := range s {
		switch {
		case r == '\uFE0F' || r == '\u200D':
			return true
		case r >= 0x1F000 && r <= 0x1FAFF:
			return true
		case r >= 0x2600 && r <= 0x27BF:
			return true
		case r >= 0x231A && r <= 0x231B:
			return true
		}
	}
	return false
}

// ValidateUpdateNickname 非空昵称：长度/敏感词等与注册一致，并禁止 Emoji。
func ValidateUpdateNickname(nickname string) error {
	if nickname == "" {
		return nil
	}
	if containsEmojiRune(nickname) {
		return errorx.NewCodeError(errorx.CodeInvalidNickname, "昵称不能包含表情符号")
	}
	return regutil.ValidateNickname(nickname)
}

// ValidateUpdateAvatar 非空头像：长度、危险 scheme 与 URL 格式。
func ValidateUpdateAvatar(avatar string) error {
	if avatar == "" {
		return nil
	}
	if len(avatar) > 500 {
		return errorx.NewCodeError(errorx.CodeInvalidProfileAvatar, "头像链接过长")
	}
	low := strings.ToLower(strings.TrimSpace(avatar))
	if strings.HasPrefix(low, "javascript:") || strings.HasPrefix(low, "data:") || strings.HasPrefix(low, "vbscript:") {
		return errorx.NewDefaultError(errorx.CodeInvalidProfileAvatar)
	}
	if err := validate.CheckURL(avatar); err != nil {
		return errorx.NewDefaultError(errorx.CodeInvalidProfileAvatar)
	}
	return nil
}
