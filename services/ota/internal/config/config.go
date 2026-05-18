package config

import (
	apicors "github.com/jacklau/audio-ai-platform/common/cors"
	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	rest.RestConf
	Auth struct {
		AccessSecret string
	}
	Postgres struct {
		DataSource string
	}
	Redis struct {
		Addr     string
		Password string
		DB       int
	}
	CORS apicors.Config `json:",optional"`
}
