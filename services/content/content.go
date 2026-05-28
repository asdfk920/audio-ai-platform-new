package main

import (
	"flag"

	apicors "github.com/jacklau/audio-ai-platform/common/cors"
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
	server.Use(apicors.Middleware(c.CORS))
	defer server.Stop()

	ctx, err := svc.NewServiceContext(c)
	if err != nil {
		panic(err)
	}

	handler.RegisterHandlers(server, ctx)
	server.Start()
}
