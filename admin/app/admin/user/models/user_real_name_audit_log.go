package models

import "time"

// UserRealNameAuditLog 实名认证审核日志
type UserRealNameAuditLog struct {
	ID         int64     `json:"id"`
	UserID     int64     `json:"user_id"`
	AuthID     int64     `json:"auth_id"`
	OperatorID int64     `json:"operator_id"` // 操作员 ID
	Action     string    `json:"action"`      // 操作类型：audit
	OldStatus  int16     `json:"old_status"`  // 审核前状态
	NewStatus  int16     `json:"new_status"`  // 审核后状态
	Remark     string    `json:"remark"`      // 审核备注
	CreatedAt  time.Time `json:"created_at"`
}
