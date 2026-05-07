package svc

import (
	"database/sql"

	"github.com/jacklau/audio-ai-platform/services/ota/internal/config"
	"github.com/jacklau/audio-ai-platform/services/ota/internal/model"
)

type ServiceContext struct {
	Config          config.Config
	OtaFirmwareRepo *model.OtaFirmwareRepo
	DeviceRepo      *model.DeviceRepo
	DB              *sql.DB
}

func NewServiceContext(c config.Config, db *sql.DB) *ServiceContext {
	return &ServiceContext{
		Config:          c,
		OtaFirmwareRepo: model.NewOtaFirmwareRepo(db),
		DeviceRepo:      model.NewDeviceRepo(db),
		DB:              db,
	}
}
