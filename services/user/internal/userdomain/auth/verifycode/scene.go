package verifycode

import (
	"strings"

	"github.com/jacklau/audio-ai-platform/common/errorx"
)

// ParseSceneOptional 非空 scene 时返回规范化场景并校验枚举；空则 ok=false（不做库表场景校验，兼容旧客户端）。
func ParseSceneOptional(scene string) (normalized string, ok bool, err error) {
	s := strings.TrimSpace(scene)
	if s == "" {
		return "", false, nil
	}
	low := strings.ToLower(s)
	if err := ValidateScene(low); err != nil {
		return "", false, err
	}
	return low, true, nil
}

// ValidateScene 仅校验场景枚举（业务规则由 repo + DB 完成）。
func ValidateScene(scene string) error {
	switch scene {
	case SceneRegister, SceneReset, SceneBind, SceneLogin:
		return nil
	default:
		return errorx.NewCodeError(errorx.CodeInvalidParam, "scene 须为 register、reset、bind 或 login")
	}
}
