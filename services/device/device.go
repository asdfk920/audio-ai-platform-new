// @title           Device Service API
// @version         1.0
// @description     设备微服务 API 文档，包含设备管理、设备控制、设备状态查询等接口
// @host            localhost:8002
// @BasePath        /api/v1
// @securityDefinitions.apikey  BearerAuth
// @in              header
// @name            Authorization
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"

	apicors "github.com/jacklau/audio-ai-platform/common/cors"
	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/common/validate"
	"github.com/jacklau/audio-ai-platform/services/device/internal/commandsvc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/config"
	"github.com/jacklau/audio-ai-platform/services/device/internal/handler"
	"github.com/jacklau/audio-ai-platform/services/device/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/device/internal/redisexpire"
	"github.com/jacklau/audio-ai-platform/services/device/internal/shadowsvc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/statuspersist"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"

	httpSwagger "github.com/swaggo/http-swagger/v2"
)

var configFile = flag.String("f", "etc/device.yaml", "the config file")

func main() {
	flag.Parse()

	if err := godotenv.Load(); err != nil {
		logx.Infof(".env 文件未找到或加载失败（可选）: %v", err)
	}

	var c config.Config
	conf.MustLoad(*configFile, &c)

	loadConfigFromEnv(&c)

	httpx.SetErrorHandlerCtx(func(_ context.Context, err error) (int, any) {
		var ce *errorx.CodeError
		if errors.As(err, &ce) {
			return errorx.HTTPStatusForCode(ce.GetCode()), errorx.Error(ce.GetCode(), ce.GetMsg())
		}
		var ute *json.UnmarshalTypeError
		if errors.As(err, &ute) {
			return http.StatusBadRequest, errorx.Error(errorx.CodeInvalidParam, "请求体 JSON 字段类型不正确")
		}
		var se *json.SyntaxError
		if errors.As(err, &se) {
			return http.StatusBadRequest, errorx.Error(errorx.CodeInvalidParam, "请求体不是合法 JSON")
		}
		errMsg := strings.TrimSpace(err.Error())
		errMsgLower := strings.ToLower(errMsg)
		switch {
		case strings.Contains(errMsgLower, "no token") || strings.Contains(errMsgLower, "missing") || strings.Contains(errMsgLower, "authorize failed"):
			return http.StatusUnauthorized, errorx.Error(errorx.CodeTokenInvalid, "")
		case strings.Contains(errMsgLower, "expired"):
			return http.StatusUnauthorized, errorx.Error(errorx.CodeTokenExpired, "")
		case strings.Contains(errMsgLower, "invalid token") || strings.Contains(errMsgLower, "token is invalid"):
			return http.StatusUnauthorized, errorx.Error(errorx.CodeTokenInvalid, "")
		}
		return http.StatusInternalServerError, errorx.Error(errorx.CodeSystemError, "系统错误")
	})

	httpx.SetValidator(validate.NewHTTPValidator())
	httpx.SetOkHandler(func(_ context.Context, v any) any { return v })

	db, err := sql.Open("pgx", c.Postgres.DataSource)
	if err != nil {
		panic("postgres open: " + err.Error())
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		panic("postgres ping: " + err.Error())
	}
	logx.Infof("postgres target (host/database): %s", postgresLogTarget(c.Postgres.DataSource))

	var rdb *redis.Client
	if addr := strings.TrimSpace(c.Redis.Addr); addr != "" {
		rdb = redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: c.Redis.Password,
			DB:       c.Redis.DB,
		})
		pctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		if err := rdb.Ping(pctx).Err(); err != nil {
			cancel()
			panic("redis ping: " + err.Error())
		}
		cancel()
	}

	server := rest.MustNewServer(c.RestConf,
		rest.WithUnauthorizedCallback(func(w http.ResponseWriter, r *http.Request, err error) {
			msg := "Token 无效"
			code := errorx.CodeTokenInvalid
			if err != nil {
				if strings.Contains(err.Error(), "expired") {
					code = errorx.CodeTokenExpired
					msg = "Token 已过期，请重新登录"
				} else if strings.Contains(err.Error(), "missing") || strings.Contains(err.Error(), "no token") {
					msg = "请先登录"
				}
			}
			httpx.WriteJsonCtx(r.Context(), w, http.StatusUnauthorized, errorx.Error(code, msg))
		}),
	)
	defer server.Stop()
	if rdb != nil {
		defer func() { _ = rdb.Close() }()
	}

	server.Use(apicors.Middleware(c.CORS))

	ctx := svc.NewServiceContext(c, db, rdb)
	ctx.WsPushJSON = logic.SendCmdToDevice
	handler.RegisterHandlers(server, ctx)

	bgCtx, bgStop := context.WithCancel(context.Background())
	defer bgStop()
	if rdb != nil {
		logic.SetWsRelayRedis(rdb)
		go logic.StartWsRelaySubscriber(bgCtx)
	}
	startCommandWorker(bgCtx, ctx)

	var persist *statuspersist.Pool
	if db != nil && c.RedisKeyspace.Enabled {
		sp := c.StatusPersist
		persist = statuspersist.NewPool(db, sp.QueueSize, sp.Workers)
		persist.Start(bgCtx)
	}

	if c.RedisKeyspace.Enabled && rdb != nil && strings.TrimSpace(c.Redis.Addr) != "" {
		subRdb := redis.NewClient(&redis.Options{
			Addr:     c.Redis.Addr,
			Password: c.Redis.Password,
			DB:       c.Redis.DB,
		})
		defer func() { _ = subRdb.Close() }()
		redisexpire.StartOnlineKeyExpiryListener(bgCtx, subRdb, db, rdb, persist, c, ctx.DeviceRepo)
	}

	// 添加 Swagger UI 路由
	swaggerHandler := httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8002/swagger/doc.json"),
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("list"),
		httpSwagger.DomID("swagger-ui"),
	)
	server.AddRoute(rest.Route{
		Method: http.MethodGet,
		Path:   "/swagger/",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			swaggerHandler.ServeHTTP(w, r)
		},
	})
	server.AddRoute(rest.Route{
		Method: http.MethodGet,
		Path:   "/swagger/index.html",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			swaggerHandler.ServeHTTP(w, r)
		},
	})
	server.AddRoute(rest.Route{
		Method: http.MethodGet,
		Path:   "/swagger/doc.json",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "./docs/swagger.json")
		},
	})
	server.AddRoute(rest.Route{
		Method: http.MethodGet,
		Path:   "/swagger/:path",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			swaggerHandler.ServeHTTP(w, r)
		},
	})

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	fmt.Printf("Swagger UI: http://%s:%d/swagger/\n", c.Host, c.Port)

	if ctx.HeartbeatMonitor != nil {
		ctx.HeartbeatMonitor.Start()
		defer ctx.HeartbeatMonitor.Stop()
	}

	server.Start()
}

// postgresLogTarget logs host + database name only (no user/password) so you can confirm migrations ran on this DB.
func postgresLogTarget(dsn string) string {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return "(empty)"
	}
	u, err := url.Parse(dsn)
	if err != nil || u.Host == "" {
		return "(unparsed)"
	}
	db := strings.TrimPrefix(strings.TrimSpace(u.Path), "/")
	if i := strings.IndexByte(db, '?'); i >= 0 {
		db = db[:i]
	}
	return u.Host + "/" + db
}

func startCommandWorker(ctx context.Context, svcCtx *svc.ServiceContext) {
	if svcCtx == nil || svcCtx.DB == nil {
		return
	}
	interval := svcCtx.Config.DeviceCommand.WorkerIntervalSeconds
	if interval <= 0 {
		interval = 10
	}
	commandSvc := commandsvc.New(svcCtx)
	shadowSvc := shadowsvc.New(svcCtx)
	go func() {
		ticker := time.NewTicker(time.Duration(interval) * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := commandSvc.ExpireAndTimeoutInstructions(ctx); err != nil {
					logx.Errorf("device command worker expire/timeout: %v", err)
				}
				schedules, err := commandSvc.LoadDueSchedules(ctx, 20)
				if err != nil {
					logx.Errorf("device command worker load schedules: %v", err)
				} else {
					for _, schedule := range schedules {
						view, runErr := shadowSvc.UpdateDesiredByUserWithOptions(
							ctx, schedule.UserID, schedule.DeviceSN, schedule.DesiredPayload, schedule.MergeDesired,
							shadowsvc.DesiredCommandOptions{
								InstructionType: commandsvc.InstructionTypeScheduled,
								ScheduleID:      &schedule.ID,
								Operator:        fmt.Sprintf("schedule:%d", schedule.ID),
								Reason:          "schedule_due",
								ExpiresAt:       schedule.ExpiresAt,
							},
						)
						if runErr != nil {
							commandSvc.MarkScheduleTriggerFailed(ctx, schedule.ID, runErr.Error())
							logx.Errorf("device command worker trigger schedule=%d: %v", schedule.ID, runErr)
							continue
						}
						if err := commandSvc.MarkScheduleTriggered(ctx, schedule, view.InstructionID); err != nil {
							logx.Errorf("device command worker mark schedule=%d: %v", schedule.ID, err)
						}
					}
				}
				if err := commandSvc.RedrivePendingInstructions(ctx, 20); err != nil {
					logx.Errorf("device command worker redrive: %v", err)
				}
			}
		}
	}()
}

func loadConfigFromEnv(c *config.Config) {
	if v := os.Getenv("POSTGRES_HOST"); v != "" {
		// 检查YAML配置是否已经是线上地址
		yamlDSN := strings.TrimSpace(c.Postgres.DataSource)
		isLocalDSN := strings.Contains(yamlDSN, "localhost") ||
			strings.Contains(yamlDSN, "127.0.0.1") ||
			yamlDSN == ""

		if isLocalDSN {
			if user := os.Getenv("POSTGRES_USER"); user != "" {
				pass := os.Getenv("POSTGRES_PASS")
				db := os.Getenv("POSTGRES_DB")
				port := os.Getenv("POSTGRES_PORT")
				if port == "" {
					port = "5432"
				}
				sslmode := os.Getenv("POSTGRES_SSLMODE")
				if sslmode == "" {
					sslmode = "disable"
				}
				tz := os.Getenv("POSTGRES_TIMEZONE")
				if tz == "" {
					tz = "Asia/Shanghai"
				}
				c.Postgres.DataSource = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s&TimeZone=%s",
					user, pass, v, port, db, sslmode, tz)
				logx.Infof("从环境变量加载 PostgreSQL 配置（YAML为本地地址，已覆盖）: host=%s", v)
			}
		} else {
			displayDSN := yamlDSN
			if len(displayDSN) > 50 {
				displayDSN = displayDSN[:50] + "..."
			}
			logx.Infof("保留 YAML 配置的 PostgreSQL 地址（忽略环境变量）: %s", displayDSN)
		}
	}

	if v := os.Getenv("REDIS_ADDR"); v != "" {
		// 只有当YAML配置是默认本地地址时，才允许环境变量覆盖
		// 如果YAML明确配置为空字符串或其他非默认值，则保留YAML配置
		yamlAddr := strings.TrimSpace(c.Redis.Addr)
		if yamlAddr == "127.0.0.1:6379" || yamlAddr == "localhost:6379" {
			c.Redis.Addr = v
			c.Redis.Password = os.Getenv("REDIS_PASS")
			if dbStr := os.Getenv("REDIS_DB"); dbStr != "" {
				_, _ = fmt.Sscanf(dbStr, "%d", &c.Redis.DB)
			}
			logx.Infof("从环境变量加载 Redis 配置（YAML为本地地址，已覆盖）: addr=%s", v)
		} else if yamlAddr == "" {
			logx.Infof("YAML配置Redis为空，跳过环境变量REDIS_ADDR（禁用Redis）")
		} else {
			logx.Infof("保留 YAML 配置的 Redis 地址（忽略环境变量）: addr=%s", yamlAddr)
		}
	}

	if v := os.Getenv("AUTH_ACCESS_SECRET"); v != "" {
		c.Auth.AccessSecret = v
		logx.Infof("从环境变量加载 Auth AccessSecret（已设置）")
	}

	if v := os.Getenv("DEVICE_AUTH_TOKEN_SECRET"); v != "" {
		c.DeviceAuth.TokenSecret = v
		logx.Infof("从环境变量加载 DeviceAuth TokenSecret（已设置）")
	}

	if v := os.Getenv("HTTP_BASE_URL"); v != "" {
		c.DeviceRegister.HttpBaseUrl = v
		logx.Infof("从环境变量加载 HTTP BaseURL: %s", v)
	}

	if v := os.Getenv("LOG_LEVEL"); v != "" {
		c.Log.Level = v
	}
	if v := os.Getenv("LOG_MODE"); v != "" {
		c.Log.Mode = v
	}
}
