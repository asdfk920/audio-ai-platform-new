package svc

import (
	"fmt"
	"strings"

	"github.com/jacklau/audio-ai-platform/services/content/internal/config"
	"github.com/jacklau/audio-ai-platform/services/content/internal/repo/schema"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ServiceContext 内容服务依赖
type ServiceContext struct {
	Config config.Config
	DB     *gorm.DB
}

// NewServiceContext 初始化
func NewServiceContext(c config.Config) (*ServiceContext, error) {
	dsn := strings.TrimSpace(c.Database.DataSource)
	if dsn == "" {
		return nil, fmt.Errorf("Database.DataSource 未配置")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}
	if err := schema.EnsurePlaylistTables(db); err != nil {
		logx.Errorf("EnsurePlaylistTables: %v", err)
		return nil, fmt.Errorf("初始化歌单表失败: %w", err)
	}
	if err := schema.EnsureArtistSubscriptionTables(db); err != nil {
		logx.Errorf("EnsureArtistSubscriptionTables: %v", err)
		return nil, fmt.Errorf("初始化艺术家订阅表失败: %w", err)
	}
	if err := schema.EnsureMessageTables(db); err != nil {
		logx.Errorf("EnsureMessageTables: %v", err)
		return nil, fmt.Errorf("初始化消息表失败: %w", err)
	}
	return &ServiceContext{Config: c, DB: db}, nil
}
