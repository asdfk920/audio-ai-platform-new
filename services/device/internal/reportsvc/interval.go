package reportsvc

import (
	"strings"

	"github.com/jacklau/audio-ai-platform/services/device/internal/config"
)

func (s *Service) resolveNextIntervalSeconds(reports []normalizedReport) int64 {
	cfg := statusReportConfigOrDefault(s.svcCtx.Config.StatusReport)
	if len(reports) == 0 {
		return clampIntervalSec(int64(cfg.IntervalNormalSec), cfg)
	}
	rep := reports[len(reports)-1].Reported

	if isCharging(rep) {
		return clampIntervalSec(int64(cfg.IntervalNormalSec), cfg)
	}
	if isEmergencyReport(rep) {
		return clampIntervalSec(int64(cfg.IntervalEmergencySec), cfg)
	}
	if isEnergySavingMode(rep) && !isCharging(rep) {
		return clampIntervalSec(int64(cfg.IntervalEnergySec), cfg)
	}
	bat, ok := effectiveBatteryPercent(rep)
	if !ok {
		return clampIntervalSec(int64(cfg.IntervalNormalSec), cfg)
	}
	energyTh := int32(cfg.EnergyBatteryPercent)
	midTh := int32(cfg.MidRangeBatteryPercent)
	if energyTh <= 0 {
		energyTh = 20
	}
	if midTh <= 0 {
		midTh = 50
	}
	if midTh <= energyTh {
		midTh = energyTh + 1
	}
	if bat < energyTh {
		return clampIntervalSec(int64(cfg.IntervalEnergySec), cfg)
	}
	if bat < midTh {
		return clampIntervalSec(int64(cfg.IntervalMidRangeSec), cfg)
	}
	return clampIntervalSec(int64(cfg.IntervalNormalSec), cfg)
}

func statusReportConfigOrDefault(c config.StatusReport) config.StatusReport {
	out := c
	if out.IntervalNormalSec <= 0 {
		out.IntervalNormalSec = 60
	}
	if out.IntervalMidRangeSec <= 0 {
		out.IntervalMidRangeSec = 120
	}
	if out.IntervalEnergySec <= 0 {
		out.IntervalEnergySec = 300
	}
	if out.IntervalEmergencySec <= 0 {
		out.IntervalEmergencySec = 10
	}
	if out.IntervalMinSec <= 0 {
		out.IntervalMinSec = 10
	}
	if out.IntervalMaxSec <= 0 {
		out.IntervalMaxSec = 300
	}
	if out.EnergyBatteryPercent <= 0 {
		out.EnergyBatteryPercent = 20
	}
	if out.MidRangeBatteryPercent <= 0 {
		out.MidRangeBatteryPercent = 50
	}
	return out
}

func clampIntervalSec(v int64, cfg config.StatusReport) int64 {
	minSec := int64(cfg.IntervalMinSec)
	maxSec := int64(cfg.IntervalMaxSec)
	if v < minSec {
		return minSec
	}
	if v > maxSec {
		return maxSec
	}
	return v
}

func effectiveBatteryPercent(reported map[string]interface{}) (int32, bool) {
	if v, ok := int32Value(reported["battery"]); ok {
		return v, true
	}
	power := nestedMap(reported, "power")
	if power != nil {
		if v, ok := int32Value(power["percent"]); ok {
			return v, true
		}
	}
	return 0, false
}

func nestedMap(m map[string]interface{}, key string) map[string]interface{} {
	if m == nil {
		return nil
	}
	v, ok := m[key]
	if !ok || v == nil {
		return nil
	}
	sub, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}
	return sub
}

func isCharging(reported map[string]interface{}) bool {
	if b, ok := reported["is_charging"].(bool); ok && b {
		return true
	}
	power := nestedMap(reported, "power")
	if power != nil {
		if b, ok := power["is_charging"].(bool); ok && b {
			return true
		}
		if s := strings.ToLower(strings.TrimSpace(asString(power["charging_state"]))); s != "" {
			switch s {
			case "charging", "full", "charged":
				return true
			}
		}
	}
	return false
}

func isEnergySavingMode(reported map[string]interface{}) bool {
	s := strings.ToLower(strings.TrimSpace(asString(reported["report_mode"])))
	return s == "energy_saving"
}

func isEmergencyReport(reported map[string]interface{}) bool {
	if b, ok := reported["emergency"].(bool); ok && b {
		return true
	}
	if strings.ToLower(strings.TrimSpace(asString(reported["report_mode"]))) == "emergency" {
		return true
	}
	alerts, ok := reported["alerts"].([]interface{})
	if !ok || len(alerts) == 0 {
		return false
	}
	for _, a := range alerts {
		m, ok := a.(map[string]interface{})
		if !ok {
			continue
		}
		sev := strings.ToLower(strings.TrimSpace(asString(m["severity"])))
		if sev == "critical" || sev == "emergency" {
			return true
		}
		lvl := strings.ToLower(strings.TrimSpace(asString(m["level"])))
		if lvl == "critical" || lvl == "emergency" {
			return true
		}
	}
	return false
}
