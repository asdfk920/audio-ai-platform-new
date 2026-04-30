package apis

import (
	"encoding/json"
	"errors"
	"fmt"

	"go-admin/app/admin/user/service"
	"go-admin/app/admin/user/service/dto"
	"go-admin/common/actions"

	"github.com/gin-gonic/gin"
	"github.com/go-admin-team/go-admin-core/sdk/api"
	jwt "github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth/user"
)

type PlatformUserList struct {
	api.Api
}

const platformUserMaxPageSize = 100

// GetPlatformUserList 平台用户列表（分页 + 多条件筛选）
// @Summary 平台用户列表
// @Tags 用户管理
// @Param page query int false "页码，默认 1"
// @Param pageSize query int false "每页条数，默认 10，最大 100"
// @Param mobile query string false "手机号（模糊搜索）"
// @Param nickname query string false "昵称（模糊搜索）"
// @Param email query string false "邮箱（模糊搜索）"
// @Param status query int false "账号状态 0 禁用 1 正常"
// @Param memberLevel query int false "会员等级"
// @Param realNameStatus query int false "实名状态"
// @Param registerTimeStart query string false "注册时间开始 (Unix 时间戳)"
// @Param registerTimeEnd query string false "注册时间结束 (Unix 时间戳)"
// @Success 200 {object} response.Response{data=dto.PlatformUserListResp}
// @Router /api/v1/platform-user/list [get]
// @Security Bearer
func (e PlatformUserList) GetPlatformUserList(c *gin.Context) {
	// 1. 上下文判空保护
	if c == nil {
		e.Error(500, errors.New("context is nil"), "请求上下文异常")
		return
	}

	// 2. 服务初始化
	s := service.PlatformUserListService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Logger.Errorf("服务初始化失败：%v", err)
		e.Error(500, err, "服务初始化失败")
		return
	}

	// 3. 解析请求参数（去掉 gender=&status= 等空串，避免 Gin 将 *int32 绑定失败）
	q := c.Request.URL.Query()
	for _, key := range []string{"gender", "status", "realNameStatus", "memberLevel", "userId"} {
		if q.Get(key) == "" {
			q.Del(key)
		}
	}
	c.Request.URL.RawQuery = q.Encode()

	var req dto.PlatformUserListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		e.Logger.Warnf("参数解析失败：%v", err)
		e.Error(400, err, "参数解析失败")
		return
	}

	// 4. 分页参数校验
	req.PageIndex = req.GetPageIndex()
	req.PageSize = req.GetPageSize()
	if req.PageSize > platformUserMaxPageSize {
		req.PageSize = platformUserMaxPageSize
	}

	// 5. 获取数据权限（简化版，暂不启用权限控制）
	p := &actions.DataPermission{}

	// 6. 调用服务层查询
	var list []dto.PlatformUserListItem
	var total int64
	err := s.GetPage(&req, p, &list, &total)
	if err != nil {
		e.Logger.Errorf("查询用户列表失败：%v", err)
		e.Error(500, err, "查询失败")
		return
	}

	// 7. 封装返回数据
	resp := dto.PlatformUserListResp{
		List:     list,
		Total:    total,
		Page:     req.PageIndex,
		PageSize: req.PageSize,
	}

	// 8. 返回成功结果
	e.OK(resp, "查询成功")
}

// GetPlatformUserInfo 获取用户详情
// @Summary 获取用户详情（多维度关联查询）
// @Tags 用户管理
// @Param userId path int true "用户 ID"
// @Success 200 {object} response.Response{data=dto.PlatformUserInfoResp}
// @Router /api/v1/platform-user/{userId} [get]
// @Security Bearer
func (e PlatformUserList) GetPlatformUserInfo(c *gin.Context) {
	// 1. 上下文判空保护
	if c == nil {
		e.Error(500, errors.New("context is nil"), "请求上下文异常")
		return
	}

	// 2. 服务初始化
	s := service.PlatformUserListService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Logger.Errorf("服务初始化失败：%v", err)
		e.Error(500, err, "服务初始化失败")
		return
	}

	// 3. 解析路径参数
	var req dto.PlatformUserGetInfoReq
	if err := c.ShouldBindUri(&req); err != nil {
		e.Logger.Warnf("参数解析失败：%v", err)
		e.Error(400, err, "参数解析失败")
		return
	}

	// 4. 参数合法性校验
	if req.UserId <= 0 {
		e.Error(400, errors.New("用户 ID 格式错误"), "参数异常")
		return
	}

	// 5. 记录操作日志
	e.Logger.Infof("请求查询用户 %d 详情", req.UserId)

	// 6. 调用服务层查询（多维度关联查询）
	info, err := s.Get(req.UserId)
	if err != nil {
		e.Logger.Errorf("查询用户详情失败：%v", err)
		if err.Error() == "用户不存在" {
			e.Error(404, err, "用户不存在")
			return
		}
		e.Error(500, err, "查询失败")
		return
	}

	// 7. 记录操作日志
	e.Logger.Infof("成功查询用户 %d 详情，返回 %d 个设备信息，%d 条登录记录",
		req.UserId, len(info.DeviceList), len(info.RecentLogins))

	// 8. 返回成功结果
	e.OK(info, "查询成功")
}

// CreatePlatformUser 创建平台用户（仅超级管理员可用；结果仅写入 users 表）
//
// 权限与落地规则：
//  1. 入口前置校验：调用方 JWT 对应的 sys_admin 行必须启用且 role_code = 'super_admin'；
//  2. 本接口忽略任何 role_ids 取值，服务层强制落为 roles.slug = 'user'（普通用户）；
//  3. 成功后数据仅写入 public.users + public.user_role_rel，绝不访问 sys_admin。
//
// @Summary 新增平台用户（超级管理员）
// @Tags 用户管理
// @Param data body dto.PlatformUserCreateReq true "用户信息"
// @Success 200 {object} response.Response
// @Router /api/v1/platform-user [post]
// @Security Bearer
func (e PlatformUserList) CreatePlatformUser(c *gin.Context) {
	if c == nil {
		e.Error(500, errors.New("context is nil"), "请求上下文异常")
		return
	}

	s := service.PlatformUserListService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Logger.Errorf("服务初始化失败：%v", err)
		e.Error(500, err, "服务初始化失败")
		return
	}

	// 1) 超级管理员校验：从 JWT 解出 adminID，再到 sys_admin 检查 role_code
	adminID := int64(user.GetUserId(c))
	if adminID <= 0 {
		e.Error(401, errors.New("unauthenticated"), "请先登录")
		return
	}
	isSuper, err := s.IsCurrentAdminSuperAdmin(adminID)
	if err != nil {
		e.Logger.Errorf("super_admin 校验失败：%v", err)
		e.Error(500, err, "权限校验失败")
		return
	}
	if !isSuper {
		e.Error(403, service.ErrPlatformUserForbidden, "仅超级管理员可新增平台用户")
		return
	}

	// 2) 解析请求参数
	var req dto.PlatformUserCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		e.Logger.Warnf("参数解析失败：%v", err)
		e.Error(400, err, "参数解析失败")
		return
	}
	if req.Mobile == "" && req.Email == "" {
		e.Error(400, errors.New("手机号和邮箱不能同时为空"), "手机号和邮箱不能同时为空")
		return
	}

	// 3) 调用服务层：service.Create 内部会忽略 RoleIds 并强制落为普通用户
	if err := s.Create(&req); err != nil {
		e.Logger.Errorf("创建用户失败：%v", err)
		switch {
		case errors.Is(err, service.ErrPlatformUserInvalidUsername):
			e.Error(400, err, "用户名格式不正确（字母开头，3~64 位，仅限字母数字 . _ -）")
		case errors.Is(err, service.ErrPlatformUserInvalidPassword):
			e.Error(400, err, "密码长度至少 6 位")
		case errors.Is(err, service.ErrPlatformUserDuplicate):
			e.Error(400, err, "用户名/手机号/邮箱已存在")
		case errors.Is(err, service.ErrPlatformUserRoleNotFound):
			e.Error(500, err, "普通用户角色未配置，请联系管理员")
		default:
			e.Error(500, err, "创建失败")
		}
		return
	}

	e.OK(nil, "创建成功")
}

// UpdatePlatformUser 更新用户
//
// URI 参数与请求体都会绑定到同一个 DTO：
//   - URI path 里的 userId 通过 PlatformUserUpdateReq.UserId 的 `uri:"userId"` 标签解析；
//   - 旧实现传的是 &req.UserId（*int64），gin 的 ShouldBindUri 需要一个带 uri tag 的 struct 指针，
//     导致 UserId 始终为 0，服务层一律报「用户不存在」。此处改为传 &req 才能正确绑定。
//
// @Summary 更新用户
// @Tags 用户管理
// @Param userId path int true "用户 ID"
// @Param data body dto.PlatformUserUpdateReq true "用户信息"
// @Success 200 {object} response.Response
// @Router /api/v1/platform-user/{userId} [put]
// @Security Bearer
func (e PlatformUserList) UpdatePlatformUser(c *gin.Context) {
	if c == nil {
		e.Error(500, errors.New("context is nil"), "请求上下文异常")
		return
	}

	s := service.PlatformUserListService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Logger.Errorf("服务初始化失败：%v", err)
		e.Error(500, err, "服务初始化失败")
		return
	}

	var req dto.PlatformUserUpdateReq
	if err := c.ShouldBindUri(&req); err != nil {
		e.Logger.Warnf("路径参数解析失败：%v", err)
		e.Error(400, err, "路径参数解析失败")
		return
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		e.Logger.Warnf("请求体解析失败：%v", err)
		e.Error(400, err, "请求体解析失败")
		return
	}
	if req.UserId <= 0 {
		e.Error(400, errors.New("invalid userId"), "用户 ID 无效")
		return
	}

	if err := s.Update(&req); err != nil {
		e.Logger.Errorf("更新用户失败：%v", err)
		switch {
		case errors.Is(err, service.ErrPlatformUserNotFound):
			e.Error(404, err, "用户不存在")
		case errors.Is(err, service.ErrPlatformUserDuplicate):
			e.Error(400, err, "手机号/邮箱已被其他用户使用")
		default:
			e.Error(500, err, "更新失败")
		}
		return
	}

	e.OK(nil, "更新成功")
}

// SetPlatformUserRoles 覆盖式设置平台用户的角色集合。
//
// 权限：仅「超级管理员」可调用。
// 请求体：{"role_ids": [1, 2, ...]}，可为空数组代表撤销所有角色。
//
// @Summary 设置平台用户角色
// @Tags 用户管理
// @Param userId path int true "用户 ID"
// @Param data body object true "{role_ids: int[]}"
// @Success 200 {object} response.Response
// @Router /api/v1/platform-user/{userId}/roles [put]
// @Security Bearer
func (e PlatformUserList) SetPlatformUserRoles(c *gin.Context) {
	if c == nil {
		e.Error(500, errors.New("context is nil"), "请求上下文异常")
		return
	}

	s := service.PlatformUserListService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Logger.Errorf("服务初始化失败：%v", err)
		e.Error(500, err, "服务初始化失败")
		return
	}

	adminID := int64(user.GetUserId(c))
	if adminID <= 0 {
		e.Error(401, errors.New("unauthorized"), "未登录")
		return
	}
	if isSuper, err := s.IsCurrentAdminSuperAdmin(adminID); err != nil {
		e.Logger.Errorf("校验超级管理员身份失败：%v", err)
		e.Error(500, err, "权限校验失败")
		return
	} else if !isSuper {
		e.Error(403, service.ErrPlatformUserForbidden, "仅超级管理员可调整角色")
		return
	}

	var uriReq struct {
		UserId int64 `uri:"userId" binding:"required"`
	}
	if err := c.ShouldBindUri(&uriReq); err != nil {
		e.Error(400, err, "路径参数解析失败")
		return
	}
	if uriReq.UserId <= 0 {
		e.Error(400, errors.New("invalid userId"), "用户 ID 无效")
		return
	}

	var body struct {
		RoleIds []int64 `json:"role_ids"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		e.Error(400, err, "请求体解析失败")
		return
	}

	if err := s.SetUserRoles(uriReq.UserId, body.RoleIds); err != nil {
		e.Logger.Errorf("设置用户角色失败：%v", err)
		switch {
		case errors.Is(err, service.ErrPlatformUserNotFound):
			e.Error(404, err, "用户不存在")
		case errors.Is(err, service.ErrPlatformUserRoleNotFound):
			e.Error(400, err, "存在未知的角色 ID")
		default:
			e.Error(500, err, "设置角色失败")
		}
		return
	}

	e.OK(nil, "角色已更新")
}

// DeletePlatformUser 删除用户
// @Summary 删除用户
// @Tags 用户管理
// @Param userId path int true "用户 ID"
// @Success 200 {object} response.Response
// @Router /api/v1/platform-user/{userId} [delete]
// @Security Bearer
func (e PlatformUserList) DeletePlatformUser(c *gin.Context) {
	// 1. 上下文判空保护
	if c == nil {
		e.Error(500, errors.New("context is nil"), "请求上下文异常")
		return
	}

	// 2. 服务初始化
	s := service.PlatformUserListService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Logger.Errorf("服务初始化失败：%v", err)
		e.Error(500, err, "服务初始化失败")
		return
	}

	// 3. 解析路径参数
	var req struct {
		UserId int64 `uri:"userId"`
	}
	if err := c.ShouldBindUri(&req); err != nil {
		e.Logger.Warnf("路径参数解析失败：%v", err)
		e.Error(400, err, "路径参数解析失败")
		return
	}

	// 4. 调用服务层删除
	if err := s.Delete(req.UserId); err != nil {
		e.Logger.Errorf("删除用户失败：%v", err)
		e.Error(500, err, "删除失败")
		return
	}

	// 5. 返回成功结果
	e.OK(nil, "删除成功")
}

// GetInfo 获取当前登录用户信息
// @Summary 获取用户信息
// @Tags 用户管理
// @Success 200 {object} response.Response{data=dto.UserInfo}
// @Router /api/v1/getinfo [get]
// @Security Bearer
func (e *PlatformUserList) GetInfo(c *gin.Context) {
	// 1. 上下文判空保护
	if c == nil {
		e.Error(500, errors.New("context is nil"), "请求上下文异常")
		return
	}

	// 2. 服务初始化
	s := service.PlatformUserListService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		if e.Logger != nil {
			e.Logger.Errorf("服务初始化失败：%v", err)
		}
		e.Error(500, err, "服务初始化失败")
		return
	}

	// 3. 从 JWT 获取用户信息
	claims := jwt.ExtractClaims(c)
	if e.Logger != nil {
		e.Logger.Infof("JWT claims: %+v", claims)
	}

	// 尝试不同的类型转换
	var userId int64
	idValue := claims["identity"]
	if e.Logger != nil {
		e.Logger.Infof("identity 值：%+v, 类型：%T", idValue, idValue)
	}

	if id, ok := idValue.(float64); ok {
		userId = int64(id)
		if e.Logger != nil {
			e.Logger.Infof("转换为 float64: %f -> %d", id, userId)
		}
	} else if id, ok := idValue.(int64); ok {
		userId = id
		if e.Logger != nil {
			e.Logger.Infof("转换为 int64: %d", id)
		}
	} else if id, ok := idValue.(int); ok {
		userId = int64(id)
		if e.Logger != nil {
			e.Logger.Infof("转换为 int: %d -> %d", id, userId)
		}
	} else if idStr, ok := idValue.(string); ok {
		// 如果是字符串，尝试转换
		fmt.Sscanf(idStr, "%d", &userId)
		if e.Logger != nil {
			e.Logger.Infof("转换为 string: %s -> %d", idStr, userId)
		}
	} else if idNum, ok := idValue.(json.Number); ok {
		// json.Number 类型
		idInt64, err := idNum.Int64()
		if err != nil {
			if e.Logger != nil {
				e.Logger.Warnf("json.Number 转换失败：%v", err)
			}
		} else {
			userId = idInt64
			if e.Logger != nil {
				e.Logger.Infof("转换为 json.Number: %s -> %d", idNum.String(), userId)
			}
		}
	} else {
		if e.Logger != nil {
			e.Logger.Warnf("无法转换的类型：%T", idValue)
		}
	}

	if e.Logger != nil {
		e.Logger.Infof("最终 userId: %d", userId)
	}

	if userId <= 0 {
		if e.Logger != nil {
			e.Logger.Warnf("未获取到用户信息，claims: %+v", claims)
		}
		e.Error(401, errors.New("未获取到用户信息"), "未授权")
		return
	}

	// 4. 查询用户详细信息
	user, err := s.GetUserInfo(int64(userId))
	if err != nil {
		if e.Logger != nil {
			e.Logger.Errorf("查询用户信息失败：%v", err)
		}
		e.Error(500, err, "查询用户信息失败")
		return
	}

	// 5. 封装返回数据（符合前端期望的格式）
	resp := dto.UserInfo{
		Roles:       user.Roles,
		Name:        user.Name,
		Avatar:      user.Avatar,
		Intro:       user.Intro,
		Permissions: user.Permissions,
	}

	// 6. 返回成功结果
	e.OK(resp, "查询成功")
}

// UpdateUserStatus 禁用/启用用户
// @Summary 禁用/启用用户
// @Tags 用户管理
// @Param body body dto.UpdateUserStatusReq true "用户状态更新请求"
// @Success 200 {object} response.Response{data=dto.UpdateUserStatusResp}
// @Router /api/v1/platform-user/status [put]
// @Security Bearer
func (e *PlatformUserList) UpdateUserStatus(c *gin.Context) {
	// 1. 上下文判空保护
	if c == nil {
		e.Error(500, errors.New("context is nil"), "请求上下文异常")
		return
	}

	// 2. 服务初始化
	s := service.PlatformUserListService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		if e.Logger != nil {
			e.Logger.Errorf("服务初始化失败：%v", err)
		}
		e.Error(500, err, "服务初始化失败")
		return
	}

	// 3. 解析请求参数（兼容 userId / user_id；避免大整数或部分客户端把数字序列化成字符串导致绑定失败）
	var req dto.UpdateUserStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		e.Logger.Warnf("参数解析失败：%v", err)
		e.Error(400, err, "参数解析失败")
		return
	}
	if req.UserId <= 0 {
		e.Error(400, errors.New("invalid user id"), "用户 ID 无效")
		return
	}

	// 4. 从 JWT 获取当前管理员信息
	claims := jwt.ExtractClaims(c)
	adminIdValue := claims["identity"]
	var adminId int64
	if id, ok := adminIdValue.(float64); ok {
		adminId = int64(id)
	} else if id, ok := adminIdValue.(int64); ok {
		adminId = id
	} else if id, ok := adminIdValue.(int); ok {
		adminId = int64(id)
	} else if idStr, ok := adminIdValue.(string); ok {
		fmt.Sscanf(idStr, "%d", &adminId)
	} else if idNum, ok := adminIdValue.(json.Number); ok {
		idInt64, _ := idNum.Int64()
		adminId = idInt64
	}

	// 5. 获取客户端 IP
	clientIP := c.ClientIP()

	// 6. 调用服务层更新用户状态
	resp, err := s.UpdateUserStatus(&req, adminId, clientIP)
	if err != nil {
		e.Logger.Errorf("更新用户状态失败：%v", err)
		// 根据错误类型返回不同的错误码
		if err.Error() == "禁止操作超级管理员账号" {
			e.Error(403, err, err.Error())
		} else if err.Error() == "禁止操作自身账号" {
			e.Error(400, err, err.Error())
		} else if err.Error() == "用户不存在" {
			e.Error(404, err, err.Error())
		} else {
			e.Error(500, err, "更新用户状态失败")
		}
		return
	}

	// 7. 返回成功结果
	e.OK(resp, "操作成功")
}

// UpdateUserVipLevel 修改用户会员等级
// @Summary 修改用户会员等级
// @Tags 用户管理
// @Param body body dto.UpdateUserVipLevelReq true "用户会员等级更新请求"
// @Success 200 {object} response.Response{data=dto.UpdateUserVipLevelResp}
// @Router /api/v1/platform-user/vip-level [put]
// @Security Bearer
func (e *PlatformUserList) UpdateUserVipLevel(c *gin.Context) {
	// 1. 上下文判空保护
	if c == nil {
		e.Error(500, errors.New("context is nil"), "请求上下文异常")
		return
	}

	// 2. 服务初始化
	s := service.PlatformUserListService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		if e.Logger != nil {
			e.Logger.Errorf("服务初始化失败：%v", err)
		}
		e.Error(500, err, "服务初始化失败")
		return
	}

	// 3. 解析请求参数
	var req dto.UpdateUserVipLevelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		e.Logger.Warnf("参数解析失败：%v", err)
		e.Error(400, err, "参数解析失败")
		return
	}

	// 4. 从 JWT 获取当前管理员信息
	claims := jwt.ExtractClaims(c)
	adminIdValue := claims["identity"]
	var adminId int64
	if id, ok := adminIdValue.(float64); ok {
		adminId = int64(id)
	} else if id, ok := adminIdValue.(int64); ok {
		adminId = id
	} else if id, ok := adminIdValue.(int); ok {
		adminId = int64(id)
	} else if idStr, ok := adminIdValue.(string); ok {
		fmt.Sscanf(idStr, "%d", &adminId)
	} else if idNum, ok := adminIdValue.(json.Number); ok {
		idInt64, _ := idNum.Int64()
		adminId = idInt64
	}

	// 5. 获取客户端 IP
	clientIP := c.ClientIP()

	// 6. 调用服务层更新用户会员等级
	resp, err := s.UpdateUserVipLevel(&req, adminId, clientIP)
	if err != nil {
		e.Logger.Errorf("更新用户会员等级失败：%v", err)
		// 根据错误类型返回不同的错误码
		if err.Error() == "禁止操作超级管理员账号" {
			e.Error(403, err, err.Error())
		} else if err.Error() == "禁止操作自身账号" {
			e.Error(400, err, err.Error())
		} else if err.Error() == "用户不存在" {
			e.Error(404, err, err.Error())
		} else if err.Error() == "用户已禁用" {
			e.Error(400, err, err.Error())
		} else if err.Error() == "等级非法" {
			e.Error(400, err, err.Error())
		} else {
			e.Error(500, err, "更新用户会员等级失败")
		}
		return
	}

	// 7. 返回成功结果
	e.OK(resp, "操作成功")
}
