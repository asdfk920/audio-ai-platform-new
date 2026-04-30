package svc

import (
	"database/sql"
	"time"

	"github.com/jacklau/audio-ai-platform/pkg/redisx"
	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/config"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type ServiceContext struct {
	Config config.Config
	DB     *sql.DB
	RedisAvailable bool
}

func NewServiceContext(c config.Config) (*ServiceContext, error) {
	db, err := sql.Open("pgx", c.Postgres.DataSource)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)

	redisOK := false
	if c.Redis.Addr != "" {
		if err := redisx.Init(redisx.Config{
			Addr:     c.Redis.Addr,
			Password: c.Redis.Password,
			DB:       c.Redis.DB,
		}); err != nil {
			if !c.Redis.Optional {
				_ = db.Close()
				return nil, err
			}
		} else {
			redisOK = true
		}
	}

	return &ServiceContext{Config: c, DB: db, RedisAvailable: redisOK}, nil
}

