package config

import "github.com/zeromicro/go-zero/rest"

type Config struct {
	rest.RestConf
	Postgres struct {
		DataSource string
	}
	Redis struct {
		Addr     string
		Password string
		DB       int
		Optional bool
	}
	Auth struct {
		AccessSecret string
	}
	Stream struct {
		RTMPBaseURL        string
		FLVBaseURL         string
		DefaultSecretKey   string
		DefaultConfigID    string
		DefaultExpiresSec  int64
	}
	SQS struct {
		Enabled  bool
		Endpoint string
		QueueURL  string
		Region   string
	}
}

