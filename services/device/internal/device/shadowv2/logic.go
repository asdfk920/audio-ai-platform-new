package shadowv2

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/logx"
)

type ShadowLogic struct {
	ctx    context.Context
	svcCtx interface {
		GetRedisShadowStore() *RedisShadowStore
	}
	notifier *ShadowNotifier
}

func NewShadowLogic(ctx context.Context, svcCtx interface {
	GetRedisShadowStore() *RedisShadowStore
}) *ShadowLogic {
	return &ShadowLogic{
		ctx:      ctx,
		svcCtx:   svcCtx,
		notifier: nil,
	}
}

func NewShadowLogicWithNotifier(ctx context.Context, svcCtx interface {
	GetRedisShadowStore() *RedisShadowStore
}, notifier *ShadowNotifier) *ShadowLogic {
	return &ShadowLogic{
		ctx:      ctx,
		svcCtx:   svcCtx,
		notifier: notifier,
	}
}

func (l *ShadowLogic) InitDeviceShadow(deviceSN string) (*InitShadowResp, error) {
	if deviceSN == "" {
		return nil, fmt.Errorf("device_sn cannot be empty")
	}

	store := l.svcCtx.GetRedisShadowStore()
	if store == nil {
		return nil, fmt.Errorf("redis store not initialized")
	}

	exists, err := store.ShadowExists(l.ctx, deviceSN)
	if err != nil {
		logx.Errorf("shadowv2: check shadow exists failed: %v", err)
		return nil, fmt.Errorf("system error")
	}

	if exists {
		return &InitShadowResp{
			Success: false,
			Message: "shadow already exists",
		}, nil
	}

	_, err = store.InitShadow(l.ctx, deviceSN)
	if err != nil {
		logx.Errorf("shadowv2: init shadow failed: %v", err)
		return nil, err
	}

	return &InitShadowResp{
		Success: true,
		Message: "shadow initialized successfully",
	}, nil
}

func (l *ShadowLogic) UpdateReportedState(req *UpdateReportedReq) (*UpdateResp, error) {
	if req.DeviceSN == "" {
		return nil, fmt.Errorf("device_sn cannot be empty")
	}
	if req.Reported == nil || len(req.Reported) == 0 {
		return nil, fmt.Errorf("reported cannot be empty")
	}

	store := l.svcCtx.GetRedisShadowStore()
	if store == nil {
		return nil, fmt.Errorf("redis store not initialized")
	}

	exists, err := store.ShadowExists(l.ctx, req.DeviceSN)
	if err != nil {
		return nil, fmt.Errorf("system error: %w", err)
	}
	if !exists {
		_, initErr := store.InitShadow(l.ctx, req.DeviceSN)
		if initErr != nil {
			logx.Slowf("shadowv2: auto init failed, try update: %v", initErr)
		}
	}

	shadow, err := store.UpdateReported(l.ctx, req.DeviceSN, req.Reported)
	if err != nil {
		logx.Errorf("shadowv2: update reported failed: %v", err)
		return nil, err
	}

	resp := &UpdateResp{
		DeviceSN:   shadow.DeviceSN,
		Reported:   shadow.Reported,
		Version:    shadow.Version,
		UpdateTime: shadow.UpdateTime,
		Status:     shadow.Status,
	}

	desired, getErr := store.GetShadow(l.ctx, req.DeviceSN)
	if getErr == nil && desired != nil {
		resp.Desired = desired.Desired
	} else {
		resp.Desired = make(map[string]interface{})
	}

	logx.Infof("shadowv2: device %s reported updated, version=%d", req.DeviceSN, shadow.Version)

	if l.notifier != nil {
		l.notifier.Notify(req.DeviceSN, map[string]interface{}{
			"cmd":       "device_status_change",
			"device_sn": req.DeviceSN,
			"data": map[string]interface{}{
				"online":      shadow.Status == StatusOnline,
				"version":     shadow.Version,
				"reported":    shadow.Reported,
				"update_time": shadow.UpdateTime,
				"status":      shadow.Status,
			},
		})
	}

	return resp, nil
}

func (l *ShadowLogic) UpdateDesiredState(req *UpdateDesiredReq) (*UpdateResp, error) {
	if req.DeviceSN == "" {
		return nil, fmt.Errorf("device_sn cannot be empty")
	}
	if req.Desired == nil || len(req.Desired) == 0 {
		return nil, fmt.Errorf("desired cannot be empty")
	}

	store := l.svcCtx.GetRedisShadowStore()
	if store == nil {
		return nil, fmt.Errorf("redis store not initialized")
	}

	exists, err := store.ShadowExists(l.ctx, req.DeviceSN)
	if err != nil {
		return nil, fmt.Errorf("system error: %w", err)
	}
	if !exists {
		_, initErr := store.InitShadow(l.ctx, req.DeviceSN)
		if initErr != nil {
			logx.Slowf("shadowv2: auto init failed, try update: %v", initErr)
		}
	}

	shadow, err := store.UpdateDesired(l.ctx, req.DeviceSN, req.Desired)
	if err != nil {
		logx.Errorf("shadowv2: update desired failed: %v", err)
		return nil, err
	}

	resp := &UpdateResp{
		DeviceSN:   shadow.DeviceSN,
		Desired:    shadow.Desired,
		Version:    shadow.Version,
		UpdateTime: shadow.UpdateTime,
		Status:     shadow.Status,
	}

	reported, getErr := store.GetShadow(l.ctx, req.DeviceSN)
	if getErr == nil && reported != nil {
		resp.Reported = reported.Reported
	} else {
		resp.Reported = make(map[string]interface{})
	}

	logx.Infof("shadowv2: device %s desired updated, version=%d", req.DeviceSN, shadow.Version)

	if l.notifier != nil {
		l.notifier.Notify(req.DeviceSN, map[string]interface{}{
			"cmd":       "device_status_change",
			"device_sn": req.DeviceSN,
			"data": map[string]interface{}{
				"online":      shadow.Status == StatusOnline,
				"version":     shadow.Version,
				"desired":     shadow.Desired,
				"update_time": shadow.UpdateTime,
				"status":      shadow.Status,
			},
		})
	}

	return resp, nil
}

func (l *ShadowLogic) QueryShadow(deviceSN string) (*QueryShadowResp, error) {
	if deviceSN == "" {
		return nil, fmt.Errorf("device_sn cannot be empty")
	}

	store := l.svcCtx.GetRedisShadowStore()
	if store == nil {
		return nil, fmt.Errorf("redis store not initialized")
	}

	shadow, err := store.GetShadow(l.ctx, deviceSN)
	if err != nil {
		if err.Error() == "shadow not found" {
			return &QueryShadowResp{
				DeviceSN: deviceSN,
				Message:  "shadow not found",
			}, nil
		}
		logx.Errorf("shadowv2: query shadow failed: %v", err)
		return nil, err
	}

	resp := &QueryShadowResp{
		DeviceSN:   shadow.DeviceSN,
		Reported:   shadow.Reported,
		Desired:    shadow.Desired,
		Version:    shadow.Version,
		UpdateTime: shadow.UpdateTime,
		Status:     shadow.Status,
	}

	logx.Infof("shadowv2: query device %s shadow success", deviceSN)
	return resp, nil
}

func (l *ShadowLogic) QueryReportedOnly(deviceSN string) (map[string]interface{}, int64, error) {
	if deviceSN == "" {
		return nil, 0, fmt.Errorf("device_sn cannot be empty")
	}

	store := l.svcCtx.GetRedisShadowStore()
	if store == nil {
		return nil, 0, fmt.Errorf("redis store not initialized")
	}

	reported, version, err := store.GetReported(l.ctx, deviceSN)
	if err != nil {
		logx.Errorf("shadowv2: query reported failed: %v", err)
		return nil, 0, err
	}

	return reported, version, nil
}

func (l *ShadowLogic) QueryVersionOnly(deviceSN string) (int64, error) {
	if deviceSN == "" {
		return 0, fmt.Errorf("device_sn cannot be empty")
	}

	store := l.svcCtx.GetRedisShadowStore()
	if store == nil {
		return 0, fmt.Errorf("redis store not initialized")
	}

	version, err := store.GetVersion(l.ctx, deviceSN)
	if err != nil {
		logx.Errorf("shadowv2: query version failed: %v", err)
		return 0, err
	}

	return version, nil
}

func (l *ShadowLogic) UpdateOnlineStatus(req *UpdateStatusReq) (*UpdateResp, error) {
	if req.DeviceSN == "" {
		return nil, fmt.Errorf("device_sn cannot be empty")
	}
	if req.Status == "" {
		return nil, fmt.Errorf("status cannot be empty")
	}

	validStatuses := map[DeviceStatus]bool{
		StatusOnline:   true,
		StatusOffline:  true,
		StatusAbnormal: true,
	}
	if !validStatuses[req.Status] {
		return nil, fmt.Errorf("invalid status: %s", req.Status)
	}

	store := l.svcCtx.GetRedisShadowStore()
	if store == nil {
		return nil, fmt.Errorf("redis store not initialized")
	}

	exists, err := store.ShadowExists(l.ctx, req.DeviceSN)
	if err != nil {
		return nil, fmt.Errorf("system error: %w", err)
	}
	if !exists {
		_, initErr := store.InitShadow(l.ctx, req.DeviceSN)
		if initErr != nil {
			logx.Slowf("shadowv2: auto init failed, try update: %v", initErr)
		}
	}

	shadow, err := store.UpdateStatus(l.ctx, req.DeviceSN, req.Status)
	if err != nil {
		logx.Errorf("shadowv2: update status failed: %v", err)
		return nil, err
	}

	fullShadow, getErr := store.GetShadow(l.ctx, req.DeviceSN)
	if getErr != nil || fullShadow == nil {
		fullShadow = shadow
	}

	resp := &UpdateResp{
		DeviceSN:   fullShadow.DeviceSN,
		Reported:   fullShadow.Reported,
		Desired:    fullShadow.Desired,
		Version:    shadow.Version,
		UpdateTime: shadow.UpdateTime,
		Status:     req.Status,
	}

	logx.Infof("shadowv2: device %s status updated to %s, version=%d", req.DeviceSN, req.Status, shadow.Version)

	if l.notifier != nil {
		l.notifier.Notify(req.DeviceSN, map[string]interface{}{
			"cmd":       "device_status_change",
			"device_sn": req.DeviceSN,
			"data": map[string]interface{}{
				"online":      req.Status == StatusOnline,
				"version":     shadow.Version,
				"update_time": shadow.UpdateTime,
				"status":      req.Status,
			},
		})
	}

	return resp, nil
}

func (l *ShadowLogic) CASUpdateWithCheck(req *CASUpdateReq) (*CASUpdateResp, error) {
	if req.DeviceSN == "" {
		return nil, fmt.Errorf("device_sn cannot be empty")
	}
	if req.ExpectVersion <= 0 {
		return nil, fmt.Errorf("expect_version must > 0")
	}

	store := l.svcCtx.GetRedisShadowStore()
	if store == nil {
		return nil, fmt.Errorf("redis store not initialized")
	}

	exists, err := store.ShadowExists(l.ctx, req.DeviceSN)
	if err != nil {
		return nil, fmt.Errorf("system error: %w", err)
	}
	if !exists {
		return &CASUpdateResp{
			Success: false,
			Message: "shadow not found",
			Error:   "please init shadow first",
		}, nil
	}

	resp, err := store.CASUpdate(l.ctx, req)
	if err != nil {
		logx.Errorf("shadowv2: CAS update failed: %v", err)
		return nil, err
	}

	if resp.Success {
		logx.Infof("shadowv2: device %s CAS update success, new version=%d", req.DeviceSN, resp.Shadow.Version)
	} else {
		logx.Slowf("shadowv2: device %s CAS update failed: %s", req.DeviceSN, resp.Error)
	}

	return resp, nil
}
