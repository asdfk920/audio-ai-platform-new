// Package reportsvc 设备状态上报服务包
// 处理设备通过 HTTP/MQTT 上报的状态数据，包括状态解析、验证、影子同步等
package reportsvc

import (
	"bytes"
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/device/internal/deviceauthsvc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/repo"
	"github.com/jacklau/audio-ai-platform/services/device/internal/shadowsvc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
)

// allowedRunStates 允许的设备运行状态枚举
var allowedRunStates = map[string]struct{}{
	"normal": {}, "fault": {}, "sleep": {}, "upgrading": {},
}

// Service 设备状态上报服务结构体
// 提供设备状态上报的处理方法，包括验证、解析、影子同步等
type Service struct {
	svcCtx *svc.ServiceContext
	shadow *shadowsvc.Service
}

// ReportInput 设备状态上报输入参数结构体
// 包含设备上报的所有状态信息，如电量、固件版本、在线状态、子设备信息等
type ReportInput struct {
	DeviceSN        string
	DeviceSecret    string
	ReportID        string
	Timestamp       int64
	Reported        json.RawMessage
	RunState        string
	Battery         *int32
	FirmwareVersion string
	Online          *bool
	IP              string
	Children        []ChildReportInput
	History         []CachedReportInput
	Source          string
}

// ChildReportInput 子设备上报输入参数结构体
// 用于上报子设备的状态信息，如在线状态、上报数据等
type ChildReportInput struct {
	ChildKey  string
	ChildSN   string
	ChildType string
	ChildName string
	Timestamp int64
	Online    *bool
	Reported  json.RawMessage
	Metadata  json.RawMessage
}

// CachedReportInput 缓存的上报输入参数结构体
// 用于批量上报时携带的历史上报数据
type CachedReportInput struct {
	ReportID        string
	Timestamp       int64
	Reported        json.RawMessage
	RunState        string
	Battery         *int32
	FirmwareVersion string
	Online          *bool
	IP              string
	Children        []ChildReportInput
}

// ReportOutput 设备状态上报输出结果结构体
// 包含处理结果，如版本号、待执行命令、下次上报间隔等
type ReportOutput struct {
	DeviceSN        string
	Version         int64
	Commands        []shadowsvc.PendingCommand
	NextInterval    int64
	AcceptedReports int
}

type normalizedReport struct {
	ReportID  string
	Reported  map[string]interface{}
	Children  []normalizedChild
	Timestamp time.Time
	IsHistory bool
	Source    string
	Raw       json.RawMessage
}

type normalizedChild struct {
	ChildKey  string
	ChildSN   string
	ChildType string
	ChildName string
	Reported  map[string]interface{}
	Metadata  map[string]interface{}
	Timestamp time.Time
}

type hostShadowState struct {
	Version        int64
	LastReportTime *time.Time
}

type childShadowState struct {
	Version        int64
	LastReportTime *time.Time
}

func New(svcCtx *svc.ServiceContext) *Service {
	return &Service{
		svcCtx: svcCtx,
		shadow: shadowsvc.New(svcCtx),
	}
}

func (s *Service) Ingest(ctx context.Context, in ReportInput) (*ReportOutput, error) {
	if s == nil || s.svcCtx == nil || s.svcCtx.DB == nil {
		return nil, errorx.NewDefaultError(errorx.CodeSystemError)
	}
	authSvc := deviceauthsvc.New(s.svcCtx)
	principal, err := authSvc.AuthenticateRequest(ctx, in.DeviceSN, in.DeviceSecret, in.IP)
	if err != nil {
		return nil, err
	}
	row := authRowFromPrincipal(principal)

	reports, err := normalizeReports(in, row)
	if err != nil {
		return nil, err
	}
	if len(reports) == 0 {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "上报内容不能为空")
	}
	for _, r := range reports {
		if err := ValidateReportedFields(r.Reported); err != nil {
			return nil, err
		}
	}

	hostState, err := s.getHostShadowState(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	applied := 0
	version := hostState.Version

	for _, report := range reports {
		batchID, inserted, err := s.insertReportBatch(ctx, row.ID, row.SN, report)
		if err != nil {
			return nil, err
		}
		if !inserted {
			continue
		}

		shouldApplyHost := shouldAdvanceCurrentState(hostState.LastReportTime, report.Timestamp)
		if err := s.insertReportEvent(ctx, batchID, row.ID, 0, "", "host", report.Timestamp, report.Raw, shouldApplyHost); err != nil {
			return nil, err
		}

		for _, child := range report.Children {
			childID, childState, err := s.ensureChildAndState(ctx, row.ID, child)
			if err != nil {
				return nil, err
			}
			applyChild := shouldAdvanceCurrentState(childState.LastReportTime, child.Timestamp)
			childPayload, _ := json.Marshal(map[string]interface{}{
				"reported": child.Reported,
				"metadata": child.Metadata,
			})
			if err := s.insertReportEvent(ctx, batchID, row.ID, childID, child.ChildKey, "child", child.Timestamp, childPayload, applyChild); err != nil {
				return nil, err
			}
			if applyChild {
				if err := s.upsertChildShadow(ctx, childID, row.ID, child, childState.Version+1); err != nil {
					return nil, err
				}
			}
		}

		if shouldApplyHost {
			view, err := s.shadow.UpdateReportedForAuthenticatedDevice(ctx, row.ID, row.SN, mustJSON(report.Reported), asString(report.Reported["ip"]))
			if err != nil {
				return nil, err
			}
			version = view.Version
			if view.LastReportTime != nil {
				hostState.LastReportTime = view.LastReportTime
			} else {
				t := report.Timestamp
				hostState.LastReportTime = &t
			}
			hostState.Version = view.Version
		}
		applied++
	}

	if version == 0 {
		latest, err := s.getHostShadowState(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		version = latest.Version
	}

	// Device reconnect scenario:
	// once a fresh report marks the device online again, try to push
	// any pending desired/diagnosis commands over MQTT immediately.
	if applied > 0 {
		s.shadow.PushPendingForDevice(ctx, row.SN)
		lastReported := reports[len(reports)-1].Reported
		go func(deviceID int64, sn string, rep map[string]interface{}) {
			defer func() { recover() }()
			s.emitStatusAlerts(deviceID, sn, rep)
		}(row.ID, row.SN, lastReported)
	}

	commands, err := s.shadow.GetPendingCommandsForAuthenticatedDevice(ctx, row.ID, row.SN, 20)
	if err != nil {
		return nil, err
	}
	nextInterval := s.resolveNextIntervalSeconds(reports)
	return &ReportOutput{
		DeviceSN:        row.SN,
		Version:         version,
		Commands:        commands,
		NextInterval:    nextInterval,
		AcceptedReports: applied,
	}, nil
}

func normalizeReports(in ReportInput, row *repo.DeviceAuthRow) ([]normalizedReport, error) {
	mainReport, err := normalizeSingleReport(in.ReportID, in.Timestamp, in.Reported, in.RunState, in.Battery, in.FirmwareVersion, in.Online, in.IP, in.Children, false, in.Source, row)
	if err != nil {
		return nil, err
	}
	reports := make([]normalizedReport, 0, 1+len(in.History))
	reports = append(reports, mainReport)
	for idx, item := range in.History {
		source := in.Source
		if source == "" {
			source = "history"
		}
		report, err := normalizeSingleReport(item.ReportID, item.Timestamp, item.Reported, item.RunState, item.Battery, item.FirmwareVersion, item.Online, item.IP, item.Children, true, source, row)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(report.ReportID) == "" {
			report.ReportID = fallbackReportID(source, row.SN, report.Timestamp, idx+1, report.Raw)
		}
		reports = append(reports, report)
	}
	if strings.TrimSpace(reports[0].ReportID) == "" {
		reports[0].ReportID = fallbackReportID(reports[0].Source, row.SN, reports[0].Timestamp, 0, reports[0].Raw)
	}
	sort.SliceStable(reports, func(i, j int) bool {
		return reports[i].Timestamp.Before(reports[j].Timestamp)
	})
	return reports, nil
}

func normalizeSingleReport(reportID string, ts int64, reportedRaw json.RawMessage, runState string, battery *int32, firmwareVersion string, online *bool, ip string, children []ChildReportInput, isHistory bool, source string, row *repo.DeviceAuthRow) (normalizedReport, error) {
	reported, err := decodeJSONObject(reportedRaw)
	if err != nil {
		return normalizedReport{}, errorx.NewCodeError(errorx.CodeInvalidParam, "reported 必须是合法 JSON 对象")
	}
	runState = strings.ToLower(strings.TrimSpace(runState))
	if runState != "" {
		if _, ok := allowedRunStates[runState]; !ok {
			return normalizedReport{}, errorx.NewCodeError(errorx.CodeInvalidParam, "run_state 不合法")
		}
		reported["run_state"] = runState
	}
	if battery != nil {
		reported["battery"] = *battery
	}
	if online != nil {
		reported["online"] = *online
	}
	fw := firstNonEmpty(strings.TrimSpace(firmwareVersion), row.FirmwareVersion)
	if fw != "" {
		reported["firmware_version"] = fw
	}
	reportIP := firstNonEmpty(strings.TrimSpace(ip), row.IP)
	if reportIP != "" {
		reported["ip"] = reportIP
	}
	if row.ProductKey != "" {
		reported["product_key"] = row.ProductKey
	}
	if row.Mac != "" {
		reported["mac"] = row.Mac
	}
	childrenNorm := make([]normalizedChild, 0, len(children))
	onlineCount := 0
	for _, item := range children {
		child, err := normalizeChild(item)
		if err != nil {
			return normalizedReport{}, err
		}
		if isChildOnline(child.Reported) {
			onlineCount++
		}
		childrenNorm = append(childrenNorm, child)
	}
	if len(childrenNorm) > 0 {
		reported["child_total"] = len(childrenNorm)
		reported["child_online"] = onlineCount
		reported["child_offline"] = len(childrenNorm) - onlineCount
	}
	if _, ok := reported["online"]; !ok {
		reported["online"] = true
	}
	eventTime := fromUnixOrNow(ts)
	raw, _ := json.Marshal(map[string]interface{}{
		"reported": reported,
		"children": childrenNorm,
	})
	return normalizedReport{
		ReportID:  strings.TrimSpace(reportID),
		Reported:  reported,
		Children:  childrenNorm,
		Timestamp: eventTime,
		IsHistory: isHistory,
		Source:    normalizeSource(source, isHistory),
		Raw:       raw,
	}, nil
}

func normalizeChild(in ChildReportInput) (normalizedChild, error) {
	reported, err := decodeJSONObject(in.Reported)
	if err != nil {
		return normalizedChild{}, errorx.NewCodeError(errorx.CodeInvalidParam, "children.reported 必须是合法 JSON 对象")
	}
	metadata, err := decodeJSONObject(in.Metadata)
	if err != nil {
		return normalizedChild{}, errorx.NewCodeError(errorx.CodeInvalidParam, "children.metadata 必须是合法 JSON 对象")
	}
	childKey := normalizeChildKey(in.ChildKey, in.ChildSN, in.ChildType)
	if childKey == "" {
		return normalizedChild{}, errorx.NewCodeError(errorx.CodeInvalidParam, "child_key 或 child_sn 不能为空")
	}
	if in.Online != nil {
		reported["online"] = *in.Online
	}
	return normalizedChild{
		ChildKey:  childKey,
		ChildSN:   strings.ToUpper(strings.TrimSpace(in.ChildSN)),
		ChildType: strings.TrimSpace(in.ChildType),
		ChildName: strings.TrimSpace(in.ChildName),
		Reported:  reported,
		Metadata:  metadata,
		Timestamp: fromUnixOrNow(in.Timestamp),
	}, nil
}

func (s *Service) getHostShadowState(ctx context.Context, deviceID int64) (*hostShadowState, error) {
	var state hostShadowState
	var last sql.NullTime
	err := s.svcCtx.DB.QueryRowContext(ctx, `
SELECT COALESCE(ds.version, 0),
       COALESCE(
         (
           SELECT MAX(event_time)
           FROM public.device_report_event dre
           WHERE dre.host_device_id = $1
             AND dre.event_kind = 'host'
             AND dre.applied_to_shadow = TRUE
         ),
         ds.last_report_time
       )
FROM public.device_shadow ds
WHERE ds.device_id = $1
LIMIT 1`, deviceID).Scan(&state.Version, &last)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &state, nil
		}
		return nil, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	if last.Valid {
		t := last.Time
		state.LastReportTime = &t
	}
	return &state, nil
}

func (s *Service) insertReportBatch(ctx context.Context, hostDeviceID int64, sn string, report normalizedReport) (int64, bool, error) {
	var id int64
	var reportedAt interface{}
	if !report.Timestamp.IsZero() {
		reportedAt = report.Timestamp
	}
	err := s.svcCtx.DB.QueryRowContext(ctx, `
INSERT INTO public.device_report_batch (host_device_id, sn, report_id, source, reported_at, is_history, child_count, event_count, payload)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb)
ON CONFLICT (host_device_id, report_id) DO NOTHING
RETURNING id`,
		hostDeviceID,
		strings.ToUpper(strings.TrimSpace(sn)),
		report.ReportID,
		report.Source,
		reportedAt,
		report.IsHistory,
		len(report.Children),
		len(report.Children)+1,
		string(report.Raw),
	).Scan(&id)
	if err == nil {
		return id, true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, false, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	return 0, false, nil
}

func (s *Service) insertReportEvent(ctx context.Context, batchID, hostDeviceID, childID int64, childKey, eventKind string, eventTime time.Time, payload json.RawMessage, applied bool) error {
	var eventAt interface{}
	if !eventTime.IsZero() {
		eventAt = eventTime
	}
	var childArg interface{}
	if childID > 0 {
		childArg = childID
	}
	_, err := s.svcCtx.DB.ExecContext(ctx, `
INSERT INTO public.device_report_event (batch_id, host_device_id, child_id, child_key, event_kind, event_time, payload, applied_to_shadow)
VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8)`,
		batchID, hostDeviceID, childArg, strings.TrimSpace(childKey), eventKind, eventAt, string(payload), applied,
	)
	if err != nil {
		return errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	return nil
}

func (s *Service) ensureChildAndState(ctx context.Context, hostDeviceID int64, child normalizedChild) (int64, *childShadowState, error) {
	var childID int64
	onlineStatus := int16(0)
	if isChildOnline(child.Reported) {
		onlineStatus = 1
	}
	metadataBytes, _ := json.Marshal(child.Metadata)
	err := s.svcCtx.DB.QueryRowContext(ctx, `
INSERT INTO public.device_child (host_device_id, child_key, child_sn, child_type, child_name, online_status, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)
ON CONFLICT (host_device_id, child_key) DO UPDATE
SET child_sn = EXCLUDED.child_sn,
    child_type = EXCLUDED.child_type,
    child_name = EXCLUDED.child_name,
    online_status = EXCLUDED.online_status,
    metadata = EXCLUDED.metadata,
    updated_at = CURRENT_TIMESTAMP
RETURNING id`,
		hostDeviceID, child.ChildKey, child.ChildSN, child.ChildType, child.ChildName, onlineStatus, string(metadataBytes),
	).Scan(&childID)
	if err != nil {
		return 0, nil, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	state, err := s.getChildShadowState(ctx, childID)
	if err != nil {
		return 0, nil, err
	}
	return childID, state, nil
}

func (s *Service) getChildShadowState(ctx context.Context, childID int64) (*childShadowState, error) {
	var state childShadowState
	var last sql.NullTime
	err := s.svcCtx.DB.QueryRowContext(ctx, `
SELECT COALESCE(version, 0), last_report_time
FROM public.device_child_shadow
WHERE child_id = $1
LIMIT 1`, childID).Scan(&state.Version, &last)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &state, nil
		}
		return nil, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	if last.Valid {
		t := last.Time
		state.LastReportTime = &t
	}
	return &state, nil
}

func (s *Service) upsertChildShadow(ctx context.Context, childID, hostDeviceID int64, child normalizedChild, version int64) error {
	reportedBytes, _ := json.Marshal(child.Reported)
	desiredBytes := "{}"
	metadataBytes, _ := json.Marshal(mergeMetadataSection(child.Metadata, child.Reported, child.Timestamp.Unix()))
	_, err := s.svcCtx.DB.ExecContext(ctx, `
INSERT INTO public.device_child_shadow (child_id, host_device_id, child_key, reported, desired, metadata, version, last_report_time)
VALUES ($1, $2, $3, $4::jsonb, $5::jsonb, $6::jsonb, $7, $8)
ON CONFLICT (child_id) DO UPDATE
SET reported = EXCLUDED.reported,
    desired = EXCLUDED.desired,
    metadata = EXCLUDED.metadata,
    version = EXCLUDED.version,
    last_report_time = EXCLUDED.last_report_time,
    updated_at = CURRENT_TIMESTAMP`,
		childID, hostDeviceID, child.ChildKey, string(reportedBytes), desiredBytes, string(metadataBytes), version, child.Timestamp,
	)
	if err != nil {
		return errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	return nil
}

func shouldAdvanceCurrentState(current *time.Time, incoming time.Time) bool {
	if current == nil || current.IsZero() {
		return true
	}
	if incoming.IsZero() {
		return true
	}
	return !incoming.Before(*current)
}

func authRowFromPrincipal(principal *deviceauthsvc.Principal) *repo.DeviceAuthRow {
	if principal == nil {
		return &repo.DeviceAuthRow{}
	}
	return &repo.DeviceAuthRow{
		ID:              principal.DeviceID,
		SN:              principal.DeviceSN,
		Status:          principal.Status,
		ProductKey:      principal.ProductKey,
		Mac:             principal.Mac,
		FirmwareVersion: principal.FirmwareVersion,
		IP:              principal.IP,
	}
}

func fallbackReportID(source, sn string, ts time.Time, index int, raw []byte) string {
	sum := sha1.Sum(append([]byte(fmt.Sprintf("%s|%s|%d|%d|", normalizeSource(source, false), strings.ToUpper(strings.TrimSpace(sn)), ts.UnixMilli(), index)), raw...))
	return normalizeSource(source, false) + "-" + hex.EncodeToString(sum[:8])
}

func normalizeSource(source string, isHistory bool) string {
	source = strings.ToLower(strings.TrimSpace(source))
	switch source {
	case "http", "mqtt", "cache", "history":
		return source
	}
	if isHistory {
		return "history"
	}
	return "http"
}

func normalizeChildKey(childKey, childSN, childType string) string {
	childKey = strings.TrimSpace(childKey)
	if childKey != "" {
		return childKey
	}
	if strings.TrimSpace(childSN) != "" {
		return strings.ToUpper(strings.TrimSpace(childSN))
	}
	if strings.TrimSpace(childType) != "" {
		return strings.ToLower(strings.TrimSpace(childType))
	}
	return ""
}

func fromUnixOrNow(ts int64) time.Time {
	if ts <= 0 {
		return time.Now()
	}
	if ts > 1_000_000_000_000 {
		return time.UnixMilli(ts)
	}
	return time.Unix(ts, 0)
}

func decodeJSONObject(raw []byte) (map[string]interface{}, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return map[string]interface{}{}, nil
	}
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]interface{}{}
	}
	return out, nil
}

func mergeMetadataSection(base map[string]interface{}, reported map[string]interface{}, ts int64) map[string]interface{} {
	if base == nil {
		base = map[string]interface{}{}
	}
	out := map[string]interface{}{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range reported {
		if sub, ok := v.(map[string]interface{}); ok {
			out[k] = mergeMetadataSection(asMap(out[k]), sub, ts)
			continue
		}
		out[k] = map[string]interface{}{"timestamp": ts}
	}
	return out
}

func asMap(v interface{}) map[string]interface{} {
	if m, ok := v.(map[string]interface{}); ok && m != nil {
		return m
	}
	return map[string]interface{}{}
}

func mustJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func isChildOnline(reported map[string]interface{}) bool {
	if v, ok := reported["online"].(bool); ok {
		return v
	}
	if v, ok := reported["status"].(string); ok {
		return strings.EqualFold(strings.TrimSpace(v), "online")
	}
	return false
}

func asString(v interface{}) string {
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func int32Value(v interface{}) (int32, bool) {
	switch n := v.(type) {
	case int32:
		return n, true
	case int:
		return int32(n), true
	case int64:
		return int32(n), true
	case float64:
		return int32(n), true
	case json.Number:
		if x, err := n.Int64(); err == nil {
			return int32(x), true
		}
	case string:
		if x, err := strconv.ParseInt(strings.TrimSpace(n), 10, 32); err == nil {
			return int32(x), true
		}
	}
	return 0, false
}
