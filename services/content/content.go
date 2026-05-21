// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2
// @title           内容微服务API文档
// @version         1.0
// @description     音频AI平台内容微服务接口文档，包含歌曲管理、播放记录、搜索、订阅、歌单、通知等功能

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8003
// @BasePath  /api/v1/content

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description 请输入Bearer Token

package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/jacklau/audio-ai-platform/services/content/docs"

	apicors "github.com/jacklau/audio-ai-platform/common/cors"
	"github.com/jacklau/audio-ai-platform/services/content/internal/config"
	"github.com/jacklau/audio-ai-platform/services/content/internal/handler"
	"github.com/jacklau/audio-ai-platform/services/content/internal/pkg/uploadsysconfig"
	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/joho/godotenv"

	httpSwagger "github.com/swaggo/http-swagger/v2"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/content.yaml", "the config file")

func main() {
	flag.Parse()

	// 可选加载本地密钥：services/content/.env 或配置文件同目录下的 .env（示例见 .env.example）
	_ = godotenv.Load()
	if cfgDir := filepath.Dir(filepath.Clean(*configFile)); cfgDir != "." && cfgDir != "" {
		_ = godotenv.Load(filepath.Join(cfgDir, ".env"))
	}

	var c config.Config
	conf.MustLoad(*configFile, &c)
	if strings.TrimSpace(c.Database.DataSource) != "" && c.Storage.SyncFromSysConfig {
		if err := uploadsysconfig.MergeFromPostgreSQL(context.Background(), c.Database.DataSource, &c); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "[content-service] FATAL: load upload storage from sys_config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("[content-service] storage fields merged from sys_config (env still overrides afterward)\n")
	}
	applyStorageEnvOverrides(&c)
	maybePromoteLocalToOSS(&c)
	fmt.Printf("[content-service] effective storage: driver=%s endpoint=%s bucket=%s\n",
		strings.TrimSpace(c.Storage.Driver),
		strings.TrimSpace(c.Storage.Endpoint),
		strings.TrimSpace(c.Storage.Bucket),
	)

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	server.Use(apicors.Middleware(c.CORS))

	ctx, err := svc.NewServiceContext(c)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "[content-service] FATAL: ServiceContext init failed: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if ctx.DB != nil {
			sqlDB, _ := ctx.DB.DB()
			if sqlDB != nil {
				_ = sqlDB.Close()
			}
		}
	}()

	swaggerHandler := httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8003/swagger/doc.json"),
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("list"),
		httpSwagger.DomID("swagger-ui"),
	)

	server.AddRoutes([]rest.Route{
		{
			Method: http.MethodGet,
			Path:   "/swagger/",
			Handler: func(w http.ResponseWriter, r *http.Request) {
				swaggerHandler.ServeHTTP(w, r)
			},
		},
		{
			Method: http.MethodGet,
			Path:   "/swagger/index.html",
			Handler: func(w http.ResponseWriter, r *http.Request) {
				swaggerHandler.ServeHTTP(w, r)
			},
		},
		{
			Method: http.MethodGet,
			Path:   "/swagger/doc.json",
			Handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				http.ServeFile(w, r, "./docs/swagger.json")
			},
		},
		{
			Method: http.MethodGet,
			Path:   "/swagger/:path",
			Handler: func(w http.ResponseWriter, r *http.Request) {
				swaggerHandler.ServeHTTP(w, r)
			},
		},
	})

	handler.RegisterHandlers(server, ctx)

	root := strings.TrimSpace(c.Local.Root)
	if root == "" {
		root = "./data/content-objects"
	}
	if strings.EqualFold(strings.TrimSpace(c.Storage.Driver), "local") {
		_, _ = fmt.Fprintf(os.Stderr, `[content-service] WARNING: Storage.Driver is "local" — private-format uploads go to LOCAL disk only (NOT Alibaba Cloud OSS).

To use OSS either:
  1) YAML: Storage.Driver=oss (+ endpoint/bucket; RAM 密钥建议仅用环境变量或 .env)。
  2) 配置中心：在 admin 后台维护 sys_config 的 upload_storage_*，并把 Storage.SyncFromSysConfig=true。
  3) 环境变量：CONTENT_STORAGE_DRIVER=oss 以及 CONTENT_OSS_* / OSS_*。

优先级（每项独立）：环境变量 > sys_config > YAML。

Current driver=%q bucket=%q endpoint=%q

`, strings.TrimSpace(c.Storage.Driver), strings.TrimSpace(c.Storage.Bucket), strings.TrimSpace(c.Storage.Endpoint))
		_ = os.MkdirAll(root, 0o755)
		fs := http.FileServer(http.Dir(root))
		server.AddRoute(rest.Route{
			Method: http.MethodGet,
			Path:   "/static-media/",
			Handler: func(w http.ResponseWriter, r *http.Request) {
				http.StripPrefix("/static-media/", fs).ServeHTTP(w, r)
			},
		})
	}

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	fmt.Printf("Swagger UI: http://%s:%d/swagger/\n", c.Host, c.Port)
	server.Start()
}

// applyStorageEnvOverrides 用环境变量补全/覆盖对象存储（敏感信息建议只走环境变量）。
func applyStorageEnvOverrides(c *config.Config) {
	pick := func(primary, yamlVal string, fallbacks ...string) string {
		if strings.TrimSpace(primary) != "" {
			return strings.TrimSpace(primary)
		}
		if strings.TrimSpace(yamlVal) != "" {
			return strings.TrimSpace(yamlVal)
		}
		for _, e := range fallbacks {
			if v := strings.TrimSpace(os.Getenv(e)); v != "" {
				return v
			}
		}
		return strings.TrimSpace(yamlVal)
	}

	if v := strings.TrimSpace(os.Getenv("CONTENT_STORAGE_DRIVER")); v != "" {
		c.Storage.Driver = v
	}

	c.Storage.Endpoint = pick(os.Getenv("CONTENT_OSS_ENDPOINT"), c.Storage.Endpoint, "OSS_ENDPOINT")
	c.Storage.AccessKey = pick(os.Getenv("CONTENT_OSS_ACCESS_KEY_ID"), c.Storage.AccessKey, "OSS_ACCESS_KEY_ID", "OSS_ACCESS_KEY")
	c.Storage.SecretKey = pick(os.Getenv("CONTENT_OSS_ACCESS_KEY_SECRET"), c.Storage.SecretKey, "OSS_ACCESS_KEY_SECRET", "OSS_SECRET_KEY")
	c.Storage.Bucket = pick(os.Getenv("CONTENT_OSS_BUCKET"), c.Storage.Bucket, "OSS_BUCKET_NAME", "OSS_BUCKET")
	if v := pick(os.Getenv("CONTENT_OSS_CDN_BASE_URL"), c.Storage.CdnBaseUrl, "OSS_PUBLIC_BASE_URL"); v != "" {
		c.Storage.CdnBaseUrl = v
	}
}

// maybePromoteLocalToOSS YAML 仍为 local，但四类 OSS 必填项均已配置且 Endpoint 形如阿里云 OSS 时，升级为 oss（避免误以为已上云）。
func maybePromoteLocalToOSS(c *config.Config) {
	d := strings.ToLower(strings.TrimSpace(c.Storage.Driver))
	if d != "local" && d != "" {
		return
	}
	ep := strings.TrimSpace(c.Storage.Endpoint)
	ak := strings.TrimSpace(c.Storage.AccessKey)
	sk := strings.TrimSpace(c.Storage.SecretKey)
	bk := strings.TrimSpace(c.Storage.Bucket)
	if ep == "" || ak == "" || sk == "" || bk == "" {
		return
	}
	el := strings.ToLower(ep)
	if !strings.Contains(el, "aliyuncs.com") || !strings.Contains(el, "oss") {
		return
	}
	fmt.Printf("[content-service] Storage.Driver was %q but OSS-style endpoint/credentials are set — switching to driver=oss (bucket=%s endpoint=%s)\n",
		c.Storage.Driver, bk, ep)
	c.Storage.Driver = "oss"
}
