package commandsvc

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	redisshadow "github.com/jacklau/audio-ai-platform/services/device/internal/device/shadow"
	"github.com/jacklau/audio-ai-platform/services/device/internal/shadowmqtt"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/robfig/cron/v3"
	"github.com/zeromicro/go-zero/core/logx"
)

// logDBErr logs the underlying SQL/driver error (otherwise only 9001 is visible) and returns a CodeError.
func logDBErr(op string, err error) error {
	if err == nil {
		return nil
	}
	logx.Errorf("commandsvc %s: %v", op, err)
	return errorx.NewDefaultError(errorx.CodeDatabaseError)
}

const (
	CommandCodeShadowSync = "shadow_sync"

	InstructionTypeManual    = "manual"
	InstructionTypeScheduled = "scheduled"

	StatusPending   int16 = 1
	StatusExecuting int16 = 2
	StatusSuccess   int16 = 3
	StatusFailed    int16 = 4
	StatusTimeout   int16 = 5
	StatusCancelled int16 = 6
)

type Service struct {
	svcCtx *svc.ServiceContext
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

type CreateImmediateInstructionInput struct {
	DeviceID        int64
	DeviceSN        string
	UserID          int64
	CommandCode     string
	InstructionType string
	Params          map[string]interface{}
	Operator        string
	Reason          string
	ExpiresAt       *time.Time
	Priority        int
	MaxRetry        int
	TimeoutSeconds  int
	ScheduleID      *int64
}

type CreateImmediateInstructionResult struct {
	InstructionID   int64
	Status          string
	QueuedCount     int
	ExpiresAt       *time.Time
	InstructionType string
	CommandCode     string
}

type InstructionHistoryFilter struct {
	UserID        int64
	DeviceSN      string
	Status        string
	Page          int
	PageSize      int
	InstructionID int64
}

type InstructionHistoryItem struct {
	InstructionID   int64           `json:"instruction_id"`
	DeviceID        int64           `json:"device_id"`
	DeviceSN        string          `json:"device_sn"`
	Cmd             string          `json:"cmd"`
	CommandCode     string          `json:"command_code"`
	InstructionType string          `json:"instruction_type"`
	Status          string          `json:"status"`
	StatusCode      int16           `json:"status_code"`
	Priority        int             `json:"priority"`
	RetryCount      int             `json:"retry_count"`
	ExpiresAt       *int64          `json:"expires_at,optional"`
	Params          json.RawMessage `json:"params"`
	Result          json.RawMessage `json:"result,optional"`
	ErrorMsg        string          `json:"error_msg,optional"`
	CreatedAt       int64           `json:"created_at"`
	UpdatedAt       int64           `json:"updated_at"`
}

type InstructionStateLogItem struct {
	FromStatus *int16 `json:"from_status,optional"`
	ToStatus   int16  `json:"to_status"`
	Note       string `json:"note"`
	Operator   string `json:"operator"`
	CreatedAt  int64  `json:"created_at"`
}

type InstructionDetail struct {
	Instruction InstructionHistoryItem    `json:"instruction"`
	Logs        []InstructionStateLogItem `json:"logs"`
}

type Schedule struct {
	ID             int64
	DeviceID       int64
	DeviceSN       string
	UserID         int64
	ScheduleType   string
	DesiredPayload json.RawMessage
	CommandPayload json.RawMessage
	MergeDesired   bool
	CronExpr       string
	Timezone       string
	NextExecuteAt  *time.Time
	LastExecuteAt  *time.Time
	Status         string
	ExpiresAt      *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ScheduleUpsertInput struct {
	ID             int64
	DeviceID       int64
	DeviceSN       string
	UserID         int64
	ScheduleType   string
	DesiredPayload json.RawMessage
	MergeDesired   bool
	CronExpr       string
	Timezone       string
	ExecuteAt      *time.Time
	ExpiresAt      *time.Time
}

type ScheduleListFilter struct {
	UserID   int64
	DeviceSN string
	Status   string
	Page     int
	PageSize int
}

func New(svcCtx *svc.ServiceContext) *Service {
	return &Service{svcCtx: svcCtx}
}

func (s *Service) CreateImmediateInstructionFromDesired(ctx context.Context, in CreateImmediateInstructionInput) (*CreateImmediateInstructionResult, error) {
	if s == nil || s.svcCtx == nil || s.svcCtx.DB == nil {
		return nil, errorx.NewDefaultError(errorx.CodeSystemError)
	}
	commandCode := strings.TrimSpace(in.CommandCode)
	if commandCode == "" {
		commandCode = CommandCodeShadowSync
	}
	instructionType := strings.TrimSpace(in.InstructionType)
	if instructionType == "" {
		instructionType = InstructionTypeManual
	}
	if in.Priority <= 0 {
		in.Priority = 100
	}
	if in.MaxRetry <= 0 {
		in.MaxRetry = s.defaultMaxRetry()
	}
	if in.TimeoutSeconds <= 0 {
		in.TimeoutSeconds = s.defaultTimeoutSeconds()
	}
	if in.ExpiresAt == nil {
		expires := time.Now().Add(time.Duration(s.defaultExpiresSeconds()) * time.Second)
		in.ExpiresAt = &expires
	}
	operator := strings.TrimSpace(in.Operator)
	if operator == "" {
		operator = "system"
	}

	tx, err := s.svcCtx.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	defer func() { _ = tx.Rollback() }()

	mergedCount, err := s.mergeConflictingPendingInstructionsTx(ctx, tx, in.DeviceID, commandCode, instructionType, operator)
	if err != nil {
		return nil, err
	}

	var instructionID int64
	paramsBytes, _ := json.Marshal(in.Params)
	err = tx.QueryRowContext(ctx, `
INSERT INTO public.device_instruction (
	device_id, sn, user_id, cmd, command_code, instruction_type, params, status,
	operator, reason, priority, expires_at, max_retry, timeout_seconds, merged_from_count, schedule_id
) VALUES (
	$1, $2, $3, $4, $5, $6, $7::jsonb, $8,
	$9, $10, $11, $12, $13, $14, $15, $16
)
RETURNING id`,
		in.DeviceID, strings.ToUpper(strings.TrimSpace(in.DeviceSN)), in.UserID,
		commandCode, commandCode, instructionType, string(paramsBytes), StatusPending,
		operator, strings.TrimSpace(in.Reason), in.Priority, in.ExpiresAt, in.MaxRetry, in.TimeoutSeconds, mergedCount, in.ScheduleID,
	).Scan(&instructionID)
	if err != nil {
		return nil, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	if err := insertStateLogTx(ctx, tx, instructionID, nil, StatusPending, "created", operator); err != nil {
		return nil, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}

	cachedNote := "cached_pending"
	if s.isDeviceOnline(ctx, in.DeviceSN) && s.svcCtx.MQTTClient() != nil {
		cachedNote = "ready_for_dispatch"
	} else {
		cachedNote = "cached_offline"
	}
	if err := insertStateLogTx(ctx, tx, instructionID, &[]int16{StatusPending}[0], StatusPending, cachedNote, operator); err != nil {
		return nil, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	if err := tx.Commit(); err != nil {
		return nil, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}

	result := &CreateImmediateInstructionResult{
		InstructionID:   instructionID,
		Status:          "cached",
		QueuedCount:     s.countPendingForDevice(ctx, in.DeviceID),
		ExpiresAt:       in.ExpiresAt,
		InstructionType: instructionType,
		CommandCode:     commandCode,
	}
	if s.isDeviceOnline(ctx, in.DeviceSN) {
		result.Status = "delivered"
		if s.svcCtx.MQTTClient() != nil {
			if pushed, _ := s.DispatchPendingInstructions(ctx, in.DeviceID, in.DeviceSN); pushed > 0 {
				result.Status = "dispatched"
			}
		}
	}
	return result, nil
}

func (s *Service) DispatchPendingInstructions(ctx context.Context, deviceID int64, sn string) (int, error) {
	if s == nil || s.svcCtx == nil || s.svcCtx.DB == nil {
		return 0, errorx.NewDefaultError(errorx.CodeSystemError)
	}
	if s.svcCtx.MQTTClient() == nil || !s.isDeviceOnline(ctx, sn) {
		return 0, nil
	}
	limit := s.dispatchBatchSize()
	rows, err := s.svcCtx.DB.QueryContext(ctx, `
SELECT id, cmd, command_code, instruction_type, params, status, priority, retry_count, expires_at, created_at, max_retry
FROM public.device_instruction
WHERE device_id = $1
  AND status = $2
  AND (expires_at IS NULL OR expires_at >= CURRENT_TIMESTAMP)
ORDER BY priority DESC, created_at ASC
LIMIT $3`, deviceID, StatusPending, limit)
	if err != nil {
		return 0, logDBErr("DispatchPendingInstructions(query)", err)
	}
	defer rows.Close()

	type row struct {
		PendingCommand
		MaxRetry int
	}
	var items []row
	for rows.Next() {
		var item row
		if err := rows.Scan(
			&item.InstructionID, &item.Cmd, &item.CommandCode, &item.InstructionType, &item.Params,
			&item.Status, &item.Priority, &item.RetryCount, &item.ExpiresAt, &item.CreatedAt, &item.MaxRetry,
		); err != nil {
			return 0, logDBErr("DispatchPendingInstructions(scan)", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return 0, logDBErr("DispatchPendingInstructions(rows)", err)
	}

	pushed := 0
	for _, item := range items {
		if item.ExpiresAt != nil && item.ExpiresAt.Before(time.Now()) {
			_ = s.markInstructionStatus(ctx, item.InstructionID, deviceID, StatusExpired(), "expired_before_dispatch", "system")
			continue
		}
		payload, _ := json.Marshal(map[string]interface{}{
			"type":             "shadow_delta",
			"instruction_id":   item.InstructionID,
			"sn":               strings.ToUpper(strings.TrimSpace(sn)),
			"cmd":              item.Cmd,
			"command_code":     item.CommandCode,
			"instruction_type": item.InstructionType,
			"priority":         item.Priority,
			"retry_count":      item.RetryCount,
			"expires_at":       toUnix(item.ExpiresAt),
			"params":           decodeMap(item.Params),
		})
		if err := shadowmqtt.PublishDesiredCommand(s.svcCtx.Config, s.svcCtx.MQTTClient(), strings.ToUpper(strings.TrimSpace(sn)), deviceID, payload); err != nil {
			if item.RetryCount+1 >= item.MaxRetry {
				_ = s.markInstructionStatus(ctx, item.InstructionID, deviceID, StatusFailed, "dispatch_retry_exhausted:"+err.Error(), "system")
			} else {
				_ = s.bumpRetry(ctx, item.InstructionID, deviceID, err.Error())
			}
			break
		}
		now := time.Now()
		if _, err := s.svcCtx.DB.ExecContext(ctx, `
UPDATE public.device_instruction
SET status = $1,
    dispatched_at = COALESCE(dispatched_at, $2),
    received_at = COALESCE(received_at, $2),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $3 AND device_id = $4 AND status = $5`,
			StatusExecuting, now, item.InstructionID, deviceID, StatusPending,
		); err != nil {
			return pushed, logDBErr("DispatchPendingInstructions(update executing)", err)
		}
		prev := StatusPending
		_ = s.insertStateLog(ctx, item.InstructionID, &prev, StatusExecuting, "mqtt_dispatched", "system")
		pushed++
	}
	return pushed, nil
}

func (s *Service) ListPendingForDevice(ctx context.Context, deviceID int64, limit int) ([]PendingCommand, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := s.svcCtx.DB.QueryContext(ctx, `
SELECT id, cmd, command_code, instruction_type, params, status, priority, retry_count, expires_at, created_at
FROM public.device_instruction
WHERE device_id = $1
  AND status IN ($2, $3)
  AND (expires_at IS NULL OR expires_at >= CURRENT_TIMESTAMP)
ORDER BY priority DESC, created_at ASC
LIMIT $4`, deviceID, StatusPending, StatusExecuting, limit)
	if err != nil {
		return nil, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	defer rows.Close()
	list := make([]PendingCommand, 0, limit)
	for rows.Next() {
		var item PendingCommand
		if err := rows.Scan(&item.InstructionID, &item.Cmd, &item.CommandCode, &item.InstructionType, &item.Params, &item.Status, &item.Priority, &item.RetryCount, &item.ExpiresAt, &item.CreatedAt); err != nil {
			return nil, errorx.NewDefaultError(errorx.CodeDatabaseError)
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

func (s *Service) MarkInstructionResult(ctx context.Context, deviceID, instructionID int64, nextStatus int16, result json.RawMessage, errorMsg, operator string) error {
	current, err := s.getInstructionStatus(ctx, deviceID, instructionID)
	if err != nil {
		return err
	}
	if !allowedDeviceTransition(*current, nextStatus) {
		return errorx.NewCodeError(errorx.CodeInvalidParam, "非法的指令状态流转")
	}
	resultText := string(bytes.TrimSpace(result))
	if resultText == "" {
		resultText = "{}"
	}
	now := time.Now()
	_, err = s.svcCtx.DB.ExecContext(ctx, `
UPDATE public.device_instruction
SET status = $1,
    result = $2::jsonb,
    error_msg = $3,
    received_at = CASE WHEN $1 >= 2 THEN COALESCE(received_at, $4) ELSE received_at END,
    completed_at = CASE WHEN $1 IN (3,4,5,6) THEN $4 ELSE completed_at END,
    executed_at = CASE WHEN $1 IN (3,4,5,6) THEN COALESCE(executed_at, $4) ELSE executed_at END,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $5 AND device_id = $6`,
		nextStatus, resultText, strings.TrimSpace(errorMsg), now, instructionID, deviceID)
	if err != nil {
		return errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	return s.insertStateLog(ctx, instructionID, current, nextStatus, "device_result", operator)
}

func (s *Service) CancelInstructionByUser(ctx context.Context, userID, instructionID int64, reason string) error {
	var current int16
	err := s.svcCtx.DB.QueryRowContext(ctx, `
SELECT status FROM public.device_instruction
WHERE id = $1 AND user_id = $2
LIMIT 1`, instructionID, userID).Scan(&current)
	if err == sql.ErrNoRows {
		return errorx.NewDefaultError(errorx.CodeDeviceCommandNotFound)
	}
	if err != nil {
		return errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	if current != StatusPending && current != StatusExecuting {
		return errorx.NewCodeError(errorx.CodeInvalidParam, "当前指令不可取消")
	}
	if _, err := s.svcCtx.DB.ExecContext(ctx, `
UPDATE public.device_instruction
SET status = $1,
    error_msg = CASE WHEN COALESCE($2, '') = '' THEN error_msg ELSE $2 END,
    completed_at = CURRENT_TIMESTAMP,
    executed_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $3 AND user_id = $4`, StatusCancelled, strings.TrimSpace(reason), instructionID, userID); err != nil {
		return errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	return s.insertStateLog(ctx, instructionID, &current, StatusCancelled, "user_cancelled", fmt.Sprintf("user:%d", userID))
}

func (s *Service) ListInstructionsForUser(ctx context.Context, filter InstructionHistoryFilter) ([]InstructionHistoryItem, int64, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}
	args := []interface{}{filter.UserID}
	where := []string{"user_id = $1"}
	if sn := strings.ToUpper(strings.TrimSpace(filter.DeviceSN)); sn != "" {
		args = append(args, sn)
		where = append(where, fmt.Sprintf("sn = $%d", len(args)))
	}
	if filter.InstructionID > 0 {
		args = append(args, filter.InstructionID)
		where = append(where, fmt.Sprintf("id = $%d", len(args)))
	}
	if st, ok := statusFromString(filter.Status); ok {
		args = append(args, st)
		where = append(where, fmt.Sprintf("status = $%d", len(args)))
	}
	whereSQL := strings.Join(where, " AND ")

	var total int64
	if err := s.svcCtx.DB.QueryRowContext(ctx, "SELECT COUNT(1) FROM public.device_instruction WHERE "+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	args = append(args, (filter.Page-1)*filter.PageSize, filter.PageSize)
	rows, err := s.svcCtx.DB.QueryContext(ctx, `
SELECT id, device_id, sn, cmd, command_code, instruction_type, status, priority, retry_count, expires_at, params, result, COALESCE(error_msg,''), created_at, updated_at
FROM public.device_instruction
WHERE `+whereSQL+`
ORDER BY id DESC
OFFSET $`+fmt.Sprintf("%d", len(args)-1)+` LIMIT $`+fmt.Sprintf("%d", len(args)), args...)
	if err != nil {
		return nil, 0, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	defer rows.Close()
	list := make([]InstructionHistoryItem, 0, filter.PageSize)
	for rows.Next() {
		var item InstructionHistoryItem
		var expiresAt *time.Time
		var createdAt time.Time
		var updatedAt time.Time
		if err := rows.Scan(&item.InstructionID, &item.DeviceID, &item.DeviceSN, &item.Cmd, &item.CommandCode, &item.InstructionType, &item.StatusCode, &item.Priority, &item.RetryCount, &expiresAt, &item.Params, &item.Result, &item.ErrorMsg, &createdAt, &updatedAt); err != nil {
			return nil, 0, errorx.NewDefaultError(errorx.CodeDatabaseError)
		}
		item.Status = statusToString(item.StatusCode)
		item.CreatedAt = createdAt.Unix()
		item.UpdatedAt = updatedAt.Unix()
		if expiresAt != nil {
			ts := expiresAt.Unix()
			item.ExpiresAt = &ts
		}
		list = append(list, item)
	}
	return list, total, rows.Err()
}

func (s *Service) GetInstructionForUser(ctx context.Context, userID, instructionID int64) (*InstructionDetail, error) {
	list, _, err := s.ListInstructionsForUser(ctx, InstructionHistoryFilter{
		UserID:        userID,
		InstructionID: instructionID,
		Page:          1,
		PageSize:      1,
	})
	if err != nil {
		return nil, err
	}
	var item *InstructionHistoryItem
	for _, candidate := range list {
		if candidate.InstructionID == instructionID {
			current := candidate
			item = &current
			break
		}
	}
	if item == nil {
		row, err := s.svcCtx.DB.QueryContext(ctx, `
SELECT id, device_id, sn, cmd, command_code, instruction_type, status, priority, retry_count, expires_at, params, result, COALESCE(error_msg,''), created_at, updated_at
FROM public.device_instruction
WHERE id = $1 AND user_id = $2
LIMIT 1`, instructionID, userID)
		if err != nil {
			return nil, errorx.NewDefaultError(errorx.CodeDatabaseError)
		}
		defer row.Close()
		if !row.Next() {
			return nil, errorx.NewDefaultError(errorx.CodeDeviceCommandNotFound)
		}
		var out InstructionHistoryItem
		var expiresAt *time.Time
		var createdAt, updatedAt time.Time
		if err := row.Scan(&out.InstructionID, &out.DeviceID, &out.DeviceSN, &out.Cmd, &out.CommandCode, &out.InstructionType, &out.StatusCode, &out.Priority, &out.RetryCount, &expiresAt, &out.Params, &out.Result, &out.ErrorMsg, &createdAt, &updatedAt); err != nil {
			return nil, errorx.NewDefaultError(errorx.CodeDatabaseError)
		}
		out.Status = statusToString(out.StatusCode)
		out.CreatedAt = createdAt.Unix()
		out.UpdatedAt = updatedAt.Unix()
		if expiresAt != nil {
			ts := expiresAt.Unix()
			out.ExpiresAt = &ts
		}
		item = &out
	}

	rows, err := s.svcCtx.DB.QueryContext(ctx, `
SELECT from_status, to_status, note, operator, created_at
FROM public.device_instruction_state_log
WHERE instruction_id = $1
ORDER BY id ASC`, instructionID)
	if err != nil {
		return nil, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	defer rows.Close()
	logs := make([]InstructionStateLogItem, 0, 8)
	for rows.Next() {
		var from sql.NullInt16
		var logItem InstructionStateLogItem
		var createdAt time.Time
		if err := rows.Scan(&from, &logItem.ToStatus, &logItem.Note, &logItem.Operator, &createdAt); err != nil {
			return nil, errorx.NewDefaultError(errorx.CodeDatabaseError)
		}
		if from.Valid {
			value := from.Int16
			logItem.FromStatus = &value
		}
		logItem.CreatedAt = createdAt.Unix()
		logs = append(logs, logItem)
	}
	return &InstructionDetail{
		Instruction: *item,
		Logs:        logs,
	}, rows.Err()
}

func (s *Service) SaveSchedule(ctx context.Context, in ScheduleUpsertInput) (*Schedule, error) {
	if strings.TrimSpace(in.ScheduleType) == "" {
		in.ScheduleType = "once"
	}
	if in.ScheduleType != "once" && in.ScheduleType != "cron" {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "schedule_type 仅支持 once/cron")
	}
	if len(bytes.TrimSpace(in.DesiredPayload)) == 0 {
		in.DesiredPayload = []byte(`{}`)
	}
	if !json.Valid(in.DesiredPayload) {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "desired_payload 必须是合法 JSON")
	}
	if strings.TrimSpace(in.Timezone) == "" {
		in.Timezone = "Asia/Shanghai"
	}
	nextExecuteAt, err := computeNextExecuteAt(in.ScheduleType, in.ExecuteAt, in.CronExpr, in.Timezone, time.Now())
	if err != nil {
		return nil, err
	}
	if in.ID <= 0 {
		var id int64
		err = s.svcCtx.DB.QueryRowContext(ctx, `
INSERT INTO public.device_command_schedule (
	device_id, device_sn, user_id, schedule_type, desired_payload, command_payload, merge_desired,
	cron_expr, timezone, next_execute_at, status, expires_at
) VALUES (
	$1, $2, $3, $4, $5::jsonb, $6::jsonb, $7,
	$8, $9, $10, 'active', $11
)
RETURNING id`,
			in.DeviceID, strings.ToUpper(strings.TrimSpace(in.DeviceSN)), in.UserID, in.ScheduleType,
			string(in.DesiredPayload), string(in.DesiredPayload), in.MergeDesired,
			nullString(in.CronExpr), in.Timezone, nextExecuteAt, in.ExpiresAt,
		).Scan(&id)
		if err != nil {
			return nil, errorx.NewDefaultError(errorx.CodeDatabaseError)
		}
		in.ID = id
	} else {
		res, err := s.svcCtx.DB.ExecContext(ctx, `
UPDATE public.device_command_schedule
SET schedule_type = $1,
    desired_payload = $2::jsonb,
    command_payload = $3::jsonb,
    merge_desired = $4,
    cron_expr = $5,
    timezone = $6,
    next_execute_at = $7,
    status = CASE WHEN status = 'cancelled' THEN status ELSE 'active' END,
    expires_at = $8,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $9 AND user_id = $10`,
			in.ScheduleType, string(in.DesiredPayload), string(in.DesiredPayload), in.MergeDesired,
			nullString(in.CronExpr), in.Timezone, nextExecuteAt, in.ExpiresAt, in.ID, in.UserID,
		)
		if err != nil {
			return nil, errorx.NewDefaultError(errorx.CodeDatabaseError)
		}
		if rows, _ := res.RowsAffected(); rows == 0 {
			return nil, errorx.NewDefaultError(errorx.CodeDeviceCommandNotFound)
		}
	}
	return s.GetScheduleByUser(ctx, in.UserID, in.ID)
}

func (s *Service) CancelSchedule(ctx context.Context, userID, scheduleID int64) error {
	res, err := s.svcCtx.DB.ExecContext(ctx, `
UPDATE public.device_command_schedule
SET status = 'cancelled',
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND user_id = $2 AND status IN ('active', 'paused')`, scheduleID, userID)
	if err != nil {
		return errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return errorx.NewDefaultError(errorx.CodeDeviceCommandNotFound)
	}
	_, _ = s.svcCtx.DB.ExecContext(ctx, `
INSERT INTO public.device_command_schedule_log (schedule_id, status, note)
VALUES ($1, 'cancelled', 'user_cancelled')`, scheduleID)
	return nil
}

func (s *Service) ListSchedulesForUser(ctx context.Context, filter ScheduleListFilter) ([]Schedule, int64, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}
	args := []interface{}{filter.UserID}
	where := []string{"user_id = $1"}
	if sn := strings.ToUpper(strings.TrimSpace(filter.DeviceSN)); sn != "" {
		args = append(args, sn)
		where = append(where, fmt.Sprintf("device_sn = $%d", len(args)))
	}
	if st := strings.TrimSpace(filter.Status); st != "" {
		args = append(args, st)
		where = append(where, fmt.Sprintf("status = $%d", len(args)))
	}
	whereSQL := strings.Join(where, " AND ")
	var total int64
	if err := s.svcCtx.DB.QueryRowContext(ctx, "SELECT COUNT(1) FROM public.device_command_schedule WHERE "+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	args = append(args, (filter.Page-1)*filter.PageSize, filter.PageSize)
	rows, err := s.svcCtx.DB.QueryContext(ctx, `
SELECT id, device_id, device_sn, user_id, schedule_type, desired_payload, command_payload, merge_desired, COALESCE(cron_expr,''), timezone, next_execute_at, last_execute_at, status, expires_at, created_at, updated_at
FROM public.device_command_schedule
WHERE `+whereSQL+`
ORDER BY id DESC
OFFSET $`+fmt.Sprintf("%d", len(args)-1)+` LIMIT $`+fmt.Sprintf("%d", len(args)), args...)
	if err != nil {
		return nil, 0, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	defer rows.Close()
	list := make([]Schedule, 0, filter.PageSize)
	for rows.Next() {
		item, err := scanSchedule(rows)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, *item)
	}
	return list, total, rows.Err()
}

func (s *Service) GetScheduleByUser(ctx context.Context, userID, scheduleID int64) (*Schedule, error) {
	rows, err := s.svcCtx.DB.QueryContext(ctx, `
SELECT id, device_id, device_sn, user_id, schedule_type, desired_payload, command_payload, merge_desired, COALESCE(cron_expr,''), timezone, next_execute_at, last_execute_at, status, expires_at, created_at, updated_at
FROM public.device_command_schedule
WHERE id = $1 AND user_id = $2
LIMIT 1`, scheduleID, userID)
	if err != nil {
		return nil, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, errorx.NewDefaultError(errorx.CodeDeviceCommandNotFound)
	}
	return scanSchedule(rows)
}

func (s *Service) LoadDueSchedules(ctx context.Context, limit int) ([]Schedule, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.svcCtx.DB.QueryContext(ctx, `
SELECT id, device_id, device_sn, user_id, schedule_type, desired_payload, command_payload, merge_desired, COALESCE(cron_expr,''), timezone, next_execute_at, last_execute_at, status, expires_at, created_at, updated_at
FROM public.device_command_schedule
WHERE status = 'active'
  AND next_execute_at IS NOT NULL
  AND next_execute_at <= CURRENT_TIMESTAMP
  AND (expires_at IS NULL OR expires_at >= CURRENT_TIMESTAMP)
ORDER BY next_execute_at ASC
LIMIT $1`, limit)
	if err != nil {
		return nil, logDBErr("LoadDueSchedules(query)", err)
	}
	defer rows.Close()
	list := make([]Schedule, 0, limit)
	for rows.Next() {
		item, err := scanSchedule(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *item)
	}
	return list, logDBErr("LoadDueSchedules(rows)", rows.Err())
}

func (s *Service) MarkScheduleTriggered(ctx context.Context, schedule Schedule, instructionID int64) error {
	now := time.Now()
	nextExecuteAt, status, err := nextAfterTrigger(schedule, now)
	if err != nil {
		return err
	}
	_, err = s.svcCtx.DB.ExecContext(ctx, `
UPDATE public.device_command_schedule
SET last_execute_at = $1,
    next_execute_at = $2,
    status = $3,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $4`, now, nextExecuteAt, status, schedule.ID)
	if err != nil {
		return errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	_, _ = s.svcCtx.DB.ExecContext(ctx, `
INSERT INTO public.device_command_schedule_log (schedule_id, instruction_id, status, note)
VALUES ($1, $2, $3, $4)`, schedule.ID, nullableInt64(instructionID), scheduleLogStatus(status), "worker_triggered")
	return nil
}

func (s *Service) MarkScheduleTriggerFailed(ctx context.Context, scheduleID int64, note string) {
	_, _ = s.svcCtx.DB.ExecContext(ctx, `
INSERT INTO public.device_command_schedule_log (schedule_id, status, note)
VALUES ($1, 'failed', $2)`, scheduleID, trimNote(note))
}

func (s *Service) ExpireAndTimeoutInstructions(ctx context.Context) error {
	if _, err := s.svcCtx.DB.ExecContext(ctx, `
UPDATE public.device_instruction
SET status = $1,
    error_msg = 'expired',
    completed_at = CURRENT_TIMESTAMP,
    executed_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE status = $2
  AND expires_at IS NOT NULL
  AND expires_at < CURRENT_TIMESTAMP`, StatusCancelled, StatusPending); err != nil {
		return logDBErr("ExpireAndTimeoutInstructions(expire)", err)
	}
	if _, err := s.svcCtx.DB.ExecContext(ctx, `
UPDATE public.device_instruction
SET status = $1,
    error_msg = 'timeout',
    completed_at = CURRENT_TIMESTAMP,
    executed_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE status = $2
  AND received_at IS NOT NULL
  AND received_at + make_interval(secs => timeout_seconds) < CURRENT_TIMESTAMP`, StatusTimeout, StatusExecuting); err != nil {
		return logDBErr("ExpireAndTimeoutInstructions(timeout)", err)
	}
	return nil
}

func (s *Service) RedrivePendingInstructions(ctx context.Context, limit int) error {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.svcCtx.DB.QueryContext(ctx, `
SELECT DISTINCT device_id, sn
FROM public.device_instruction
WHERE status = $1
  AND (expires_at IS NULL OR expires_at >= CURRENT_TIMESTAMP)
ORDER BY device_id ASC
LIMIT $2`, StatusPending, limit)
	if err != nil {
		return logDBErr("RedrivePendingInstructions(query)", err)
	}
	defer rows.Close()
	for rows.Next() {
		var deviceID int64
		var sn string
		if err := rows.Scan(&deviceID, &sn); err != nil {
			return logDBErr("RedrivePendingInstructions(scan)", err)
		}
		if _, err := s.DispatchPendingInstructions(ctx, deviceID, sn); err != nil {
			return err
		}
	}
	return logDBErr("RedrivePendingInstructions(rows)", rows.Err())
}

func (s *Service) defaultExpiresSeconds() int {
	if s == nil || s.svcCtx == nil || s.svcCtx.Config.DeviceCommand.DefaultExpiresSeconds <= 0 {
		return 900
	}
	return s.svcCtx.Config.DeviceCommand.DefaultExpiresSeconds
}

func (s *Service) defaultTimeoutSeconds() int {
	if s == nil || s.svcCtx == nil || s.svcCtx.Config.DeviceCommand.DefaultTimeoutSeconds <= 0 {
		return 300
	}
	return s.svcCtx.Config.DeviceCommand.DefaultTimeoutSeconds
}

func (s *Service) defaultMaxRetry() int {
	if s == nil || s.svcCtx == nil || s.svcCtx.Config.DeviceCommand.DefaultMaxRetry <= 0 {
		return 3
	}
	return s.svcCtx.Config.DeviceCommand.DefaultMaxRetry
}

func (s *Service) dispatchBatchSize() int {
	if s == nil || s.svcCtx == nil || s.svcCtx.Config.DeviceCommand.DispatchBatchSize <= 0 {
		return 20
	}
	return s.svcCtx.Config.DeviceCommand.DispatchBatchSize
}

func (s *Service) countPendingForDevice(ctx context.Context, deviceID int64) int {
	var count int
	_ = s.svcCtx.DB.QueryRowContext(ctx, `
SELECT COUNT(1)
FROM public.device_instruction
WHERE device_id = $1 AND status IN ($2, $3)`, deviceID, StatusPending, StatusExecuting).Scan(&count)
	return count
}

func (s *Service) getInstructionStatus(ctx context.Context, deviceID, instructionID int64) (*int16, error) {
	var status int16
	err := s.svcCtx.DB.QueryRowContext(ctx, `
SELECT status
FROM public.device_instruction
WHERE id = $1 AND device_id = $2
LIMIT 1`, instructionID, deviceID).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errorx.NewDefaultError(errorx.CodeDeviceCommandNotFound)
	}
	if err != nil {
		return nil, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	return &status, nil
}

func (s *Service) mergeConflictingPendingInstructionsTx(ctx context.Context, tx *sql.Tx, deviceID int64, commandCode, instructionType, operator string) (int, error) {
	rows, err := tx.QueryContext(ctx, `
SELECT id, status, instruction_type
FROM public.device_instruction
WHERE device_id = $1
  AND command_code = $2
  AND status = $3`, deviceID, commandCode, StatusPending)
	if err != nil {
		return 0, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		var status int16
		var oldType string
		if err := rows.Scan(&id, &status, &oldType); err != nil {
			return 0, errorx.NewDefaultError(errorx.CodeDatabaseError)
		}
		if instructionType == InstructionTypeManual || oldType == InstructionTypeManual || oldType == InstructionTypeScheduled {
			ids = append(ids, id)
		}
	}
	for _, id := range ids {
		if _, err := tx.ExecContext(ctx, `
UPDATE public.device_instruction
SET status = $1,
    error_msg = 'merged_by_newer_instruction',
    completed_at = CURRENT_TIMESTAMP,
    executed_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $2 AND status = $3`, StatusCancelled, id, StatusPending); err != nil {
			return 0, errorx.NewDefaultError(errorx.CodeDatabaseError)
		}
		prev := StatusPending
		if err := insertStateLogTx(ctx, tx, id, &prev, StatusCancelled, "merged_by_newer_instruction", operator); err != nil {
			return 0, errorx.NewDefaultError(errorx.CodeDatabaseError)
		}
	}
	return len(ids), nil
}

func (s *Service) bumpRetry(ctx context.Context, instructionID, deviceID int64, message string) error {
	_, err := s.svcCtx.DB.ExecContext(ctx, `
UPDATE public.device_instruction
SET retry_count = retry_count + 1,
    error_msg = $1,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $2 AND device_id = $3`, trimNote(message), instructionID, deviceID)
	if err != nil {
		return errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	prev := StatusPending
	return s.insertStateLog(ctx, instructionID, &prev, StatusPending, "dispatch_retry", "system")
}

func (s *Service) markInstructionStatus(ctx context.Context, instructionID, deviceID int64, nextStatus int16, note, operator string) error {
	current, err := s.getInstructionStatus(ctx, deviceID, instructionID)
	if err != nil {
		return err
	}
	_, err = s.svcCtx.DB.ExecContext(ctx, `
UPDATE public.device_instruction
SET status = $1,
    error_msg = $2,
    completed_at = CASE WHEN $1 IN (3,4,5,6) THEN CURRENT_TIMESTAMP ELSE completed_at END,
    executed_at = CASE WHEN $1 IN (3,4,5,6) THEN CURRENT_TIMESTAMP ELSE executed_at END,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $3 AND device_id = $4`, nextStatus, trimNote(note), instructionID, deviceID)
	if err != nil {
		return errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	return s.insertStateLog(ctx, instructionID, current, nextStatus, note, operator)
}

func (s *Service) insertStateLog(ctx context.Context, instructionID int64, from *int16, to int16, note, operator string) error {
	_, err := s.svcCtx.DB.ExecContext(ctx, `
INSERT INTO public.device_instruction_state_log (instruction_id, from_status, to_status, note, operator)
VALUES ($1, $2, $3, $4, $5)`, instructionID, nullableInt16(from), to, trimNote(note), strings.TrimSpace(operator))
	return err
}

func insertStateLogTx(ctx context.Context, tx *sql.Tx, instructionID int64, from *int16, to int16, note, operator string) error {
	_, err := tx.ExecContext(ctx, `
INSERT INTO public.device_instruction_state_log (instruction_id, from_status, to_status, note, operator)
VALUES ($1, $2, $3, $4, $5)`, instructionID, nullableInt16(from), to, trimNote(note), strings.TrimSpace(operator))
	return err
}

func (s *Service) isDeviceOnline(ctx context.Context, sn string) bool {
	if s == nil || s.svcCtx == nil {
		logx.Errorf("isDeviceOnline: svcCtx is nil, sn=%s", sn)
		return false
	}

	if s.svcCtx.Redis != nil {
		val, err := s.svcCtx.Redis.Get(ctx, redisshadow.OnlineKey(sn)).Result()
		if err == nil && strings.TrimSpace(val) == "1" {
			logx.Infof("isDeviceOnline: Redis hit, sn=%s, online=true", sn)
			return true
		}
		logx.Infof("isDeviceOnline: Redis miss, sn=%s, err=%v, val=%s", sn, err, val)
	}

	if s.svcCtx.DB != nil {
		var onlineStatus int16
		upperSN := strings.ToUpper(strings.TrimSpace(sn))
		err := s.svcCtx.DB.QueryRowContext(ctx, `
			SELECT online_status FROM device
			WHERE UPPER(TRIM(sn)) = $1 AND deleted_at IS NULL
			LIMIT 1
		`, upperSN).Scan(&onlineStatus)
		if err != nil {
			logx.Errorf("isDeviceOnline: MySQL query failed, sn=%s, upperSN=%s, err=%v", sn, upperSN, err)
		} else {
			logx.Infof("isDeviceOnline: MySQL result, sn=%s, onlineStatus=%d", sn, onlineStatus)
			if onlineStatus == 1 {
				return true
			}
		}
	} else {
		logx.Errorf("isDeviceOnline: DB is nil, sn=%s", sn)
	}

	return false
}

func allowedDeviceTransition(current, next int16) bool {
	switch next {
	case StatusExecuting:
		return current == StatusPending || current == StatusExecuting
	case StatusSuccess, StatusFailed:
		return current == StatusPending || current == StatusExecuting
	default:
		return false
	}
}

func statusToString(status int16) string {
	switch status {
	case StatusPending:
		return "pending"
	case StatusExecuting:
		return "executing"
	case StatusSuccess:
		return "success"
	case StatusFailed:
		return "failed"
	case StatusTimeout:
		return "timeout"
	case StatusCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

func statusFromString(status string) (int16, bool) {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", "all":
		return 0, false
	case "pending":
		return StatusPending, true
	case "executing":
		return StatusExecuting, true
	case "success":
		return StatusSuccess, true
	case "failed":
		return StatusFailed, true
	case "timeout":
		return StatusTimeout, true
	case "cancelled":
		return StatusCancelled, true
	default:
		return 0, false
	}
}

func computeNextExecuteAt(scheduleType string, executeAt *time.Time, cronExpr, timezone string, now time.Time) (*time.Time, error) {
	switch scheduleType {
	case "once":
		if executeAt == nil {
			return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "once 类型必须传 execute_at")
		}
		t := executeAt.UTC()
		return &t, nil
	case "cron":
		expr := strings.TrimSpace(cronExpr)
		if expr == "" {
			return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "cron 类型必须传 cron_expr")
		}
		loc, err := time.LoadLocation(timezone)
		if err != nil {
			return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "timezone 非法")
		}
		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		schedule, err := parser.Parse(expr)
		if err != nil {
			return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "cron_expr 非法")
		}
		next := schedule.Next(now.In(loc))
		next = next.UTC()
		return &next, nil
	default:
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "未知 schedule_type")
	}
}

func nextAfterTrigger(schedule Schedule, now time.Time) (*time.Time, string, error) {
	if schedule.ScheduleType == "once" {
		return nil, "completed", nil
	}
	next, err := computeNextExecuteAt(schedule.ScheduleType, nil, schedule.CronExpr, schedule.Timezone, now)
	if err != nil {
		return nil, "", err
	}
	if schedule.ExpiresAt != nil && next != nil && next.After(*schedule.ExpiresAt) {
		return nil, "completed", nil
	}
	return next, "active", nil
}

func scheduleLogStatus(status string) string {
	switch status {
	case "completed":
		return "completed"
	default:
		return "executed"
	}
}

func scanSchedule(scanner interface {
	Scan(dest ...interface{}) error
}) (*Schedule, error) {
	var item Schedule
	var nextExecuteAt sql.NullTime
	var lastExecuteAt sql.NullTime
	var expiresAt sql.NullTime
	if err := scanner.Scan(
		&item.ID, &item.DeviceID, &item.DeviceSN, &item.UserID, &item.ScheduleType,
		&item.DesiredPayload, &item.CommandPayload, &item.MergeDesired, &item.CronExpr, &item.Timezone,
		&nextExecuteAt, &lastExecuteAt, &item.Status, &expiresAt, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return nil, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	if nextExecuteAt.Valid {
		t := nextExecuteAt.Time
		item.NextExecuteAt = &t
	}
	if lastExecuteAt.Valid {
		t := lastExecuteAt.Time
		item.LastExecuteAt = &t
	}
	if expiresAt.Valid {
		t := expiresAt.Time
		item.ExpiresAt = &t
	}
	return &item, nil
}

func decodeMap(raw json.RawMessage) map[string]interface{} {
	if len(bytes.TrimSpace(raw)) == 0 {
		return map[string]interface{}{}
	}
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]interface{}{}
	}
	return out
}

func trimNote(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if len(s) > 500 {
		return s[:500]
	}
	return s
}

func nullableInt16(v *int16) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

func nullableInt64(v int64) interface{} {
	if v <= 0 {
		return nil
	}
	return v
}

func nullString(v string) interface{} {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return v
}

func toUnix(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.Unix()
}

func StatusExpired() int16 {
	return StatusCancelled
}
