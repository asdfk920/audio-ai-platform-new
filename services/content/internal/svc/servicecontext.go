package svc

import (
	"context"

	"github.com/jacklau/audio-ai-platform/services/content/internal/config"
	"github.com/jacklau/audio-ai-platform/services/content/internal/pkg/storage"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type ServiceContext struct {
	Config  config.Config
	DB      *gorm.DB
	Redis   *redis.Redis
	Storage storage.Uploader
}

func NewServiceContext(c config.Config) *ServiceContext {
	ctx := context.Background()
	var db *gorm.DB
	dsn := c.Database.DataSource
	if dsn != "" {
		gdb, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Warn),
		})
		if err != nil {
			logx.Errorf("content service: postgres open failed: %v", err)
		} else {
			db = gdb
		}
	} else {
		logx.Error("content service: Database.DataSource empty, catalog upload disabled")
	}

	up, err := storage.NewUploader(ctx, c)
	if err != nil {
		logx.Errorf("content service: storage init failed: %v", err)
		up = nil
	}

	var rds *redis.Redis
	if len(c.CacheRedis) > 0 {
		r, rerr := redis.NewRedis(c.CacheRedis[0], redis.WithPass(c.CacheRedis[0].Pass))
		if rerr != nil {
			logx.Errorf("content service: redis init failed: %v", rerr)
		} else {
			rds = r
		}
	}

	return &ServiceContext{
		Config:  c,
		DB:      db,
		Redis:   rds,
		Storage: up,
	}
}
