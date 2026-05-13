package svc

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/services/ai-worker/internal/config"
	"github.com/jacklau/audio-ai-platform/services/ai-worker/internal/model"
)

type ServiceContext struct {
	Config    config.Config
	AIService *model.AIService
	DB        *sql.DB
}

func NewServiceContext(c config.Config) (*ServiceContext, error) {
	logx.Info("Initializing AI Worker ServiceContext...")

	db, err := initDatabase(c)
	if err != nil {
		logx.Errorf("Failed to initialize database: %v", err)
		return nil, err
	}

	aiSvc, err := initializeAIService(c)
	if err != nil {
		logx.Errorf("Failed to initialize AI service: %v", err)
		return nil, err
	}

	svcCtx := &ServiceContext{
		Config:    c,
		AIService: aiSvc,
		DB:        db,
	}

	logx.Info("AI Worker ServiceContext initialized successfully")
	return svcCtx, nil
}

func initDatabase(c config.Config) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.DBName,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("打开数据库连接失败：%w", err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("连接数据库失败：%w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * 60 * 1000000000) // 5分钟（纳秒）

	logx.Info("✅ 数据库连接成功")

	return db, nil
}

func initializeAIService(c config.Config) (*model.AIService, error) {
	modelConfig := &model.ModelConfig{
		Type:              model.ModelBSRFormer,
		Name:              c.AI.ModelName,
		ModelPath:         c.AI.ModelPath,
		GPUDevice:         c.AI.GPUID,
		SegmentSize:       int(c.AI.SegmentSize), // 转换为 int
		Overlap:           c.AI.Overlap,
		UseFP16:           c.AI.UseFP16,
		BatchSize:         1,
		OutputDir:         c.AI.OutputDir,
		CacheDir:          c.AI.CacheDir,
		MaxConcurrentJobs: c.AI.MaxConcurrentJobs,
	}

	svc, err := model.NewAIService(modelConfig)
	if err != nil {
		return nil, err
	}

	return svc, nil
}
