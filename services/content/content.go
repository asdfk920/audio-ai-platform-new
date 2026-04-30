// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/jacklau/audio-ai-platform/services/content/internal/config"
	"github.com/jacklau/audio-ai-platform/services/content/internal/handler"
	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"

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

	// 本地存储时，通过 /static-media/* 提供上传文件访问（与 CdnBaseUrl 对齐）
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
	server.Start()
}
