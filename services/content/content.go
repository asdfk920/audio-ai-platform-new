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
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	_ "github.com/jacklau/audio-ai-platform/services/content/docs"

	"github.com/jacklau/audio-ai-platform/services/content/internal/config"
	"github.com/jacklau/audio-ai-platform/services/content/internal/handler"
	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"

	httpSwagger "github.com/swaggo/http-swagger/v2"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/content.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	server.AddRoute(rest.Route{
		Method: http.MethodGet,
		Path:   "/swagger/",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			httpSwagger.Handler(
				httpSwagger.URL("http://localhost:8003/swagger/doc.json"),
			).ServeHTTP(w, r)
		},
	})

	server.AddRoute(rest.Route{
		Method: http.MethodGet,
		Path:   "/swagger/index.html",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			httpSwagger.Handler(
				httpSwagger.URL("http://localhost:8003/swagger/doc.json"),
			).ServeHTTP(w, r)
		},
	})

	server.AddRoute(rest.Route{
		Method: http.MethodGet,
		Path:   "/swagger/doc.json",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			http.ServeFile(w, r, "./docs/swagger.json")
		},
	})

	root := strings.TrimSpace(c.Local.Root)
	if root == "" {
		root = "./data/content-objects"
	}
	if strings.EqualFold(strings.TrimSpace(c.Storage.Driver), "local") {
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
	fmt.Printf("Swagger UI: http://%s:%d/swagger/index.html\n", c.Host, c.Port)
	server.Start()
}
