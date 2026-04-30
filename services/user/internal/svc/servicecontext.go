package svc

import (
	"database/sql"

	"github.com/jacklau/audio-ai-platform/services/user/internal/config"
	"github.com/jacklau/audio-ai-platform/services/user/internal/entitlementsvc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/repo/dao"
)

type ServiceContext struct {
	Config         config.Config
	DB             *sql.DB
	SendVerify     *dao.SendVerifyRepo
	UserRepo       *dao.UserRepo
	DeviceBind     *dao.UserDeviceBindRepo
	MemberOrder    *dao.MemberOrderRepo
	DownloadRecord *dao.UserDownloadRecordRepo
	Entitlement    *entitlementsvc.Service
}

func NewServiceContext(c config.Config, db *sql.DB) *ServiceContext {
	return &ServiceContext{
		Config:         c,
		DB:             db,
		SendVerify:     dao.NewSendVerifyRepo(db),
		UserRepo:       dao.NewUserRepo(db),
		DeviceBind:     dao.NewUserDeviceBindRepo(db),
		MemberOrder:    dao.NewMemberOrderRepo(db),
		DownloadRecord: dao.NewUserDownloadRecordRepo(db),
		Entitlement:    entitlementsvc.NewService(),
	}
}
