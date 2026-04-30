package models

import "time"

// UserRealNameAuth 实名认证审核记录
type UserRealNameAuth struct {
	ID                    int64      `json:"id"`
	UserID                int64      `json:"user_id"`
	CertType              int16      `json:"cert_type"`              // 证件类型：1 身份证 2 统一社会信用代码
	RealNameMasked        string     `json:"real_name_masked"`       // 脱敏姓名
	IdNumberEncrypted     string     `json:"id_number_encrypted"`    // 加密的身份证号
	IdNumberLast4         string     `json:"id_number_last4"`        // 身份证号后 4 位
	IdPhotoRef            *string    `json:"id_photo_ref"`           // 证件照片引用
	IdCardFrontRef        *string    `json:"id_card_front_ref"`      // 身份证人像面引用
	IdCardBackRef         *string    `json:"id_card_back_ref"`       // 身份证国徽面引用
	FaceDataRef           *string    `json:"face_data_ref"`          // 人脸数据引用
	AuthStatus            int16      `json:"auth_status"`            // 审核状态：10 待三方核验 11 三方通过 12 三方失败 20 待人工审核 21 人工通过 22 人工驳回
	ThirdPartyFlowNo      *string    `json:"third_party_flow_no"`    // 三方流水号
	ThirdPartyChannel     *string    `json:"third_party_channel"`    // 三方渠道
	ThirdPartyRawResponse *string    `json:"third_party_raw_response"` // 三方原始响应
	FailReason            *string    `json:"fail_reason"`            // 失败原因
	ReviewerNote          *string    `json:"reviewer_note"`          // 审核备注
	ReviewedAt            *time.Time `json:"reviewed_at"`            // 审核时间
	ReviewedBy            *string    `json:"reviewed_by"`            // 审核人 ID
	DeviceInfo            *string    `json:"device_info"`            // 设备信息
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

// UserRealNameAuditReq 实名认证审核请求
type UserRealNameAuditReq struct {
	UserID       int64  `json:"user_id" binding:"required"`   // 用户 ID
	AuditResult  int16  `json:"audit_result" binding:"required"` // 审核结果：1 通过 2 驳回
	AuditRemark  string `json:"audit_remark"` // 驳回时必填，由接口层校验
	OperatorID   int64  `json:"operator_id"`                  // 操作员 ID（从 token 获取）
}

// UserRealNameAuditResp 实名认证审核响应
type UserRealNameAuditResp struct {
	UserID         int64  `json:"user_id"`
	RealNameStatus int16  `json:"real_name_status"` // 最新实名状态
	Message        string `json:"message"`
}

// UserRealNameListReq 实名认证列表请求
type UserRealNameListReq struct {
	Page       int `json:"page" binding:"required"`
	PageSize   int `json:"page_size" binding:"required"`
	AuthStatus int `json:"auth_status"` // 可选：按状态筛选
}

// UserRealNameListResp 实名认证列表响应
type UserRealNameListResp struct {
	List  []UserRealNameListItem `json:"list"`
	Total int64                  `json:"total"`
}

// UserRealNameListItem 实名认证列表项
type UserRealNameListItem struct {
	ID             int64     `json:"id"`
	UserID         int64     `json:"user_id"`
	Nickname       *string   `json:"nickname"`
	Email          *string   `json:"email"`
	Mobile         *string   `json:"mobile"`
	RealNameMasked string    `json:"real_name_masked"`
	IdNumberLast4  string    `json:"id_number_last4"`
	CertType       int16     `json:"cert_type"`
	AuthStatus     int16     `json:"auth_status"`
	AuthStatusText string    `json:"auth_status_text"`
	CreatedAt      time.Time `json:"created_at"`
}
