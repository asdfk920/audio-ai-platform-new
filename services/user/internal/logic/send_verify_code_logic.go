package logic

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/pkg/redisx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	// redisKeyVerifyCode 存放验证码本体（Key 按接收方区分：邮箱/手机号必须和验证场景一致）
	// 设计原因：验证码校验只依赖“目标账号 + code”，避免额外状态；同时天然支持邮箱/手机号两条通道。
	redisKeyVerifyCode = "user:verify_code:%s"
	// redisKeyVerifySendCnt 发送频控计数（1分钟窗口内最多 N 次）
	// 设计原因：防止短信/邮件轰炸与资源滥用，且实现简单（incr + expire）。
	redisKeyVerifySendCnt = "user:verify_send_cnt:%s"
	// redisKeyVerifyBlock 超限后短时间封禁
	// 设计原因：对“超过频率”的请求给出明确冷却时间，避免一直刷验证码。
	redisKeyVerifyBlock    = "user:verify_block:%s"
	verifyCodeExpire       = 3 * time.Minute
	verifySendWindow       = time.Minute
	verifyBlockDuration    = 3 * time.Minute
	verifyCodeMaxPerMinute = 3
)

type SendVerifyCodeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 发送验证码（邮箱或手机二选一，1分钟有效，1分钟最多3条，超过提示3分钟后重试）
func NewSendVerifyCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendVerifyCodeLogic {
	return &SendVerifyCodeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SendVerifyCodeLogic) SendVerifyCode(req *types.SendVerifyCodeReq) (resp *types.SendVerifyCodeResp, err error) {
	target, by := l.normalizeTarget(req)
	if target == "" {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "请填写邮箱或手机号其中一项")
	}

	// 频控：先检查是否处于封禁期
	blockKey := fmt.Sprintf(redisKeyVerifyBlock, target)
	if n, _ := redisx.Exists(l.ctx, blockKey); n > 0 {
		return nil, errorx.NewDefaultError(errorx.CodeVerifyCodeLimit)
	}

	// 频控：窗口计数 + 首次设置过期时间
	cntKey := fmt.Sprintf(redisKeyVerifySendCnt, target)
	n, err := redisx.Incr(l.ctx, cntKey)
	if err != nil {
		l.Logger.Errorf("redis incr %s: %v", cntKey, err)
		return nil, errorx.NewCodeError(errorx.CodeRedisError, err.Error())
	}
	if n == 1 {
		_ = redisx.Expire(l.ctx, cntKey, verifySendWindow)
	}
	if n > int64(verifyCodeMaxPerMinute) {
		// 超限：写入封禁 key，避免持续刷接口
		_ = redisx.Set(l.ctx, blockKey, "1", verifyBlockDuration)
		return nil, errorx.NewDefaultError(errorx.CodeVerifyCodeLimit)
	}

	// 生成并写入验证码（有效期较短，降低被撞库/泄露后的风险）
	code := l.genCode()
	codeKey := fmt.Sprintf(redisKeyVerifyCode, target)
	if err := redisx.Set(l.ctx, codeKey, code, verifyCodeExpire); err != nil {
		l.Logger.Errorf("redis set %s: %v", codeKey, err)
		return nil, errorx.NewCodeError(errorx.CodeRedisError, err.Error())
	}

	// TODO: 实际发送邮件/短信；开发阶段可打日志
	if by == "email" {
		l.Logger.Infof("[dev] verify code for %s: %s (expire 3min)", target, code)
	} else {
		l.Logger.Infof("[dev] verify code for mobile %s: %s (expire 3min)", target, code)
	}

	expireSec := 180 // 默认 3 分钟
	if l.svcCtx.Config.VerifyCode.ExpireSeconds > 0 {
		expireSec = l.svcCtx.Config.VerifyCode.ExpireSeconds
	}
	return &types.SendVerifyCodeResp{ExpireSeconds: expireSec}, nil
}

func (l *SendVerifyCodeLogic) normalizeTarget(req *types.SendVerifyCodeReq) (target, by string) {
	hasEmail := req.Email != ""
	hasMobile := req.Mobile != ""
	if hasEmail && !hasMobile {
		return req.Email, "email"
	}
	if hasMobile && !hasEmail {
		return req.Mobile, "mobile"
	}
	return "", ""
}

func (l *SendVerifyCodeLogic) genCode() string {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	return fmt.Sprintf("%06d", rnd.Intn(1000000))
}
