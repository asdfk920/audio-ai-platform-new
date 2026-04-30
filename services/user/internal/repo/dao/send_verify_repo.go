package dao

import (
	"context"
	"database/sql"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/userdomain/auth/verifycode"
)

// SendVerifyRepo 发验证码前的库表规则校验（按场景）。
type SendVerifyRepo struct {
	db *sql.DB
}

func NewSendVerifyRepo(db *sql.DB) *SendVerifyRepo {
	return &SendVerifyRepo{db: db}
}

// ValidateScene register/bind：账号未被占用；reset：账号必须已存在。
func (r *SendVerifyRepo) ValidateScene(ctx context.Context, scene, channel, target string) error {
	if r == nil || r.db == nil {
		return errorx.NewDefaultError(errorx.CodeSystemError)
	}
	if err := ctx.Err(); err != nil {
		return errorx.NewDefaultError(errorx.CodeSystemError)
	}
	if err := verifycode.ValidateChannel(channel); err != nil {
		return err
	}
	if err := verifycode.ValidateScene(scene); err != nil {
		return err
	}

	var registered bool
	var err error
	switch channel {
	case verifycode.ChannelEmail:
		registered, err = EmailRegistered(ctx, r.db, target)
	case verifycode.ChannelMobile:
		registered, err = MobileRegistered(ctx, r.db, target)
	default:
		return errorx.NewCodeError(errorx.CodeInvalidParam, "无效的联系方式类型")
	}
	if err != nil {
		// 不向客户端暴露 SQL/驱动细节
		return errorx.NewDefaultError(errorx.CodeDatabaseError)
	}

	switch scene {
	case verifycode.SceneRegister, verifycode.SceneBind:
		if registered {
			return errorx.NewDefaultError(errorx.CodeUserExists)
		}
	case verifycode.SceneReset:
		if !registered {
			return errorx.NewDefaultError(errorx.CodeUserNotFound)
		}
	case verifycode.SceneLogin:
		// 已注册 / 未注册均可发码（验证码登录 + 静默注册）
	default:
		return errorx.NewCodeError(errorx.CodeInvalidParam, "无效的场景")
	}
	return nil
}
