package dto

import (
	"go-admin/common/dto"
	common "go-admin/common/models"
)

// PlatformMemberListReq 会员列表查询请求
type PlatformMemberListReq struct {
	dto.Pagination  `search:"-"`
	Mobile          string `form:"mobile" search:"type:contains;column:mobile;table:users" comment:"手机号"`
	Nickname        string `form:"nickname" search:"type:contains;column:nickname;table:users" comment:"昵称"`
	MemberLevel     *int32 `form:"memberLevel" search:"type:exact;column:member_level;table:users" comment:"会员等级"`
	MemberStatus    *int32 `form:"memberStatus" search:"type:exact;column:status;table:user_member" comment:"会员状态"`
	StartTimeStart  string `form:"startTimeStart" search:"type:gte;column:created_at;table:user_member" comment:"开通会员时间开始"`
	StartTimeEnd    string `form:"startTimeEnd" search:"type:lte;column:created_at;table:user_member" comment:"开通会员时间结束"`
	ExpireTimeStart string `form:"expireTimeStart" search:"type:gte;column:expire_at;table:user_member" comment:"过期时间开始"`
	ExpireTimeEnd   string `form:"expireTimeEnd" search:"type:lte;column:expire_at;table:user_member" comment:"过期时间结束"`
	MemberOrder
}

type MemberOrder struct {
	MemberLevelOrder   string `search:"type:order;column:member_level;table:users" form:"memberLevelOrder"`
	StartTimeOrder     string `search:"type:order;column:created_at;table:user_member" form:"startTimeOrder"`
	ExpireTimeOrder    string `search:"type:order;column:expire_at;table:user_member" form:"expireTimeOrder"`
	RemainingDaysOrder string `search:"type:order;column:remaining_days;table:member_stats" form:"remainingDaysOrder"`
}

func (m *PlatformMemberListReq) GetNeedSearch() interface{} {
	return *m
}

// PlatformMemberListItem 会员列表项
type PlatformMemberListItem struct {
	UserId          int64  `json:"userId"`          // 用户 ID
	Mobile          string `json:"mobile"`          // 手机号（脱敏）
	Nickname        string `json:"nickname"`        // 用户昵称
	Avatar          string `json:"avatar"`          // 头像
	MemberLevel     int32  `json:"memberLevel"`     // 会员等级
	MemberLevelName string `json:"memberLevelName"` // 会员等级名称
	MemberStatus    int32  `json:"memberStatus"`    // 会员状态 0-正常 1-过期 2-冻结
	ExpireTime      int64  `json:"expireTime"`      // 会员过期时间戳
	RemainingDays   int64  `json:"remainingDays"`   // 剩余天数
	StartTime       int64  `json:"startTime"`       // 开通会员时间戳
	BindDeviceCount int64  `json:"bindDeviceCount"` // 绑定设备数量
	CreateTime      int64  `json:"createTime"`      // 开通会员时间戳
	UpdateTime      int64  `json:"updateTime"`      // 最后更新时间戳
}

// PlatformMemberListResp 会员列表响应
type PlatformMemberListResp struct {
	List     []PlatformMemberListItem `json:"list"`     // 会员列表
	Total    int64                    `json:"total"`    // 总条数
	Page     int                      `json:"page"`     // 当前页码
	PageSize int                      `json:"pageSize"` // 每页条数
}

// PlatformMemberDetailReq 会员详情查询请求
type PlatformMemberDetailReq struct {
	UserId int64 `uri:"userId" validate:"required"`
}

func (s *PlatformMemberDetailReq) GetId() interface{} {
	return s.UserId
}

// PlatformMemberDetailResp 会员详情响应
type PlatformMemberDetailResp struct {
	// 用户基础信息
	UserId     int64  `json:"userId"`     // 用户 ID
	Mobile     string `json:"mobile"`     // 手机号（脱敏）
	Nickname   string `json:"nickname"`   // 用户昵称
	Avatar     string `json:"avatar"`     // 头像
	UserStatus int32  `json:"userStatus"` // 用户账号状态 0-禁用 1-正常

	// 会员核心信息
	MemberLevel     int32  `json:"memberLevel"`     // 会员等级
	MemberLevelName string `json:"memberLevelName"` // 会员等级名称
	MemberStatus    int32  `json:"memberStatus"`    // 会员状态 0-正常 1-过期 2-冻结 3-未开通
	ExpireTime      int64  `json:"expireTime"`      // 会员过期时间戳
	RemainingDays   int64  `json:"remainingDays"`   // 剩余天数
	StartTime       int64  `json:"startTime"`       // 开通会员时间戳
	LastRenewalTime int64  `json:"lastRenewalTime"` // 最后续费时间
	IsAutoRenew     bool   `json:"isAutoRenew"`     // 是否自动续费
	IsExpiringSoon  bool   `json:"isExpiringSoon"`  // 是否即将到期（7 天内）

	// 会员权益信息
	DeviceBindLimit   int64         `json:"deviceBindLimit"`   // 绑定设备上限
	CurrentBindCount  int64         `json:"currentBindCount"`  // 当前绑定数量
	AvailableBenefits []BenefitInfo `json:"availableBenefits"` // 可用权益列表
	UsedBenefits      []BenefitInfo `json:"usedBenefits"`      // 已用权益列表

	// 购买/订单记录
	OrderRecords []OrderRecord `json:"orderRecords"` // 订单记录

	// 管理员操作记录
	OperateRecords []OperateRecord `json:"operateRecords"` // 操作记录

	// 其他统计
	BindDeviceCount int64              `json:"bindDeviceCount"` // 绑定设备数量
	LevelConfig     *MemberLevelConfig `json:"levelConfig"`     // 会员等级配置
	RenewalRecords  []RenewalRecord    `json:"renewalRecords"`  // 续费记录
}

// MemberLevelConfig 会员等级配置
type MemberLevelConfig struct {
	Level           int32  `json:"level"`           // 等级
	Name            string `json:"name"`            // 等级名称
	Color           string `json:"color"`           // 颜色标签
	Description     string `json:"description"`     // 描述
	DeviceBindLimit int64  `json:"deviceBindLimit"` // 绑定设备上限
}

// RenewalRecord 续费记录
type RenewalRecord struct {
	RenewalTime int64 `json:"renewalTime"` // 续费时间戳
	Days        int64 `json:"days"`        // 续费天数
	Amount      int64 `json:"amount"`      // 续费金额（分）
}

// BenefitInfo 权益信息
type BenefitInfo struct {
	BenefitName    string `json:"benefitName"`    // 权益名称
	BenefitCode    string `json:"benefitCode"`    // 权益编码
	Status         int32  `json:"status"`         // 状态 0-可用 1-已用完 2-已过期
	TotalCount     int64  `json:"totalCount"`     // 总次数
	UsedCount      int64  `json:"usedCount"`      // 已用次数
	RemainingCount int64  `json:"remainingCount"` // 剩余次数
	ExpireTime     int64  `json:"expireTime"`     // 过期时间戳
}

// OrderRecord 订单记录
type OrderRecord struct {
	OrderId     string `json:"orderId"`     // 订单 ID
	PayAmount   int64  `json:"payAmount"`   // 支付金额（分）
	PayType     int32  `json:"payType"`     // 支付方式 1-微信 2-支付宝 3-银行卡
	OrderTime   int64  `json:"orderTime"`   // 订单时间戳
	OrderStatus int32  `json:"orderStatus"` // 订单状态 0-待支付 1-已支付 2-已取消 3-已退款
	OrderType   int32  `json:"orderType"`   // 订单类型 1-新购 2-续费 3-升级
	MemberLevel int32  `json:"memberLevel"` // 会员等级
	MemberDays  int64  `json:"memberDays"`  // 会员天数
}

// OperateRecord 管理员操作记录
type OperateRecord struct {
	OperateAdmin     int64  `json:"operateAdmin"`     // 操作管理员 ID
	OperateAdminName string `json:"operateAdminName"` // 操作管理员姓名
	OperateType      int32  `json:"operateType"`      // 操作类型 1-修改等级 2-冻结 3-解冻 4-修改过期时间
	OldLevel         int32  `json:"oldLevel"`         // 原等级
	NewLevel         int32  `json:"newLevel"`         // 新等级
	OldExpireTime    int64  `json:"oldExpireTime"`    // 原过期时间
	NewExpireTime    int64  `json:"newExpireTime"`    // 新过期时间
	OperateTime      int64  `json:"operateTime"`      // 操作时间戳
	Remark           string `json:"remark"`           // 备注
}

// PlatformMemberUpdateReq 更新会员信息请求
type PlatformMemberUpdateReq struct {
	UserId        int64  `json:"userId" validate:"required"`
	MemberLevel   int32  `json:"memberLevel" validate:"required,oneof=0 1 2 3"`
	ExpireTime    int64  `json:"expireTime" validate:"required"`
	Days          int64  `json:"days"`          // 有效期天数（与 expireTime 二选一）
	Remark        string `json:"remark"`        // 操作备注（必填）
	OperatorId    int64  `json:"operatorId"`    // 操作管理员 ID（从 JWT 获取）
	OperationType int32  `json:"operationType"` // 操作类型 1-开通 2-续费 3-升级 4-降级 5-延长
	common.ControlBy
}

func (s *PlatformMemberUpdateReq) GetId() interface{} {
	return s.UserId
}

// PlatformMemberUpdateResp 更新会员信息响应
type PlatformMemberUpdateResp struct {
	UserId          int64  `json:"userId"`          // 用户 ID
	MemberLevel     int32  `json:"memberLevel"`     // 会员等级
	MemberLevelName string `json:"memberLevelName"` // 会员等级名称
	MemberStatus    int32  `json:"memberStatus"`    // 会员状态
	ExpireTime      int64  `json:"expireTime"`      // 会员过期时间戳
	RemainingDays   int64  `json:"remainingDays"`   // 剩余天数
	OperationType   int32  `json:"operationType"`   // 操作类型
	OldLevel        int32  `json:"oldLevel"`        // 原等级
	OldExpireTime   int64  `json:"oldExpireTime"`   // 原过期时间
	Remark          string `json:"remark"`          // 操作备注
	UpdateTime      int64  `json:"updateTime"`      // 更新时间戳
}

// PlatformMemberFreezeReq 冻结会员请求
type PlatformMemberFreezeReq struct {
	UserId       int64  `json:"userId" validate:"required"`
	FreezeReason string `json:"freezeReason" validate:"required"` // 冻结原因（必填）
	OperatorId   int64  `json:"operatorId"`                       // 操作管理员 ID
	common.ControlBy
}

func (s *PlatformMemberFreezeReq) GetId() interface{} {
	return s.UserId
}

// PlatformMemberFreezeResp 冻结会员响应
type PlatformMemberFreezeResp struct {
	UserId          int64  `json:"userId"`          // 用户 ID
	MemberLevel     int32  `json:"memberLevel"`     // 会员等级
	MemberLevelName string `json:"memberLevelName"` // 会员等级名称
	MemberStatus    int32  `json:"memberStatus"`    // 会员状态（2-冻结）
	ExpireTime      int64  `json:"expireTime"`      // 会员过期时间戳
	FreezeTime      int64  `json:"freezeTime"`      // 冻结时间戳
	FreezeReason    string `json:"freezeReason"`    // 冻结原因
	OperatorId      int64  `json:"operatorId"`      // 操作管理员 ID
	UpdateTime      int64  `json:"updateTime"`      // 更新时间戳
}

// PlatformMemberUnfreezeReq 解冻会员请求
type PlatformMemberUnfreezeReq struct {
	UserId         int64  `json:"userId" validate:"required"`
	UnfreezeReason string `json:"unfreezeReason" validate:"required"` // 解冻原因（必填）
	OperatorId     int64  `json:"operatorId"`                         // 操作管理员 ID
	common.ControlBy
}

func (s *PlatformMemberUnfreezeReq) GetId() interface{} {
	return s.UserId
}

// PlatformMemberUnfreezeResp 解冻会员响应
type PlatformMemberUnfreezeResp struct {
	UserId          int64  `json:"userId"`          // 用户 ID
	MemberLevel     int32  `json:"memberLevel"`     // 会员等级
	MemberLevelName string `json:"memberLevelName"` // 会员等级名称
	MemberStatus    int32  `json:"memberStatus"`    // 会员状态（0-正常）
	ExpireTime      int64  `json:"expireTime"`      // 会员过期时间戳
	UnfreezeTime    int64  `json:"unfreezeTime"`    // 解冻时间戳
	UnfreezeReason  string `json:"unfreezeReason"`  // 解冻原因
	OperatorId      int64  `json:"operatorId"`      // 操作管理员 ID
	UpdateTime      int64  `json:"updateTime"`      // 更新时间戳
}

// MemberRightConfig 会员权益配置项
type MemberRightConfig struct {
	RightKey   string      `json:"rightKey"`   // 权益键
	RightName  string      `json:"rightName"`  // 权益名称
	RightValue interface{} `json:"rightValue"` // 权益值
	RightType  string      `json:"rightType"`  // 权益类型：int/bool/string
}

// PlatformMemberRightConfigReq 会员权益配置请求
type PlatformMemberRightConfigReq struct {
	LevelId    int32               `json:"levelId" validate:"required"` // 会员等级 ID
	LevelName  string              `json:"levelName"`                   // 会员等级名称
	Status     int32               `json:"status" validate:"oneof=0 1"` // 状态 0-禁用 1-启用
	Rights     []MemberRightConfig `json:"rights"`                      // 权益配置列表
	Remark     string              `json:"remark"`                      // 备注
	OperatorId int64               `json:"operatorId"`                  // 操作管理员 ID
	common.ControlBy
}

func (s *PlatformMemberRightConfigReq) GetId() interface{} {
	return s.LevelId
}

// PlatformMemberRightConfigResp 会员权益配置响应
type PlatformMemberRightConfigResp struct {
	LevelId      int32               `json:"levelId"`      // 会员等级 ID
	LevelName    string              `json:"levelName"`    // 会员等级名称
	Status       int32               `json:"status"`       // 状态
	Rights       []MemberRightConfig `json:"rights"`       // 权益配置列表
	UpdateTime   int64               `json:"updateTime"`   // 更新时间戳
	OperatorId   int64               `json:"operatorId"`   // 操作管理员 ID
	OperatorName string              `json:"operatorName"` // 操作人名称
}
