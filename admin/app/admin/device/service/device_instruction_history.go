package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

// 设备指令状态（device_instruction.status），与迁移 047 一致
const (
	InstrStatusPending    int16 = 1 // pending 待执行
	InstrStatusExecuting  int16 = 2 // executing 执行中（已下发设备）
	InstrStatusSuccess    int16 = 3 // success
	InstrStatusFailed     int16 = 4 // failed
	InstrStatusTimeout    int16 = 5 // timeout
	InstrStatusCancelled  int16 = 6 // cancelled
)

// InstructionListFilter 指令历史列表筛选
type InstructionListFilter struct {
	DeviceID    int64
	Sn          string
	SnExact     bool
	Cmd         string
	Status      string // pending|executing|success|failed|timeout|cancelled 空为全部
	CreatedFrom *time.Time
	CreatedTo   *time.Time
}

// InstructionHistoryItem 列表行
type InstructionHistoryItem struct {
	ID          int64           `json:"id"`
	DeviceID    int64           `json:"device_id"`
	Sn          string          `json:"sn"`
	DeviceModel string          `json:"device_model,omitempty"`
	Command     string          `json:"command"`
	CommandID   string          `json:"command_id,omitempty"`
	Params      json.RawMessage `json:"params"`
	Status      string          `json:"status"`
	Operator    string          `json:"operator"`
	Reason      string          `json:"reason,omitempty"`
	Response    json.RawMessage `json:"response,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// InstructionDetailOut 指令详情
type InstructionDetailOut struct {
	Instruction InstructionHistoryItem   `json:"instruction"`
	Device      *InstructionDeviceBrief  `json:"device,omitempty"`
	Milestones  []InstructionMilestone     `json:"milestones"`
	Events      []DeviceEventSummary     `json:"recent_events,omitempty"`
}

// InstructionDeviceBrief 设备摘要
type InstructionDeviceBrief struct {
	ID              int64  `json:"id"`
	Sn              string `json:"sn"`
	ProductKey      string `json:"product_key"`
	Model           string `json:"model"`
	FirmwareVersion string `json:"firmware_version"`
	OnlineStatus    int16  `json:"online_status"`
}

// InstructionMilestone 状态节点（当前库仅精确记录 created_at/updated_at）
type InstructionMilestone struct {
	Phase  string `json:"phase"`  // created | last_update
	Status string `json:"status"` // 该时刻所知的业务状态
	At     string `json:"at"`     // RFC3339
	Note   string `json:"note,omitempty"`
}

func instructionStatusToString(st int16) string {
	switch st {
	case InstrStatusPending:
		return "pending"
	case InstrStatusExecuting:
		return "executing"
	case InstrStatusSuccess:
		return "success"
	case InstrStatusFailed:
		return "failed"
	case InstrStatusTimeout:
		return "timeout"
	case InstrStatusCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

// InstructionStatusFilterOK 校验列表筛选 status 参数是否合法（空或 all 视为不限）
func InstructionStatusFilterOK(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || strings.EqualFold(s, "all") {
		return true
	}
	_, ok := instructionStatusFromString(s)
	return ok
}

func instructionStatusFromString(s string) (int16, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "all":
		return 0, false
	case "pending":
		return InstrStatusPending, true
	case "executing":
		return InstrStatusExecuting, true
	case "success":
		return InstrStatusSuccess, true
	case "failed":
		return InstrStatusFailed, true
	case "timeout":
		return InstrStatusTimeout, true
	case "cancelled":
		return InstrStatusCancelled, true
	default:
		return 0, false
	}
}

func extractCommandID(params json.RawMessage) string {
	if len(params) == 0 {
		return ""
	}
	var m map[string]interface{}
	if json.Unmarshal(params, &m) != nil {
		return ""
	}
	if v, ok := m["command_id"].(string); ok {
		return v
	}
	return ""
}

func (e *PlatformDeviceService) instructionListQuery(f InstructionListFilter) *gorm.DB {
	q := e.Orm.Table("device_instruction AS di").
		Joins("LEFT JOIN device AS d ON d.id = di.device_id")

	if f.DeviceID > 0 {
		q = q.Where("di.device_id = ?", f.DeviceID)
	}
	if s := strings.TrimSpace(f.Sn); s != "" {
		if f.SnExact {
			q = q.Where("di.sn = ?", s)
		} else {
			q = q.Where("di.sn ILIKE ?", "%"+s+"%")
		}
	}
	if s := strings.TrimSpace(f.Cmd); s != "" {
		q = q.Where("di.cmd ILIKE ?", "%"+s+"%")
	}
	if f.Status != "" && !strings.EqualFold(strings.TrimSpace(f.Status), "all") {
		if st, ok := instructionStatusFromString(f.Status); ok {
			q = q.Where("di.status = ?", st)
		}
	}
	if f.CreatedFrom != nil {
		q = q.Where("di.created_at >= ?", *f.CreatedFrom)
	}
	if f.CreatedTo != nil {
		t := *f.CreatedTo
		if t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 {
			t = t.Add(24*time.Hour - time.Nanosecond)
		}
		q = q.Where("di.created_at <= ?", t)
	}
	return q
}

// ListDeviceInstructions 分页查询指令历史
func (e *PlatformDeviceService) ListDeviceInstructions(page, pageSize int, f InstructionListFilter) ([]InstructionHistoryItem, int64, error) {
	if e.Orm == nil {
		return nil, 0, fmt.Errorf("orm nil")
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}

	var total int64
	if err := e.instructionListQuery(f).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	type row struct {
		ID           int64           `gorm:"column:id"`
		DeviceID     int64           `gorm:"column:device_id"`
		Sn           string          `gorm:"column:sn"`
		Cmd          string          `gorm:"column:cmd"`
		Params       json.RawMessage `gorm:"column:params"`
		Status       int16           `gorm:"column:status"`
		Operator     string          `gorm:"column:operator"`
		Reason       string          `gorm:"column:reason"`
		Result       json.RawMessage `gorm:"column:result"`
		CreatedAt    time.Time       `gorm:"column:created_at"`
		UpdatedAt    time.Time       `gorm:"column:updated_at"`
		DeviceModel  string          `gorm:"column:device_model"`
	}
	var rows []row
	if err := e.instructionListQuery(f).Select(`di.id, di.device_id, di.sn, di.cmd, di.params, di.status, di.operator, di.reason, di.result, di.created_at, di.updated_at,
		COALESCE(d.model, '') AS device_model`).
		Order("di.id DESC").
		Offset(offset).Limit(pageSize).
		Scan(&rows).Error; err != nil {
		return nil, 0, err
	}

	out := make([]InstructionHistoryItem, 0, len(rows))
	for _, r := range rows {
		op := strings.TrimSpace(r.Operator)
		reason := strings.TrimSpace(r.Reason)
		if reason == "" {
			reason = inferReasonFromParams(r.Params)
		}
		out = append(out, InstructionHistoryItem{
			ID:          r.ID,
			DeviceID:    r.DeviceID,
			Sn:          r.Sn,
			DeviceModel: strings.TrimSpace(r.DeviceModel),
			Command:     r.Cmd,
			CommandID:   extractCommandID(r.Params),
			Params:      r.Params,
			Status:      instructionStatusToString(r.Status),
			Operator:    op,
			Reason:      reason,
			Response:    r.Result,
			CreatedAt:   r.CreatedAt,
			UpdatedAt:   r.UpdatedAt,
		})
	}
	return out, total, nil
}

func inferReasonFromParams(params json.RawMessage) string {
	var m map[string]interface{}
	if json.Unmarshal(params, &m) != nil {
		return ""
	}
	if v, ok := m["reason"].(string); ok {
		return v
	}
	return ""
}

// GetDeviceInstructionDetail 指令详情
func (e *PlatformDeviceService) GetDeviceInstructionDetail(id int64) (*InstructionDetailOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	if id <= 0 {
		return nil, ErrPlatformDeviceInvalid
	}

	type row struct {
		ID        int64           `gorm:"column:id"`
		DeviceID  int64           `gorm:"column:device_id"`
		Sn        string          `gorm:"column:sn"`
		Cmd       string          `gorm:"column:cmd"`
		Params    json.RawMessage `gorm:"column:params"`
		Status    int16           `gorm:"column:status"`
		Operator  string          `gorm:"column:operator"`
		Reason    string          `gorm:"column:reason"`
		Result    json.RawMessage `gorm:"column:result"`
		CreatedAt time.Time       `gorm:"column:created_at"`
		UpdatedAt time.Time       `gorm:"column:updated_at"`
	}
	var r row
	err := e.Orm.Table("device_instruction").Where("id = ?", id).Take(&r).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrPlatformDeviceNotFound
	}
	if err != nil {
		return nil, err
	}

	op := strings.TrimSpace(r.Operator)
	reason := strings.TrimSpace(r.Reason)
	if reason == "" {
		reason = inferReasonFromParams(r.Params)
	}
	item := InstructionHistoryItem{
		ID:        r.ID,
		DeviceID:  r.DeviceID,
		Sn:        r.Sn,
		Command:   r.Cmd,
		CommandID: extractCommandID(r.Params),
		Params:    r.Params,
		Status:    instructionStatusToString(r.Status),
		Operator:  op,
		Reason:    reason,
		Response:  r.Result,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}

	var dev InstructionDeviceBrief
	if err := e.Orm.Table("device").
		Select("id, sn, product_key, model, firmware_version, online_status").
		Where("id = ?", r.DeviceID).
		Take(&dev).Error; err == nil {
		// ok
	} else {
		dev = InstructionDeviceBrief{ID: r.DeviceID, Sn: r.Sn}
	}

	milestones := buildInstructionMilestones(r.Status, r.CreatedAt, r.UpdatedAt)

	var ev []DeviceEventSummary
	_ = e.Orm.Table("device_event_log").
		Select("id, event_type, content, operator, created_at").
		Where("device_id = ? AND created_at >= ?", r.DeviceID, r.CreatedAt.Add(-2*time.Minute)).
		Order("id ASC").
		Limit(30).
		Scan(&ev).Error

	return &InstructionDetailOut{
		Instruction: item,
		Device:      &dev,
		Milestones:  milestones,
		Events:      ev,
	}, nil
}

func buildInstructionMilestones(st int16, created, updated time.Time) []InstructionMilestone {
	ms := []InstructionMilestone{
		{Phase: "created", Status: "pending", At: created.Format(time.RFC3339), Note: "指令已创建"},
	}
	if updated.After(created) || st != InstrStatusPending {
		ms = append(ms, InstructionMilestone{
			Phase:  "last_update",
			Status: instructionStatusToString(st),
			At:     updated.Format(time.RFC3339),
			Note:   "最近一次状态更新时间（细分节点依赖设备上报）",
		})
	}
	return ms
}

// CancelDeviceInstructionIn 取消指令
type CancelDeviceInstructionIn struct {
	ID       int64
	Reason   string
	Operator string
}

// CancelDeviceInstruction 仅 pending 可取消
func (e *PlatformDeviceService) CancelDeviceInstruction(in *CancelDeviceInstructionIn) error {
	if e.Orm == nil {
		return fmt.Errorf("orm nil")
	}
	if in == nil || in.ID <= 0 {
		return ErrPlatformDeviceInvalid
	}
	op := strings.TrimSpace(in.Operator)
	if op == "" {
		op = "admin"
	}
	cr := strings.TrimSpace(in.Reason)

	var cur struct {
		DeviceID int64  `gorm:"column:device_id"`
		Sn       string `gorm:"column:sn"`
		Status   int16  `gorm:"column:status"`
	}
	if err := e.Orm.Table("device_instruction").Select("device_id, sn, status").Where("id = ?", in.ID).Take(&cur).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPlatformDeviceNotFound
		}
		return err
	}
	if cur.Status != InstrStatusPending {
		return fmt.Errorf("仅待执行(pending)状态的指令可取消，当前状态=%s", instructionStatusToString(cur.Status))
	}

	return e.Orm.Transaction(func(tx *gorm.DB) error {
		res := tx.Exec(`UPDATE device_instruction SET status = ?, completed_at = COALESCE(completed_at, CURRENT_TIMESTAMP), updated_at = CURRENT_TIMESTAMP WHERE id = ? AND status = ?`,
			InstrStatusCancelled, in.ID, InstrStatusPending)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return fmt.Errorf("取消失败：指令状态已变更")
		}
		msg := "管理员取消指令"
		if cr != "" {
			msg = msg + ": " + cr
		}
		if err := tx.Exec(`INSERT INTO device_event_log (device_id, sn, event_type, content, operator) VALUES (?,?,?,?,?)`,
			cur.DeviceID, cur.Sn, "instruction_cancelled", truncateEvent(msg), op).Error; err != nil {
			return err
		}
		fp := InstrStatusPending
		return InsertInstructionStateLog(tx, in.ID, &fp, InstrStatusCancelled, truncateEvent(msg), op)
	})
}

// ParseInstructionID 解析路径参数 id
func ParseInstructionID(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("empty id")
	}
	return strconv.ParseInt(s, 10, 64)
}
