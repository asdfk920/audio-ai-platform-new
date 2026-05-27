package svc

import (
	"database/sql"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/pkg/redisx"
	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/config"
	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/drmpass"
	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/repository"
	"github.com/jacklau/audio-ai-platform/services/media-processing/netease"

	"github.com/zeromicro/go-zero/core/logx"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type ServiceContext struct {
	Config                config.Config
	DB                    *sql.DB
	RedisAvailable        bool
	ThirdPartyAccountRepo *repository.ThirdPartyAccountRepo
	NeteaseClient         *netease.Client
	// DRMPass 可选：私钥失败时 Signer 为 nil，仅做上游头透传
	DRMPass *DRMPassRuntime
}

// DRMPassRuntime 媒体代理侧 DRM 元数据策略
type DRMPassRuntime struct {
	AttestEnabled bool
	Issuer        string
	Signer        *drmpass.Signer
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

	var drm = &DRMPassRuntime{
		Issuer: strings.TrimSpace(c.DRMPass.Issuer),
	}
	if drm.Issuer == "" {
		drm.Issuer = "audio-ai-media-processing"
	}
	if kp := strings.TrimSpace(c.DRMPass.SigningKeyPath); kp != "" {
		s, err := drmpass.LoadSignerFromPath(kp)
		if err != nil {
			logx.Errorf("svc: DRMPass SigningKeyPath 加载失败: %v", err)
		} else {
			drm.Signer = s
		}
	}
	if drm.Signer == nil {
		if pem := strings.TrimSpace(c.DRMPass.SigningKeyPEM); pem != "" {
			s, err := drmpass.LoadSignerFromPEM([]byte(pem))
			if err != nil {
				logx.Errorf("svc: DRMPass SigningKeyPEM 解析失败: %v", err)
			} else {
				drm.Signer = s
			}
		}
	}
	drm.AttestEnabled = c.DRMPass.AttestSigning && drm.Signer != nil
	if c.DRMPass.AttestSigning && drm.Signer == nil {
		logx.Infof("svc: DRMPass.AttestSigning=true 但未配置有效私钥，已跳过平台签名")
	}

	sctx := &ServiceContext{
		Config:                c,
		DB:                    db,
		RedisAvailable:        redisOK,
		ThirdPartyAccountRepo: repository.NewThirdPartyAccountRepo(db),
		DRMPass:               drm,
	}

	if c.ThirdPartyMusic.BaseURL != "" && c.ThirdPartyMusic.AppID != "" && c.ThirdPartyMusic.AppSecret != "" {
		sctx.NeteaseClient = netease.NewClient(
			c.ThirdPartyMusic.BaseURL,
			c.ThirdPartyMusic.AppID,
			c.ThirdPartyMusic.AppSecret,
		)
		logx.Infof("svc: ✅ 初始化网易云音乐客户端 | BaseURL: %s | AppID: %s",
			c.ThirdPartyMusic.BaseURL, c.ThirdPartyMusic.AppID)
	} else {
		logx.Errorf("svc: ⚠️ 未配置网易云音乐凭证（BaseURL/AppID/AppSecret），第三方登录功能将不可用")
	}

	return sctx, nil
}
