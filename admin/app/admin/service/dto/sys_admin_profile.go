package dto

// SysAdminProfileUpdateReq 当前登录管理员的个人资料更新请求。
// 注意：该接口的目标 ID 从 Token 获取，不允许前端传 user_id。
//
// 为兼容前端“只改部分字段”，这里全部字段都设计为可选；
// service 层仍会做必要的业务校验（如昵称不能为空、角色至少 1 个等），因此 handler 会在缺省时回填旧值。
type SysAdminProfileUpdateReq struct {
	Nickname string `json:"nickname" binding:"omitempty,max=50" comment:"昵称"`
	RealName string `json:"real_name" binding:"omitempty,max=64" comment:"真实姓名"`
	Email    string `json:"email" binding:"omitempty,email,max=100" comment:"邮箱"`
	Phone    string `json:"phone" binding:"omitempty,len=11" comment:"手机号"`
	Avatar   string `json:"avatar" binding:"omitempty,max=255" comment:"头像 URL"`
	DeptId   *int   `json:"dept_id" comment:"部门 ID"`
	RoleIds  []int  `json:"role_ids" comment:"角色 ID 列表（仅超管可改）"`
	Status   string `json:"status" binding:"omitempty,oneof=1 2" comment:"状态（仅超管可改）"`
	Remark   string `json:"remark" binding:"omitempty,max=255" comment:"备注"`
}

