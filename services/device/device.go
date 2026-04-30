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
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/common/validate"
	"github.com/jacklau/audio-ai-platform/pkg/mqttx"
	"github.com/jacklau/audio-ai-platform/services/device/internal/commandsvc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/config"
	"github.com/jacklau/audio-ai-platform/services/device/internal/handler"
	"github.com/jacklau/audio-ai-platform/services/device/internal/mqttingest"
	"github.com/jacklau/audio-ai-platform/services/device/internal/redisexpire"
	"github.com/jacklau/audio-ai-platform/services/device/internal/shadowsvc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/statuspersist"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
)

var configFile = flag.String("f", "etc/device.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

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

	ctx := svc.NewServiceContext(c, db, rdb)
	handler.RegisterHandlers(server, ctx)

	bgCtx, bgStop := context.WithCancel(context.Background())
	defer bgStop()
	startCommandWorker(bgCtx, ctx)

	var persist *statuspersist.Pool
	if db != nil && (c.MqttIngest.Enabled || c.RedisKeyspace.Enabled) {
		sp := c.StatusPersist
		persist = statuspersist.NewPool(db, sp.QueueSize, sp.Workers)
		persist.Start(bgCtx)
	}

	var mqttClient *mqttx.Client
	defer func() {
		if mqttClient != nil {
			mqttClient.Disconnect()
		}
	}()

	if c.MqttIngest.Enabled {
		if rdb == nil || db == nil {
			logx.Error("MqttIngest enabled but Redis or DB nil; skip MQTT consumer")
		} else {
			broker := strings.TrimSpace(c.MqttIngest.Broker)
			if broker == "" {
				broker = strings.TrimSpace(c.DeviceRegister.MqttBroker)
			}
			if broker == "" {
				logx.Error("MqttIngest enabled but Broker empty (set MqttIngest.Broker or DeviceRegister.MqttBroker)")
			} else {
				cid := strings.TrimSpace(c.MqttIngest.ClientID)
				if cid == "" {
					cid = "device-api-mqtt"
				}
				mc, err := mqttx.NewClient(mqttx.Config{
					Broker:   broker,
					ClientID: cid,
					Username: c.MqttIngest.Username,
					Password: c.MqttIngest.Password,
				})
				if err != nil {
					logx.Errorf("mqtt consumer connect: %v", err)
				} else if mc != nil {
					mqttClient = mc
					ctx.SetMQTTClient(mc)
					topic := strings.TrimSpace(c.MqttIngest.SubscribeTopic)
					if topic == "" {
						topic = "device/+/report"
					}
					qos := c.MqttIngest.QOS
					if qos < 0 || qos > 2 {
						qos = 1
					}
					h := mqttingest.NewHandler(db, rdb, c, persist, mc)
					if err := mc.Subscribe(topic, byte(qos), h.OnMessage); err != nil {
						logx.Errorf("mqtt subscribe %s: %v", topic, err)
					} else {
						logx.Infof("mqtt consumer subscribed topic=%s qos=%d", topic, qos)
					}

					// 订阅 MQTT 连接/断开事件（EMQX 系统主题）
					connHandler := mqttingest.NewConnectionEventHandler(db, rdb, c)
					eventTopics := []struct {
						topic   string
						handler mqtt.MessageHandler
					}{
						{"$SYS/brokers/+/clients/+/connected", connHandler.OnConnect},
						{"$SYS/brokers/+/clients/+/disconnected", connHandler.OnDisconnect},
					}
					for _, et := range eventTopics {
						if err := mc.Subscribe(et.topic, 0, et.handler); err != nil {
							logx.Errorf("mqtt subscribe event %s: %v", et.topic, err)
						} else {
							logx.Infof("mqtt event subscribed topic=%s", et.topic)
						}
					}
				}
			}
		}
	}

	if c.RedisKeyspace.Enabled && rdb != nil && strings.TrimSpace(c.Redis.Addr) != "" {
		subRdb := redis.NewClient(&redis.Options{
			Addr:     c.Redis.Addr,
			Password: c.Redis.Password,
			DB:       c.Redis.DB,
		})
		defer func() { _ = subRdb.Close() }()
		redisexpire.StartOnlineKeyExpiryListener(bgCtx, subRdb, db, rdb, persist, c)
	}

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
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
