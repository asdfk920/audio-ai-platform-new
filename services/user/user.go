package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/common/validate"
	"github.com/jacklau/audio-ai-platform/pkg/redisx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/config"
	"github.com/jacklau/audio-ai-platform/services/user/internal/handler"
	"github.com/jacklau/audio-ai-platform/services/user/internal/pkg/logger"
	"github.com/jacklau/audio-ai-platform/services/user/internal/scheduler"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
)

var configFile = flag.String("f", "etc/user.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	// 统一错误响应为 JSON：{ code, msg }（与 errorx.Response 信封一致，失败无 data）
	httpx.SetErrorHandlerCtx(func(_ context.Context, err error) (int, any) {
		// 须用 errors.As：go-zero/校验链路上可能对 *errorx.CodeError 做 %w 包装，直接类型断言会落到 9004。
		var ce *errorx.CodeError
		if errors.As(err, &ce) {
			return errorx.HTTPStatusForCode(ce.GetCode()), errorx.Error(ce.GetCode(), ce.GetMsg())
		}

		// JSON 解析错误：返回参数错误，不要兜底成 9004
		var ute *json.UnmarshalTypeError
		if errors.As(err, &ute) {
			return http.StatusBadRequest, errorx.Error(errorx.CodeInvalidParam, "请求体 JSON 字段类型不正确")
		}
		var se *json.SyntaxError
		if errors.As(err, &se) {
			return http.StatusBadRequest, errorx.Error(errorx.CodeInvalidParam, "请求体不是合法 JSON")
		}

		// #region agent log
		// 兜底：go-zero 在鉴权失败时可能返回普通 error（*errors.errorString），未实现 CodeError。
		// 这里按错误文案映射到 token 语义，避免被统一兜底成 9004 系统错误。
		errMsg := strings.TrimSpace(err.Error())
		errMsgLower := strings.ToLower(errMsg)
		switch {
		case strings.Contains(errMsgLower, "no token") || strings.Contains(errMsgLower, "missing") || strings.Contains(errMsgLower, "authorize failed"):
			// #region agent log
			logger.AgentNDJSON("H26", "user.go:SetErrorHandlerCtx", "mapped auth error->CodeTokenInvalid", map[string]any{
				"errType": logger.ErrType(err),
				"errMsg": func() string {
					if len(errMsg) > 220 {
						return errMsg[:220]
					} else {
						return errMsg
					}
				}(),
				"mappedCode": errorx.CodeTokenInvalid,
			})
			// #endregion
			return http.StatusUnauthorized, errorx.Error(errorx.CodeTokenInvalid, "")
		case strings.Contains(errMsgLower, "expired"):
			// #region agent log
			logger.AgentNDJSON("H26", "user.go:SetErrorHandlerCtx", "mapped auth error->CodeTokenExpired", map[string]any{
				"errType": logger.ErrType(err),
				"errMsg": func() string {
					if len(errMsg) > 220 {
						return errMsg[:220]
					} else {
						return errMsg
					}
				}(),
				"mappedCode": errorx.CodeTokenExpired,
			})
			// #endregion
			return http.StatusUnauthorized, errorx.Error(errorx.CodeTokenExpired, "")
		case strings.Contains(errMsgLower, "invalid token") || strings.Contains(errMsgLower, "token is invalid"):
			// #region agent log
			logger.AgentNDJSON("H26", "user.go:SetErrorHandlerCtx", "mapped auth error->CodeTokenInvalid", map[string]any{
				"errType": logger.ErrType(err),
				"errMsg": func() string {
					if len(errMsg) > 220 {
						return errMsg[:220]
					} else {
						return errMsg
					}
				}(),
				"mappedCode": errorx.CodeTokenInvalid,
			})
			// #endregion
			return http.StatusUnauthorized, errorx.Error(errorx.CodeTokenInvalid, "")
		}
		// #endregion

		// #region agent log
		logger.AgentNDJSON("H1", "user.go:SetErrorHandlerCtx", "fallback system error (non-CodeError)", map[string]any{
			"errType": logger.ErrType(err),
			"errMsg": func() string {
				if len(errMsg) > 220 {
					return errMsg[:220]
				} else {
					return errMsg
				}
			}(),
		})
		// #endregion
		return http.StatusInternalServerError, errorx.Error(errorx.CodeSystemError, "系统错误")
	})

	// httpx.Parse 完成后按 struct validate 标签校验（go-playground/validator）
	httpx.SetValidator(validate.NewHTTPValidator())

	// OkJsonCtx 在调用链上会先走 okHandler；显式设为透传，避免被其它 init 改成「只输出业务体」。
	httpx.SetOkHandler(func(_ context.Context, v any) any { return v })

	if err := redisx.Init(redisx.Config{
		Addr:     c.Redis.Addr,
		Password: c.Redis.Password,
		DB:       c.Redis.DB,
	}); err != nil {
		panic("redis init: " + err.Error())
	}
	defer redisx.Close()

	db, err := sql.Open("pgx", c.Postgres.DataSource)
	if err != nil {
		panic("postgres open: " + err.Error())
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		panic("postgres ping: " + err.Error())
	}

	server := rest.MustNewServer(c.RestConf,
		rest.WithUnauthorizedCallback(func(w http.ResponseWriter, r *http.Request, err error) {
			// #region agent log
			authHdr := r.Header.Get("Authorization")
			errSnippet := ""
			if err != nil {
				errSnippet = err.Error()
				if len(errSnippet) > 220 {
					errSnippet = errSnippet[:220]
				}
			}
			trimAuth := strings.TrimSpace(authHdr)
			rawTok := strings.TrimSpace(strings.TrimPrefix(trimAuth, "Bearer "))
			jwtDotCount := strings.Count(rawTok, ".")
			logger.AgentNDJSON("H1", "user.go:UnauthorizedCallback", "jwt unauthorized", map[string]any{
				"runId":           "pre-fix",
				"path":            r.URL.Path,
				"authHeaderLen":   len(authHdr),
				"hasBearerPrefix": strings.HasPrefix(trimAuth, "Bearer "),
				"jwtDotCount":     jwtDotCount,
				"looksLikeJWT":    jwtDotCount == 2,
				"errType":         logger.ErrType(err),
				"errSnippet":      errSnippet,
				"errHasExpired":   err != nil && strings.Contains(strings.ToLower(err.Error()), "expired"),
				"errHasSignature": err != nil && strings.Contains(strings.ToLower(err.Error()), "signature"),
				"errHasMalformed": err != nil && strings.Contains(strings.ToLower(err.Error()), "malformed"),
				"serverNowUnix":   time.Now().Unix(),
			})
			// #endregion
			msg := "Token 无效"
			code := errorx.CodeTokenInvalid
			if err != nil {
				// go-zero 在服务端日志里输出 err 详情，这里给前端更友好的提示
				if strings.Contains(err.Error(), "expired") {
					code = errorx.CodeTokenExpired
					msg = "Token 已过期，请重新登录"
				} else if strings.Contains(err.Error(), "missing") || strings.Contains(err.Error(), "no token") {
					msg = "请先登录"
				}
			}
			httpx.WriteJsonCtx(r.Context(), w, http.StatusUnauthorized, errorx.Error(code, msg))
		}),
		rest.WithNotFoundHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// #region agent log
			// debug-mode log: capture 404 requests (no secrets / no PII)
			func() {
				type payload struct {
					SessionId     string         `json:"sessionId"`
					RunId         string         `json:"runId"`
					HypothesisId  string         `json:"hypothesisId"`
					Location      string         `json:"location"`
					Message       string         `json:"message"`
					Data          map[string]any `json:"data,omitempty"`
					TimestampUnix int64          `json:"timestamp"`
				}
				p := payload{
					SessionId:    "046601",
					RunId:        "pre-fix",
					HypothesisId: "H3_notfound",
					Location:     "user.go:WithNotFoundHandler",
					Message:      "request hit not-found handler",
					Data: map[string]any{
						"method": r.Method,
						"path":   r.URL.Path,
					},
					TimestampUnix: time.Now().UnixMilli(),
				}
				b, _ := json.Marshal(p)
				f, ferr := os.OpenFile("C:\\Users\\Lenovo\\Desktop\\audio-ai-platform\\debug-046601.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
				if ferr == nil {
					_, _ = f.Write(append(b, '\n'))
					_ = f.Close()
				}
			}()
			// #endregion

			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("404 page not found"))
		})),
	)
	defer func() { server.Stop() }()

	ctx := svc.NewServiceContext(c, db)
	handler.RegisterHandlers(server, ctx)

	// 启动账号注销定时任务
	sched := scheduler.NewCancellationScheduler(context.Background(), ctx, c.Cancellation)
	if err := sched.Start(); err != nil {
		panic("启动注销定时任务失败：" + err.Error())
	}
	defer func() { _ = sched.Stop() }()

	deviceShareSched := scheduler.NewDeviceShareScheduler(context.Background(), ctx, c.DeviceShare)
	if err := deviceShareSched.Start(); err != nil {
		panic("启动设备共享过期任务失败：" + err.Error())
	}
	defer func() { _ = deviceShareSched.Stop() }()

	arSched := scheduler.NewMemberAutoRenewScheduler(context.Background(), ctx, c.MemberAutoRenew)
	if err := arSched.Start(); err != nil {
		panic("启动会员自动续费扫描任务失败：" + err.Error())
	}
	defer func() { _ = arSched.Stop() }()

	drSched := scheduler.NewDownloadRecordCleanScheduler(context.Background(), ctx, c.DownloadRecordClean)
	if err := drSched.Start(); err != nil {
		panic("启动下载记录清理任务失败：" + err.Error())
	}
	defer func() { _ = drSched.Stop() }()

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
