package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// InstructionExecutionQuery 单条指令执行状态查询条件：instruction_id 与 sn+command_id 二选一
type InstructionExecutionQuery struct {
	InstructionID int64
	Sn            string
	CommandID     string // params.command_id，如 cmd_123
}

// InstructionExecutionOut 单条指令执行状态（全流程视图）
type InstructionExecutionOut struct {
	CmdID          string                    `json:"cmd_id"`
	DeviceID       int64                     `json:"device_id"`
	Sn             string                    `json:"sn"`
	Command        string                    `json:"command"`
	Params         json.RawMessage           `json:"params"`
	Status         string                    `json:"status"`
	StatusText     string                    `json:"status_text"`
	Progress       int                       `json:"progress"`
	CreatedAt      time.Time                 `json:"created_at"`
	ReceivedAt     *time.Time                `json:"received_at,omitempty"`
	CompletedAt    *time.Time                `json:"completed_at,omitempty"`
	Operator       string                    `json:"operator"`
	Reason         string                    `json:"reason,omitempty"`
	Response       json.RawMessage           `json:"response,omitempty"`
	ErrorMsg       string                    `json:"error_msg,omitempty"`
	RetryCount     int                       `json:"retry_count"`
	TimeoutSeconds int                       `json:"timeout"`
	StateTimeline  []InstructionStateEntry   `json:"state_timeline"`
	OperationLogs  []DeviceEventSummary      `json:"operation_logs"`
}

// InstructionStateEntry 状态变化时序
type InstructionStateEntry struct {
	FromStatus string    `json:"from_status,omitempty"`
	ToStatus   string    `json:"to_status"`
	At         time.Time `json:"at"`
	Note       string    `json:"note,omitempty"`
	Operator   string    `json:"operator,omitempty"`
}

func instructionStatusTextZH(st int16) string {
	switch st {
	case InstrStatusPending:
		return "待执行"
	case InstrStatusExecuting:
		return "执行中"
	case InstrStatusSuccess:
		return "执行成功"
	case InstrStatusFailed:
		return "执行失败"
	case InstrStatusTimeout:
		return "已超时"
	case InstrStatusCancelled:
		return "已取消"
	default:
		return "未知"
	}
}

func instructionExecutionProgress(st int16) int {
	switch st {
	case InstrStatusPending:
		return 5 // 0~10 区间取中
	case InstrStatusExecuting:
		return 50 // 10~90 区间取中
	case InstrStatusSuccess, InstrStatusFailed, InstrStatusTimeout, InstrStatusCancelled:
		return 100
	default:
		return 0
	}
}

func extractErrorMsg(result json.RawMessage, column string) string {
	s := strings.TrimSpace(column)
	if s != "" {
		return s
	}
	if len(result) == 0 {
		return ""
	}
	var m map[string]interface{}
	if json.Unmarshal(result, &m) != nil {
		return ""
	}
	for _, k := range []string{"error_msg", "error", "message", "err"} {
		if v, ok := m[k].(string); ok && strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

type instructionFullRow struct {
	ID             int64           `gorm:"column:id"`
	DeviceID       int64           `gorm:"column:device_id"`
	Sn             string          `gorm:"column:sn"`
	Cmd            string          `gorm:"column:cmd"`
	Params         json.RawMessage `gorm:"column:params"`
	Status         int16           `gorm:"column:status"`
	Operator       string          `gorm:"column:operator"`
	Reason         string          `gorm:"column:reason"`
	Result         json.RawMessage `gorm:"column:result"`
	ReceivedAt     *time.Time      `gorm:"column:received_at"`
	CompletedAt    *time.Time      `gorm:"column:completed_at"`
	TimeoutSeconds int             `gorm:"column:timeout_seconds"`
	RetryCount     int             `gorm:"column:retry_count"`
	ErrorMsg       string          `gorm:"column:error_msg"`
	CreatedAt      time.Time       `gorm:"column:created_at"`
	UpdatedAt      time.Time       `gorm:"column:updated_at"`
}

type stateLogRow struct {
	FromStatus *int16    `gorm:"column:from_status"`
	ToStatus   int16     `gorm:"column:to_status"`
	Note       string    `gorm:"column:note"`
	Operator   string    `gorm:"column:operator"`
	CreatedAt  time.Time `gorm:"column:created_at"`
}

// GetInstructionExecution 查询单条指令执行全流程状态
func (e *PlatformDeviceService) GetInstructionExecution(q InstructionExecutionQuery) (*InstructionExecutionOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	var r instructionFullRow
	var err error
	if q.InstructionID > 0 {
		err = e.Orm.Table("device_instruction").Where("id = ?", q.InstructionID).Take(&r).Error
	} else if strings.TrimSpace(q.Sn) != "" && strings.TrimSpace(q.CommandID) != "" {
		err = e.Orm.Table("device_instruction").
			Where("sn = ? AND (params::jsonb ->> 'command_id') = ?", strings.TrimSpace(q.Sn), strings.TrimSpace(q.CommandID)).
			Order("id DESC").
			First(&r).Error
	} else {
		return nil, fmt.Errorf("请传 instruction_id，或同时传 sn 与 command_id")
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrPlatformDeviceNotFound
	}
	if err != nil {
		return nil, err
	}

	cmdID := extractCommandID(r.Params)
	if cmdID == "" {
		cmdID = fmt.Sprintf("cmd_%d", r.ID)
	}

	reason := strings.TrimSpace(r.Reason)
	if reason == "" {
		reason = inferReasonFromParams(r.Params)
	}
	op := strings.TrimSpace(r.Operator)

	timeline := e.loadInstructionStateTimeline(r.ID)
	if len(timeline) == 0 {
		timeline = synthesizeInstructionStateTimeline(r)
	}

	var ops []DeviceEventSummary
	_ = e.Orm.Table("device_event_log").
		Select("id, event_type, content, operator, created_at").
		Where("device_id = ? AND created_at >= ? AND created_at <= ?", r.DeviceID, r.CreatedAt.Add(-1*time.Minute), r.UpdatedAt.Add(1*time.Hour)).
		Order("id ASC").
		Limit(50).
		Scan(&ops).Error

	received := r.ReceivedAt
	completed := r.CompletedAt
	// 终态若未单独记录 completed_at，用 updated_at 便于前端展示（旧数据兼容）
	if completed == nil && (r.Status == InstrStatusSuccess || r.Status == InstrStatusFailed || r.Status == InstrStatusTimeout || r.Status == InstrStatusCancelled) {
		t := r.UpdatedAt
		completed = &t
	}

	out := &InstructionExecutionOut{
		CmdID:          cmdID,
		DeviceID:       r.DeviceID,
		Sn:             r.Sn,
		Command:        r.Cmd,
		Params:         r.Params,
		Status:         instructionStatusToString(r.Status),
		StatusText:     instructionStatusTextZH(r.Status),
		Progress:       instructionExecutionProgress(r.Status),
		CreatedAt:      r.CreatedAt,
		ReceivedAt:     received,
		CompletedAt:    completed,
		Operator:       op,
		Reason:         reason,
		Response:       r.Result,
		ErrorMsg:       extractErrorMsg(r.Result, r.ErrorMsg),
		RetryCount:     r.RetryCount,
		TimeoutSeconds: r.TimeoutSeconds,
		StateTimeline:  timeline,
		OperationLogs:  ops,
	}
	if out.TimeoutSeconds <= 0 {
		out.TimeoutSeconds = 300
	}
	return out, nil
}

func (e *PlatformDeviceService) loadInstructionStateTimeline(instructionID int64) []InstructionStateEntry {
	var rows []stateLogRow
	if err := e.Orm.Table("device_instruction_state_log").
		Where("instruction_id = ?", instructionID).
		Order("id ASC").
		Scan(&rows).Error; err != nil || len(rows) == 0 {
		return nil
	}
	out := make([]InstructionStateEntry, 0, len(rows))
	for _, row := range rows {
		ent := InstructionStateEntry{
			ToStatus: instructionStatusToString(row.ToStatus),
			At:       row.CreatedAt,
			Note:     row.Note,
			Operator: strings.TrimSpace(row.Operator),
		}
		if row.FromStatus != nil {
			ent.FromStatus = instructionStatusToString(*row.FromStatus)
		}
		out = append(out, ent)
	}
	return out
}

func synthesizeInstructionStateTimeline(r instructionFullRow) []InstructionStateEntry {
	var tl []InstructionStateEntry
	tl = append(tl, InstructionStateEntry{
		ToStatus: "pending",
		At:       r.CreatedAt,
		Note:     "指令已创建",
		Operator: strings.TrimSpace(r.Operator),
	})
	switch r.Status {
	case InstrStatusPending:
		return tl
	case InstrStatusExecuting:
		at := r.UpdatedAt
		if r.ReceivedAt != nil {
			at = *r.ReceivedAt
		}
		tl = append(tl, InstructionStateEntry{
			FromStatus: "pending",
			ToStatus:   "executing",
			At:         at,
			Note:       "执行中（无流水表时的推断）",
		})
		return tl
	case InstrStatusCancelled:
		at := r.UpdatedAt
		if r.CompletedAt != nil {
			at = *r.CompletedAt
		}
		tl = append(tl, InstructionStateEntry{
			FromStatus: "pending",
			ToStatus:   "cancelled",
			At:         at,
			Note:       "已取消",
		})
		return tl
	default:
		recv := r.UpdatedAt
		if r.ReceivedAt != nil {
			recv = *r.ReceivedAt
		} else if r.CreatedAt.Before(r.UpdatedAt) {
			recv = r.CreatedAt.Add(time.Second)
		}
		tl = append(tl, InstructionStateEntry{
			FromStatus: "pending",
			ToStatus:   "executing",
			At:         recv,
			Note:       "设备已接收（推断）",
		})
		done := r.UpdatedAt
		if r.CompletedAt != nil {
			done = *r.CompletedAt
		}
		tl = append(tl, InstructionStateEntry{
			FromStatus: "executing",
			ToStatus:   instructionStatusToString(r.Status),
			At:         done,
			Note:       "状态已更新（推断）",
		})
		return tl
	}
}

// InsertInstructionStateLog 写入状态流水（供指令创建/下发/取消等调用；表不存在时忽略错误由调用方处理）
func InsertInstructionStateLog(tx *gorm.DB, instructionID int64, from *int16, to int16, note, operator string) error {
	if tx == nil {
		return fmt.Errorf("tx nil")
	}
	q := `INSERT INTO device_instruction_state_log (instruction_id, from_status, to_status, note, operator) VALUES (?,?,?,?,?)`
	var fromVal interface{}
	if from != nil {
		fromVal = *from
	} else {
		fromVal = nil
	}
	return tx.Exec(q, instructionID, fromVal, to, truncateEvent(note), strings.TrimSpace(operator)).Error
}
