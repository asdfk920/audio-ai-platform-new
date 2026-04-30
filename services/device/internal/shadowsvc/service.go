package shadowsvc

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/device/internal/commandsvc"
	redisshadow "github.com/jacklau/audio-ai-platform/services/device/internal/device/shadow"
	"github.com/jacklau/audio-ai-platform/services/device/internal/deviceauthsvc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/repo"
	"github.com/jacklau/audio-ai-platform/services/device/internal/shadowmqtt"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/redis/go-redis/v9"
)

type Service struct {
	svcCtx *svc.ServiceContext
}

type View struct {
	DeviceID        int64
	DeviceSN        string
	Online          bool
	Reported        json.RawMessage
	Desired         json.RawMessage
	Delta           json.RawMessage
	Metadata        json.RawMessage
	Version         int64
	LastReportTime  *time.Time
	InstructionID   int64
	CommandStatus   string
	QueuedCount     int
	ExpiresAt       *time.Time
	InstructionType string
	CommandCode     string
}

type PendingCommand struct {
	InstructionID   int64
	Cmd             string
	CommandCode     string
	InstructionType string
	Params          json.RawMessage
	Status          int16
	Priority        int
	RetryCount      int
	ExpiresAt       *time.Time
	CreatedAt       time.Time
}

// ShadowSyncResult 设备鉴权下一键拉取影子 + 待执行命令（断网重连同步）。
type ShadowSyncResult struct {
	View           *View
	Pending        []PendingCommand
	ServerTimeUnix int64
	VersionStale   bool
}

type CommandResultInput struct {
	DeviceSN      string
	DeviceSecret  string
	InstructionID int64
	Status        int16
	Result        json.RawMessage
	ErrorMsg      string
	Reported      json.RawMessage
}

type DesiredCommandOptions struct {
	InstructionType string
	ScheduleID      *int64
	Operator        string
	Reason          string
	ExpiresAt       *time.Time
}

type shadowRow struct {
	DeviceID       int64
	SN             string
	Reported       []byte
	Desired        []byte
	Metadata       []byte
	Version        int64
	LastReportTime *time.Time
}

type instructionRow struct {
	ID        int64
	DeviceID  int64
	SN        string
	UserID    int64
	Cmd       string
	Params    []byte
	Status    int16
	CreatedAt time.Time
}

func writeShadowDebugLog(runID, hypothesisID, location, message string, data map[string]interface{}) {
	f, err := os.OpenFile("debug-29e955.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	body, _ := json.Marshal(data)
	_, _ = fmt.Fprintf(f, "{\"sessionId\":\"29e955\",\"runId\":%q,\"hypothesisId\":%q,\"location\":%q,\"message\":%q,\"data\":%s,\"timestamp\":%d}\n", runID, hypothesisID, location, message, body, time.Now().UnixMilli())
}

func New(svcCtx *svc.ServiceContext) *Service {
	return &Service{svcCtx: svcCtx}
}

func (s *Service) GetShadowForUser(ctx context.Context, userID int64, sn string) (*View, error) {
	device, err := repo.GetBoundDeviceByUserAndSN(ctx, s.svcCtx.DB, userID, sn)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errorx.NewDefaultError(errorx.CodeDeviceNoPermission)
		}
		return nil, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	if err := repo.ErrIfNotQueryable(device.Status); err != nil {
		return nil, err
	}
	return s.buildView(ctx, device.DeviceID, device.SN, device.OnlineStatus == 1)
}

func (s *Service) UpdateDesiredByUser(ctx context.Context, userID int64, sn string, desiredRaw json.RawMessage, merge bool) (*View, error) {
	return s.UpdateDesiredByUserWithOptions(ctx, userID, sn, desiredRaw, merge, DesiredCommandOptions{})
}

func (s *Service) UpdateDesiredByUserWithOptions(ctx context.Context, userID int64, sn string, desiredRaw json.RawMessage, merge bool, opts DesiredCommandOptions) (*View, error) {
	device, err := repo.GetBoundDeviceByUserAndSN(ctx, s.svcCtx.DB, userID, sn)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errorx.NewDefaultError(errorx.CodeDeviceNoPermission)
		}
		return nil, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	if err := repo.ErrIfNotQueryable(device.Status); err != nil {
		return nil, err
	}

	desiredMap, err := decodeJSONObject(desiredRaw)
	if err != nil {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "desired 必须是合法 JSON 对象")
	}

	row, redisMap, err := s.loadShadowState(ctx, device.DeviceID, device.SN)
	if err != nil {
		return nil, err
	}

	reportedMap := aggregateReported(row, redisMap)
	currentDesired := decodeMapOrEmpty(row.Desired)
	if merge {
		currentDesired = mergeMaps(currentDesired, desiredMap)
	} else {
		currentDesired = desiredMap
	}
	deltaMap := computeJSONDelta(currentDesired, reportedMap)
	metadataMap := decodeMapOrEmpty(row.Metadata)
	metadataMap["desired"] = mergeMetadataSection(asMapOrEmpty(metadataMap["desired"]), desiredMap, time.Now().Unix())
	nextVersion := row.Version + 1
	now := time.Now()

	if err := s.upsertShadow(ctx, device.DeviceID, device.SN, reportedMap, currentDesired, metadataMap, nextVersion, row.LastReportTime, &now); err != nil {
		return nil, err
	}

	var commandResult *commandsvc.CreateImmediateInstructionResult
	if len(deltaMap) > 0 {
		commandResult, err = commandsvc.New(s.svcCtx).CreateImmediateInstructionFromDesired(ctx, commandsvc.CreateImmediateInstructionInput{
			DeviceID:        device.DeviceID,
			DeviceSN:        device.SN,
			UserID:          userID,
			CommandCode:     commandsvc.CommandCodeShadowSync,
			InstructionType: normalizedInstructionType(opts.InstructionType),
			Params:          deltaMap,
			Operator:        desiredOperator(userID, opts.Operator),
			Reason:          desiredReason(opts.Reason),
			ExpiresAt:       opts.ExpiresAt,
			ScheduleID:      opts.ScheduleID,
		})
		if err != nil {
			return nil, err
		}
	}

	if err := s.syncRedisShadow(ctx, device.DeviceID, device.SN, redisMap, reportedMap, currentDesired, deltaMap, metadataMap, nextVersion, nil); err != nil {
		return nil, err
	}
	if len(deltaMap) > 0 {
		_ = shadowmqtt.PublishJSONDelta(s.svcCtx.Config, s.svcCtx.MQTTClient(), strings.ToUpper(strings.TrimSpace(device.SN)), device.DeviceID, deltaMap)
	}
	view, err := s.buildView(ctx, device.DeviceID, device.SN, device.OnlineStatus == 1)
	if err != nil {
		return nil, err
	}
	if commandResult != nil {
		view.InstructionID = commandResult.InstructionID
		view.CommandStatus = commandResult.Status
		view.QueuedCount = commandResult.QueuedCount
		view.ExpiresAt = commandResult.ExpiresAt
		view.InstructionType = commandResult.InstructionType
		view.CommandCode = commandResult.CommandCode
	} else {
		view.CommandStatus = "noop"
	}
	return view, nil
}

func (s *Service) UpdateReportedForUser(ctx context.Context, userID int64, sn string, reportedRaw json.RawMessage) (*View, error) {
	device, err := repo.GetBoundDeviceByUserAndSN(ctx, s.svcCtx.DB, userID, sn)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errorx.NewDefaultError(errorx.CodeDeviceNoPermission)
		}
		return nil, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	if err := repo.ErrIfNotQueryable(device.Status); err != nil {
		return nil, err
	}
	return s.updateReportedByDeviceID(ctx, device.DeviceID, device.SN, reportedRaw, "app", "")
}

func (s *Service) UpdateReportedForDevice(ctx context.Context, sn, deviceSecret string, reportedRaw json.RawMessage, clientIP string) (*View, error) {
	principal, err := deviceauthsvc.New(s.svcCtx).AuthenticateRequest(ctx, sn, deviceSecret, clientIP)
	if err != nil {
		return nil, err
	}
	return s.updateReportedByDeviceID(ctx, principal.DeviceID, principal.DeviceSN, reportedRaw, "device", clientIP)
}

func (s *Service) UpdateReportedForAuthenticatedDevice(ctx context.Context, deviceID int64, sn string, reportedRaw json.RawMessage, clientIP string) (*View, error) {
	return s.updateReportedByDeviceID(ctx, deviceID, sn, reportedRaw, "device", clientIP)
}

func (s *Service) GetPendingCommandsForDevice(ctx context.Context, sn, deviceSecret string, limit int) ([]PendingCommand, error) {
	// #region agent log
	writeShadowDebugLog("pre-fix", "H2", "services/device/internal/shadowsvc/service.go:193", "GetPendingCommandsForDevice entered", map[string]interface{}{
		"sn":              strings.ToUpper(strings.TrimSpace(sn)),
		"deviceSecretLen": len(strings.TrimSpace(deviceSecret)),
		"limit":           limit,
	})
	// #endregion
	principal, err := deviceauthsvc.New(s.svcCtx).AuthenticateRequest(ctx, sn, deviceSecret, "")
	if err != nil {
		writeShadowDebugLog("pre-fix", "H3", "services/device/internal/shadowsvc/service.go:202", "GetDeviceForMQTTAuth failed", map[string]interface{}{
			"sn":  strings.ToUpper(strings.TrimSpace(sn)),
			"err": err.Error(),
		})
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	list, err := commandsvc.New(s.svcCtx).ListPendingForDevice(ctx, principal.DeviceID, limit)
	if err != nil {
		return nil, err
	}
	commands := make([]PendingCommand, 0, len(list))
	for _, item := range list {
		commands = append(commands, PendingCommand{
			InstructionID:   item.InstructionID,
			Cmd:             item.Cmd,
			CommandCode:     item.CommandCode,
			InstructionType: item.InstructionType,
			Params:          item.Params,
			Status:          item.Status,
			Priority:        item.Priority,
			RetryCount:      item.RetryCount,
			ExpiresAt:       item.ExpiresAt,
			CreatedAt:       item.CreatedAt,
		})
	}
	// #region agent log
	writeShadowDebugLog("pre-fix", "H4", "services/device/internal/shadowsvc/service.go:246", "pending query completed", map[string]interface{}{
		"deviceId": principal.DeviceID,
		"sn":       principal.DeviceSN,
		"count":    len(commands),
	})
	// #endregion
	return commands, nil
}

func (s *Service) GetPendingCommandsForAuthenticatedDevice(ctx context.Context, deviceID int64, sn string, limit int) ([]PendingCommand, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	list, err := commandsvc.New(s.svcCtx).ListPendingForDevice(ctx, deviceID, limit)
	if err != nil {
		return nil, err
	}
	commands := make([]PendingCommand, 0, len(list))
	for _, item := range list {
		commands = append(commands, PendingCommand{
			InstructionID:   item.InstructionID,
			Cmd:             item.Cmd,
			CommandCode:     item.CommandCode,
			InstructionType: item.InstructionType,
			Params:          item.Params,
			Status:          item.Status,
			Priority:        item.Priority,
			RetryCount:      item.RetryCount,
			ExpiresAt:       item.ExpiresAt,
			CreatedAt:       item.CreatedAt,
		})
	}
	return commands, nil
}

func (s *Service) ReportCommandResult(ctx context.Context, in CommandResultInput) (*View, error) {
	principal, err := deviceauthsvc.New(s.svcCtx).AuthenticateRequest(ctx, in.DeviceSN, in.DeviceSecret, "")
	if err != nil {
		return nil, err
	}

	if err := commandsvc.New(s.svcCtx).MarkInstructionResult(
		ctx, principal.DeviceID, in.InstructionID, in.Status, in.Result, in.ErrorMsg, "device:"+strings.ToUpper(strings.TrimSpace(principal.DeviceSN)),
	); err != nil {
		return nil, err
	}

	if len(bytes.TrimSpace(in.Reported)) > 0 {
		view, err := s.updateReportedByDeviceID(ctx, principal.DeviceID, principal.DeviceSN, in.Reported, "device_result", "")
		if err != nil {
			return nil, err
		}
		_, _ = commandsvc.New(s.svcCtx).DispatchPendingInstructions(ctx, principal.DeviceID, principal.DeviceSN)
		return view, nil
	}

	_, _ = commandsvc.New(s.svcCtx).DispatchPendingInstructions(ctx, principal.DeviceID, principal.DeviceSN)
	return s.buildView(ctx, principal.DeviceID, principal.DeviceSN, true)
}

func (s *Service) ReportCommandResultForAuthenticatedDevice(ctx context.Context, deviceID int64, sn string, in CommandResultInput) (*View, error) {
	if err := commandsvc.New(s.svcCtx).MarkInstructionResult(
		ctx, deviceID, in.InstructionID, in.Status, in.Result, in.ErrorMsg, "device:"+strings.ToUpper(strings.TrimSpace(sn)),
	); err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(in.Reported)) > 0 {
		view, err := s.updateReportedByDeviceID(ctx, deviceID, sn, in.Reported, "device_result", "")
		if err != nil {
			return nil, err
		}
		_, _ = commandsvc.New(s.svcCtx).DispatchPendingInstructions(ctx, deviceID, sn)
		return view, nil
	}
	_, _ = commandsvc.New(s.svcCtx).DispatchPendingInstructions(ctx, deviceID, sn)
	return s.buildView(ctx, deviceID, sn, true)
}

func (s *Service) PushPendingForDevice(ctx context.Context, sn string) {
	row, err := repo.GetDeviceForMQTTAuth(ctx, s.svcCtx.DB, sn)
	if err != nil {
		return
	}
	_, _ = commandsvc.New(s.svcCtx).DispatchPendingInstructions(ctx, row.ID, row.SN)
}

func (s *Service) updateReportedByDeviceID(ctx context.Context, deviceID int64, sn string, reportedRaw json.RawMessage, source string, clientIP string) (*View, error) {
	reportedPatch, err := decodeJSONObject(reportedRaw)
	if err != nil {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "reported 必须是合法 JSON 对象")
	}

	row, redisMap, err := s.loadShadowState(ctx, deviceID, sn)
	if err != nil {
		return nil, err
	}
	currentReported := aggregateReported(row, redisMap)
	currentDesired := decodeMapOrEmpty(row.Desired)
	nextReported := mergeMaps(currentReported, reportedPatch)
	deltaMap := computeJSONDelta(currentDesired, nextReported)
	metadataMap := decodeMapOrEmpty(row.Metadata)
	metadataMap["reported"] = mergeMetadataSection(asMapOrEmpty(metadataMap["reported"]), reportedPatch, time.Now().Unix())
	nextVersion := row.Version + 1
	now := time.Now()

	if err := s.upsertShadow(ctx, deviceID, sn, nextReported, currentDesired, metadataMap, nextVersion, &now, row.LastReportTime); err != nil {
		return nil, err
	}
	online := detectOnlineOverride(reportedPatch)
	if online == nil {
		inferred := true
		online = &inferred
		nextReported["online"] = true
	}
	if err := s.syncRedisShadow(ctx, deviceID, sn, redisMap, nextReported, currentDesired, deltaMap, metadataMap, nextVersion, online); err != nil {
		return nil, err
	}
	if err := s.persistSnapshot(ctx, deviceID, sn, nextReported, *online, source, clientIP); err != nil {
		return nil, err
	}
	return s.buildView(ctx, deviceID, sn, online != nil && *online)
}

func (s *Service) buildView(ctx context.Context, deviceID int64, sn string, defaultOnline bool) (*View, error) {
	row, redisMap, err := s.loadShadowState(ctx, deviceID, sn)
	if err != nil {
		return nil, err
	}
	reportedMap := aggregateReported(row, redisMap)
	desiredMap := decodeMapOrEmpty(row.Desired)
	deltaMap := computeJSONDelta(desiredMap, reportedMap)
	metadataMap := decodeMapOrEmpty(row.Metadata)

	reportedBytes, _ := json.Marshal(reportedMap)
	desiredBytes, _ := json.Marshal(desiredMap)
	deltaBytes, _ := json.Marshal(deltaMap)
	metadataBytes, _ := json.Marshal(metadataMap)

	return &View{
		DeviceID:       deviceID,
		DeviceSN:       strings.ToUpper(strings.TrimSpace(sn)),
		Online:         resolveOnline(redisMap, defaultOnline),
		Reported:       reportedBytes,
		Desired:        desiredBytes,
		Delta:          deltaBytes,
		Metadata:       metadataBytes,
		Version:        row.Version,
		LastReportTime: row.LastReportTime,
	}, nil
}

func (s *Service) loadShadowState(ctx context.Context, deviceID int64, sn string) (*shadowRow, map[string]string, error) {
	row, err := s.getShadowRow(ctx, deviceID)
	if err != nil {
		return nil, nil, err
	}
	if row == nil {
		row = &shadowRow{DeviceID: deviceID, SN: strings.ToUpper(strings.TrimSpace(sn))}
	}

	redisMap := map[string]string{}
	if s.svcCtx.Redis != nil {
		redisMap, err = s.svcCtx.Redis.HGetAll(ctx, redisshadow.ShadowKey(sn)).Result()
		if err != nil && err != redis.Nil {
			return nil, nil, errorx.NewDefaultError(errorx.CodeRedisError)
		}
	}
	return row, redisMap, nil
}

func (s *Service) getShadowRow(ctx context.Context, deviceID int64) (*shadowRow, error) {
	if deviceID <= 0 {
		return nil, errorx.NewDefaultError(errorx.CodeInvalidParam)
	}
	var row shadowRow
	var reported, desired, metadata []byte
	var last sql.NullTime
	err := s.svcCtx.DB.QueryRowContext(ctx, `
SELECT device_id, sn, COALESCE(reported, '{}'::jsonb)::text, COALESCE(desired, '{}'::jsonb)::text,
       COALESCE(metadata, '{}'::jsonb)::text, COALESCE(version, 0), last_report_time
FROM public.device_shadow
WHERE device_id = $1
LIMIT 1`, deviceID).Scan(&row.DeviceID, &row.SN, &reported, &desired, &metadata, &row.Version, &last)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	row.Reported = reported
	row.Desired = desired
	row.Metadata = metadata
	if last.Valid {
		t := last.Time
		row.LastReportTime = &t
	}
	return &row, nil
}

func (s *Service) upsertShadow(ctx context.Context, deviceID int64, sn string, reported, desired, metadata map[string]interface{}, version int64, lastReportTime *time.Time, fallbackReportTime *time.Time) error {
	reportedBytes, _ := json.Marshal(reported)
	desiredBytes, _ := json.Marshal(desired)
	metadataBytes, _ := json.Marshal(metadata)
	var reportArg interface{}
	if lastReportTime != nil {
		reportArg = *lastReportTime
	} else if fallbackReportTime != nil {
		reportArg = *fallbackReportTime
	} else {
		reportArg = nil
	}
	_, err := s.svcCtx.DB.ExecContext(ctx, `
INSERT INTO public.device_shadow (device_id, sn, reported, desired, metadata, version, last_report_time)
VALUES ($1, $2, $3::jsonb, $4::jsonb, $5::jsonb, $6, $7)
ON CONFLICT (device_id) DO UPDATE
SET sn = EXCLUDED.sn,
    reported = EXCLUDED.reported,
    desired = EXCLUDED.desired,
    metadata = EXCLUDED.metadata,
    version = EXCLUDED.version,
    last_report_time = COALESCE(EXCLUDED.last_report_time, public.device_shadow.last_report_time),
    updated_at = CURRENT_TIMESTAMP
`, deviceID, strings.ToUpper(strings.TrimSpace(sn)), string(reportedBytes), string(desiredBytes), string(metadataBytes), version, reportArg)
	if err != nil {
		return errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	return nil
}

func (s *Service) syncRedisShadow(ctx context.Context, deviceID int64, sn string, prev map[string]string, reported, desired, delta, metadata map[string]interface{}, version int64, onlineOverride *bool) error {
	if s.svcCtx.Redis == nil {
		return nil
	}
	ttlSec := s.svcCtx.Config.DeviceShadow.HeartbeatTTLSeconds
	if ttlSec <= 0 {
		ttlSec = 60
	}
	ttl := time.Duration(ttlSec) * time.Second
	fields := make(map[string]interface{})
	for k, v := range prev {
		fields[k] = v
	}
	fields[redisshadow.FSN] = strings.ToUpper(strings.TrimSpace(sn))
	fields[redisshadow.FDeviceID] = strconv.FormatInt(deviceID, 10)
	fields[redisshadow.FReportedJSON] = string(mustJSON(reported))
	fields[redisshadow.FDesiredJSON] = string(mustJSON(desired))
	fields[redisshadow.FDeltaJSON] = string(mustJSON(delta))
	fields[redisshadow.FMetadataJSON] = string(mustJSON(metadata))
	fields[redisshadow.FVersion] = strconv.FormatInt(version, 10)

	applyScalarFields(fields, reported)
	nowMs := strconv.FormatInt(time.Now().UnixMilli(), 10)
	fields[redisshadow.FUpdatedMs] = nowMs
	if onlineOverride != nil {
		if *onlineOverride {
			fields[redisshadow.FOnline] = "1"
		} else {
			fields[redisshadow.FOnline] = "0"
		}
		fields[redisshadow.FLastActiveMs] = nowMs
	}

	pipe := s.svcCtx.Redis.Pipeline()
	sk := redisshadow.ShadowKey(sn)
	ok := redisshadow.OnlineKey(sn)
	pipe.HSet(ctx, sk, fields)
	pipe.Expire(ctx, sk, ttl)
	if onlineOverride != nil {
		if *onlineOverride {
			pipe.Set(ctx, ok, "1", ttl)
		} else {
			pipe.Set(ctx, ok, "0", ttl)
		}
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		return errorx.NewDefaultError(errorx.CodeRedisError)
	}
	return nil
}

func (s *Service) persistSnapshot(ctx context.Context, deviceID int64, sn string, reported map[string]interface{}, online bool, source string, clientIP string) error {
	runState, _ := reported["run_state"].(string)
	fw, _ := reported["firmware_version"].(string)
	battery := int32FromAny(reported["battery"])
	onlineStatus := int16(0)
	if online {
		onlineStatus = 1
	}
	lastActive := time.Now()
	if err := repo.InsertDeviceStatusRow(ctx, s.svcCtx.DB, deviceID, sn, runState, battery, fw, onlineStatus, lastActive, source); err != nil {
		return errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	ip := clientIP
	if ip == "" {
		if v, ok := reported["ip"].(string); ok {
			ip = v
		}
	}
	if err := repo.UpdateDeviceOnlineMeta(ctx, s.svcCtx.DB, deviceID, onlineStatus, lastActive, fw, ip); err != nil {
		return errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	return nil
}

func (s *Service) insertInstruction(ctx context.Context, deviceID int64, sn string, userID int64, cmd string, params map[string]interface{}, operator, reason string) (int64, error) {
	var id int64
	paramsBytes, _ := json.Marshal(params)
	err := s.svcCtx.DB.QueryRowContext(ctx, `
INSERT INTO public.device_instruction (device_id, sn, user_id, cmd, params, status, operator, reason)
VALUES ($1, $2, $3, $4, $5::jsonb, 1, $6, $7)
RETURNING id`, deviceID, strings.ToUpper(strings.TrimSpace(sn)), userID, cmd, string(paramsBytes), operator, reason).Scan(&id)
	if err != nil {
		return 0, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	_ = s.insertInstructionStateLog(ctx, id, nil, 1, "created", operator)
	return id, nil
}

func (s *Service) getInstructionStatus(ctx context.Context, deviceID, instructionID int64) (*int16, error) {
	var status int16
	err := s.svcCtx.DB.QueryRowContext(ctx, `
SELECT status
FROM public.device_instruction
WHERE id = $1 AND device_id = $2
LIMIT 1`, instructionID, deviceID).Scan(&status)
	if err == sql.ErrNoRows {
		return nil, errorx.NewDefaultError(errorx.CodeDeviceCommandNotFound)
	}
	if err != nil {
		return nil, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	return &status, nil
}

func (s *Service) updateInstructionResult(ctx context.Context, deviceID int64, in CommandResultInput, previous *int16) error {
	now := time.Now()
	resultText := string(bytes.TrimSpace(in.Result))
	if resultText == "" {
		resultText = "{}"
	}
	_, err := s.svcCtx.DB.ExecContext(ctx, `
UPDATE public.device_instruction
SET status = $1,
    result = $2::jsonb,
    error_msg = $3,
    received_at = COALESCE(received_at, $4),
    completed_at = CASE WHEN $1 IN (3,4,5,6) THEN $4 ELSE completed_at END,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $5 AND device_id = $6`,
		in.Status, resultText, strings.TrimSpace(in.ErrorMsg), now, in.InstructionID, deviceID)
	if err != nil {
		return errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	_ = s.insertInstructionStateLog(ctx, in.InstructionID, previous, in.Status, "device_result", "device:"+strings.ToUpper(strings.TrimSpace(in.DeviceSN)))
	return nil
}

func (s *Service) insertInstructionStateLog(ctx context.Context, instructionID int64, from *int16, to int16, note, operator string) error {
	var fromVal interface{}
	if from != nil {
		fromVal = *from
	}
	_, err := s.svcCtx.DB.ExecContext(ctx, `
INSERT INTO public.device_instruction_state_log (instruction_id, from_status, to_status, note, operator)
VALUES ($1, $2, $3, $4, $5)`, instructionID, fromVal, to, note, operator)
	return err
}

func (s *Service) pushPendingCommandsIfOnline(ctx context.Context, deviceID int64, sn string) (int, error) {
	if s.svcCtx.MQTTClient() == nil {
		return 0, nil
	}
	if !s.isDeviceOnline(ctx, sn) {
		return 0, nil
	}
	rows, err := s.svcCtx.DB.QueryContext(ctx, `
SELECT id, device_id, sn, user_id, cmd, params, status, created_at
FROM public.device_instruction
WHERE device_id = $1 AND status = 1
ORDER BY created_at ASC
LIMIT 20`, deviceID)
	if err != nil {
		return 0, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	defer rows.Close()

	pushed := 0
	for rows.Next() {
		var item instructionRow
		if err := rows.Scan(&item.ID, &item.DeviceID, &item.SN, &item.UserID, &item.Cmd, &item.Params, &item.Status, &item.CreatedAt); err != nil {
			return pushed, errorx.NewDefaultError(errorx.CodeDatabaseError)
		}
		payload, _ := json.Marshal(map[string]interface{}{
			"type":           "shadow_delta",
			"instruction_id": item.ID,
			"sn":             item.SN,
			"cmd":            item.Cmd,
			"params":         decodeMapOrEmpty(item.Params),
		})
		if err := shadowmqtt.PublishDesiredCommand(s.svcCtx.Config, s.svcCtx.MQTTClient(), item.SN, item.DeviceID, payload); err != nil {
			return pushed, nil
		}
		now := time.Now()
		if _, err := s.svcCtx.DB.ExecContext(ctx, `
UPDATE public.device_instruction
SET status = 2,
    received_at = COALESCE(received_at, $1),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $2 AND status = 1`, now, item.ID); err == nil {
			prev := int16(1)
			_ = s.insertInstructionStateLog(ctx, item.ID, &prev, 2, "mqtt_pushed", "system")
		}
		pushed++
	}
	return pushed, rows.Err()
}

func (s *Service) isDeviceOnline(ctx context.Context, sn string) bool {
	if s.svcCtx.Redis == nil {
		return false
	}
	val, err := s.svcCtx.Redis.Get(ctx, redisshadow.OnlineKey(sn)).Result()
	if err != nil {
		return false
	}
	return strings.TrimSpace(val) == "1"
}

func aggregateReported(row *shadowRow, redisMap map[string]string) map[string]interface{} {
	reported := decodeMapOrEmpty(row.Reported)
	if len(redisMap) == 0 {
		return reported
	}
	if v := strings.TrimSpace(redisMap[redisshadow.FReportedJSON]); v != "" {
		reported = mergeMaps(reported, decodeMapOrEmpty([]byte(v)))
	}
	if v := strings.TrimSpace(redisMap[redisshadow.FFirmwareVersion]); v != "" {
		reported["firmware_version"] = v
	}
	if v := strings.TrimSpace(redisMap[redisshadow.FRunState]); v != "" {
		reported["run_state"] = v
	}
	if v := strings.TrimSpace(redisMap[redisshadow.FIP]); v != "" {
		reported["ip"] = v
	}
	if v := strings.TrimSpace(redisMap[redisshadow.FMac]); v != "" {
		reported["mac"] = v
	}
	if v := strings.TrimSpace(redisMap[redisshadow.FProductKey]); v != "" {
		reported["product_key"] = v
	}
	if v := strings.TrimSpace(redisMap[redisshadow.FBattery]); v != "" {
		if n, err := strconv.ParseInt(v, 10, 32); err == nil {
			reported["battery"] = int32(n)
		}
	}
	if v := strings.TrimSpace(redisMap[redisshadow.FOnline]); v != "" {
		reported["online"] = v == "1"
	}
	return reported
}

func computeJSONDelta(desired, reported map[string]interface{}) map[string]interface{} {
	if len(desired) == 0 {
		return map[string]interface{}{}
	}
	delta := map[string]interface{}{}
	for k, dv := range desired {
		rv, ok := reported[k]
		if !ok {
			delta[k] = dv
			continue
		}
		db, _ := json.Marshal(dv)
		rb, _ := json.Marshal(rv)
		if !bytes.Equal(db, rb) {
			delta[k] = dv
		}
	}
	return delta
}

// ComputeJSONDelta 计算 desired 相对 reported 的差分。
func ComputeJSONDelta(desired, reported map[string]interface{}) map[string]interface{} {
	return computeJSONDelta(desired, reported)
}

// GetShadowSyncForDevice 设备凭 SN+Secret 聚合影子视图与待处理指令（HTTP 重连同步）。
func (s *Service) GetShadowSyncForDevice(ctx context.Context, sn, deviceSecret string, clientVersion int64, limit int) (*ShadowSyncResult, error) {
	principal, err := deviceauthsvc.New(s.svcCtx).AuthenticateRequest(ctx, sn, deviceSecret, "")
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	view, err := s.buildView(ctx, principal.DeviceID, principal.DeviceSN, true)
	if err != nil {
		return nil, err
	}
	list, err := commandsvc.New(s.svcCtx).ListPendingForDevice(ctx, principal.DeviceID, limit)
	if err != nil {
		return nil, err
	}
	pending := make([]PendingCommand, 0, len(list))
	for _, item := range list {
		pending = append(pending, PendingCommand{
			InstructionID:   item.InstructionID,
			Cmd:             item.Cmd,
			CommandCode:     item.CommandCode,
			InstructionType: item.InstructionType,
			Params:          item.Params,
			Status:          item.Status,
			Priority:        item.Priority,
			RetryCount:      item.RetryCount,
			ExpiresAt:       item.ExpiresAt,
			CreatedAt:       item.CreatedAt,
		})
	}
	stale := clientVersion > 0 && clientVersion < view.Version
	return &ShadowSyncResult{
		View:           view,
		Pending:        pending,
		ServerTimeUnix: time.Now().Unix(),
		VersionStale:   stale,
	}, nil
}

func mergeMaps(dst, patch map[string]interface{}) map[string]interface{} {
	if dst == nil {
		dst = map[string]interface{}{}
	}
	out := map[string]interface{}{}
	for k, v := range dst {
		out[k] = v
	}
	for k, v := range patch {
		if existing, ok := out[k].(map[string]interface{}); ok {
			if next, ok := asMap(v); ok {
				out[k] = mergeMaps(existing, next)
				continue
			}
		}
		out[k] = v
	}
	return out
}

func normalizedInstructionType(v string) string {
	switch strings.TrimSpace(v) {
	case commandsvc.InstructionTypeScheduled:
		return commandsvc.InstructionTypeScheduled
	default:
		return commandsvc.InstructionTypeManual
	}
}

func desiredOperator(userID int64, operator string) string {
	if v := strings.TrimSpace(operator); v != "" {
		return v
	}
	return fmt.Sprintf("user:%d", userID)
}

func desiredReason(reason string) string {
	if v := strings.TrimSpace(reason); v != "" {
		return v
	}
	return "desired_update"
}

func mergeMetadataSection(existing map[string]interface{}, patch map[string]interface{}, ts int64) map[string]interface{} {
	if existing == nil {
		existing = map[string]interface{}{}
	}
	out := map[string]interface{}{}
	for k, v := range existing {
		out[k] = v
	}
	for k, v := range patch {
		if sub, ok := asMap(v); ok {
			out[k] = mergeMetadataSection(asMapOrEmpty(out[k]), sub, ts)
			continue
		}
		out[k] = map[string]interface{}{"timestamp": ts}
	}
	return out
}

func applyScalarFields(fields map[string]interface{}, reported map[string]interface{}) {
	if online, ok := detectBool(reported["online"]); ok {
		if online {
			fields[redisshadow.FOnline] = "1"
		} else {
			fields[redisshadow.FOnline] = "0"
		}
	}
	if v, ok := reported["firmware_version"].(string); ok && strings.TrimSpace(v) != "" {
		fields[redisshadow.FFirmwareVersion] = strings.TrimSpace(v)
	}
	if v, ok := reported["run_state"].(string); ok && strings.TrimSpace(v) != "" {
		fields[redisshadow.FRunState] = strings.TrimSpace(v)
	}
	if v, ok := reported["ip"].(string); ok && strings.TrimSpace(v) != "" {
		fields[redisshadow.FIP] = strings.TrimSpace(v)
	}
	if v, ok := reported["mac"].(string); ok && strings.TrimSpace(v) != "" {
		fields[redisshadow.FMac] = strings.TrimSpace(v)
	}
	if v, ok := reported["product_key"].(string); ok && strings.TrimSpace(v) != "" {
		fields[redisshadow.FProductKey] = strings.TrimSpace(v)
	}
	if _, ok := reported["battery"]; ok {
		fields[redisshadow.FBattery] = strconv.FormatInt(int64(int32FromAny(reported["battery"])), 10)
	}
}

func resolveOnline(redisMap map[string]string, defaultOnline bool) bool {
	if len(redisMap) == 0 {
		return defaultOnline
	}
	v := strings.TrimSpace(redisMap[redisshadow.FOnline])
	if v == "1" {
		return true
	}
	if v == "0" {
		return false
	}
	return defaultOnline
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

func decodeMapOrEmpty(raw []byte) map[string]interface{} {
	m, err := decodeJSONObject(raw)
	if err != nil {
		return map[string]interface{}{}
	}
	return m
}

func asMap(v interface{}) (map[string]interface{}, bool) {
	m, ok := v.(map[string]interface{})
	return m, ok
}

func asMapOrEmpty(v interface{}) map[string]interface{} {
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	return map[string]interface{}{}
}

func mustJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}

func detectBool(v interface{}) (bool, bool) {
	b, ok := v.(bool)
	return b, ok
}

func detectOnlineOverride(reported map[string]interface{}) *bool {
	if online, ok := detectBool(reported["online"]); ok {
		return &online
	}
	return nil
}

func int32FromAny(v interface{}) int32 {
	switch n := v.(type) {
	case int32:
		return n
	case int:
		return int32(n)
	case int64:
		return int32(n)
	case float64:
		return int32(n)
	case json.Number:
		iv, _ := n.Int64()
		return int32(iv)
	default:
		return 0
	}
}
