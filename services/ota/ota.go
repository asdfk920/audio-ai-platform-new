package main

import (
	"database/sql"
	"flag"
	"fmt"
	"net/url"
	"strings"

	apicors "github.com/jacklau/audio-ai-platform/common/cors"
	"github.com/jacklau/audio-ai-platform/services/ota/internal/config"
	"github.com/jacklau/audio-ai-platform/services/ota/internal/handler"
	"github.com/jacklau/audio-ai-platform/services/ota/internal/svc"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/ota.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	db, err := sql.Open("pgx", c.Postgres.DataSource)
	if err != nil {
		panic("postgres open: " + err.Error())
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		panic("postgres ping: " + err.Error())
	}
	logx.Infof("postgres target: %s", postgresLogTarget(c.Postgres.DataSource))

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	server.Use(apicors.Middleware(c.CORS))

	svcCtx := svc.NewServiceContext(c, db)

	handler.RegisterHandlers(server, svcCtx)

	fmt.Printf("Starting OTA service at %s:%d...\n", c.Host, c.Port)
	fmt.Println("=========================================")
	fmt.Println("🚀 OTA Service 启动中...")
	fmt.Println("=========================================")
	fmt.Printf("📡 服务地址: http://%s:%d\n", c.Host, c.Port)
	fmt.Printf("🔍 健康检查: http://%s:%d/health\n", c.Host, c.Port)
	fmt.Printf("📦 批量检测: POST http://%s:%d/api/v1/ota/batch-check\n", c.Host, c.Port)
	fmt.Println("=========================================")
	fmt.Println("🎉 服务启动完成，等待请求...")
	fmt.Println("=========================================")

	server.Start()
}

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
