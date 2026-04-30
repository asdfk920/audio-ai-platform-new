package reportsvc

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// emitStatusAlerts 根据 reported 阈值写结构化日志，并异步落库 device_status_alert（失败仅打日志）。
func (s *Service) emitStatusAlerts(deviceID int64, sn string, reported map[string]interface{}) {
	if s == nil || s.svcCtx == nil || s.svcCtx.DB == nil || reported == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	bat, hasBat := effectiveBatteryPercent(reported)
	if hasBat {
		switch {
		case bat < 10:
			s.logAndPersist(ctx, deviceID, sn, "battery", "critical", map[string]interface{}{"percent": bat})
		case bat < 20:
			s.logAndPersist(ctx, deviceID, sn, "battery", "warning", map[string]interface{}{"percent": bat})
		}
	}

	st := nestedMap(reported, "storage")
	if st != nil {
		total, okT := toFloat64(st["total_bytes"])
		used, okU := toFloat64(st["used_bytes"])
		if okT && okU && total > 0 {
			ratio := used / total
			switch {
			case ratio >= 0.9:
				s.logAndPersist(ctx, deviceID, sn, "storage", "critical", map[string]interface{}{"used_ratio": ratio})
			case ratio >= 0.8:
				s.logAndPersist(ctx, deviceID, sn, "storage", "warning", map[string]interface{}{"used_ratio": ratio})
			}
		}
	}

	uwb := nestedMap(reported, "uwb")
	if uwb != nil {
		if acc, ok := toFloat64(uwb["accuracy_m"]); ok && acc > 5 {
			s.logAndPersist(ctx, deviceID, sn, "uwb_accuracy", "warning", map[string]interface{}{"accuracy_m": acc})
		}
	}

	acoustic := nestedMap(reported, "acoustic")
	if acoustic != nil {
		state := strings.ToLower(strings.TrimSpace(asString(acoustic["calibration_state"])))
		if state != "" && state != "calibrated" {
			s.logAndPersist(ctx, deviceID, sn, "calibration", "info", map[string]interface{}{"calibration_state": state})
		}
	}
}

func (s *Service) logAndPersist(ctx context.Context, deviceID int64, sn, alertType, severity string, payload map[string]interface{}) {
	logx.Infof("[device_status_alert] sn=%s device_id=%d type=%s severity=%s payload=%v",
		sn, deviceID, alertType, severity, payload)
	if severity == "info" {
		return
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_, err = s.svcCtx.DB.ExecContext(ctx, `
INSERT INTO public.device_status_alert (device_id, alert_type, severity, payload)
VALUES ($1, $2, $3, $4::jsonb)`,
		deviceID, alertType, severity, string(b))
	if err != nil {
		logx.Infof("device_status_alert persist failed sn=%s type=%s: %v", sn, alertType, err)
	}
}
