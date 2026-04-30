package apis

import (
	"errors"

	"go-admin/app/admin/user/service"
	"go-admin/app/admin/user/service/dto"
	"go-admin/common/actions"

	"github.com/gin-gonic/gin"
	"github.com/go-admin-team/go-admin-core/sdk/api"
)

type PlatformMemberList struct {
	api.Api
}

const platformMemberMaxPageSize = 100

// GetPlatformMemberList 会员列表（分页 + 多条件筛选）
// @Summary 会员列表
// @Tags 会员管理
// @Param page query int false "页码，默认 1"
// @Param pageSize query int false "每页条数，默认 10，最大 100"
// @Param mobile query string false "手机号（模糊搜索）"
// @Param nickname query string false "昵称（模糊搜索）"
// @Param memberLevel query int false "会员等级"
// @Param memberStatus query int false "会员状态 0-正常 1-过期 2-冻结"
// @Param startTimeStart query string false "开通会员时间开始 (Unix 时间戳)"
// @Param startTimeEnd query string false "开通会员时间结束 (Unix 时间戳)"
// @Success 200 {object} response.Response{data=dto.PlatformMemberListResp}
// @Router /api/v1/platform-member/list [get]
// @Security Bearer
func (e PlatformMemberList) GetPlatformMemberList(c *gin.Context) {
	// 1. 上下文判空保护
	if c == nil {
		e.Error(500, errors.New("context is nil"), "请求上下文异常")
		return
	}

	// 2. 服务初始化
	s := service.PlatformMemberListService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Logger.Errorf("服务初始化失败：%v", err)
		e.Error(500, err, "服务初始化失败")
		return
	}

	// 3. 解析请求参数
	var req dto.PlatformMemberListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		e.Logger.Warnf("参数解析失败：%v", err)
		e.Error(400, err, "参数解析失败")
		return
	}

	// 4. 分页参数校验
	req.PageIndex = req.GetPageIndex()
	req.PageSize = req.GetPageSize()
	if req.PageSize > platformMemberMaxPageSize {
		req.PageSize = platformMemberMaxPageSize
	}

	// 5. 获取数据权限（简化版，暂不启用权限控制）
	p := &actions.DataPermission{}

	// 6. 调用服务层查询
	var list []dto.PlatformMemberListItem
	var total int64
	err := s.GetPage(&req, p, &list, &total)
	if err != nil {
		e.Logger.Errorf("查询会员列表失败：%v", err)
		e.Error(500, err, "查询失败")
		return
	}

	// 7. 封装返回数据
	resp := dto.PlatformMemberListResp{
		List:     list,
		Total:    total,
		Page:     req.PageIndex,
		PageSize: req.PageSize,
	}

	// 8. 记录操作日志
	e.Logger.Infof("查询会员列表成功，共 %d 条记录", total)

	// 9. 返回成功结果
	e.OK(resp, "查询成功")
}

// GetPlatformMemberDetail 会员详情
// @Summary 会员详情
// @Tags 会员管理
// @Param userId path int true "用户 ID"
// @Success 200 {object} response.Response{data=dto.PlatformMemberDetailResp}
// @Router /api/v1/platform-member/{userId} [get]
// @Security Bearer
func (e PlatformMemberList) GetPlatformMemberDetail(c *gin.Context) {
	// 1. 上下文判空保护
	if c == nil {
		e.Error(500, errors.New("context is nil"), "请求上下文异常")
		return
	}

	// 2. 服务初始化
	s := service.PlatformMemberListService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Logger.Errorf("服务初始化失败：%v", err)
		e.Error(500, err, "服务初始化失败")
		return
	}

	// 3. 解析路径参数
	var req dto.PlatformMemberDetailReq
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
	e.Logger.Infof("请求查询会员 %d 详情", req.UserId)

	// 6. 调用服务层查询
	info, err := s.GetDetail(req.UserId)
	if err != nil {
		e.Logger.Errorf("查询会员详情失败：%v", err)
		if err.Error() == "用户不存在" {
			e.Error(404, err, "用户不存在")
			return
		}
		e.Error(500, err, "查询失败")
		return
	}

	// 7. 记录操作日志
	e.Logger.Infof("成功查询会员 %d 详情，会员等级：%s，剩余天数：%d",
		req.UserId, info.MemberLevelName, info.RemainingDays)

	// 8. 返回成功结果
	e.OK(info, "查询成功")
}

// UpdatePlatformMember 更新会员信息
// @Summary 更新会员信息（开通/续费/升级/降级/延长）
// @Tags 会员管理
// @Param data body dto.PlatformMemberUpdateReq true "会员信息"
// @Success 200 {object} response.Response{data=dto.PlatformMemberUpdateResp}
// @Router /api/v1/platform-member [put]
// @Security Bearer
func (e PlatformMemberList) UpdatePlatformMember(c *gin.Context) {
	// 1. 上下文判空保护
	if c == nil {
		e.Error(500, errors.New("context is nil"), "请求上下文异常")
		return
	}

	// 2. 服务初始化
	s := service.PlatformMemberListService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Logger.Errorf("服务初始化失败：%v", err)
		e.Error(500, err, "服务初始化失败")
		return
	}

	// 3. 解析请求参数
	var req dto.PlatformMemberUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		e.Logger.Warnf("参数解析失败：%v", err)
		e.Error(400, err, "参数解析失败")
		return
	}

	// 4. 参数合法性校验
	if req.UserId <= 0 {
		e.Error(400, errors.New("用户 ID 格式错误"), "参数异常")
		return
	}

	// 校验备注必填
	if req.Remark == "" {
		e.Error(400, errors.New("操作备注不能为空"), "参数异常")
		return
	}

	// 从 JWT 获取管理员 ID（简化处理，暂时设为 0）
	// TODO: 实现 JWT 管理员 ID 解析
	req.OperatorId = 0

	// 5. 记录操作日志
	e.Logger.Infof("管理员 %d 请求更新会员 %d 信息，等级：%d，过期时间：%d，备注：%s",
		req.OperatorId, req.UserId, req.MemberLevel, req.ExpireTime, req.Remark)

	// 6. 调用服务层更新
	resp, err := s.Update(&req)
	if err != nil {
		e.Logger.Errorf("更新会员信息失败：%v", err)
		if err.Error() == "用户不存在" {
			e.Error(404, err, "用户不存在")
			return
		}
		if err.Error() == "用户账号状态异常" || err.Error() == "操作备注不能为空" || err.Error() == "到期时间必须晚于当前时间" || err.Error() == "禁止设置超过 10 年的有效期" {
			e.Error(400, err, "参数异常")
			return
		}
		e.Error(500, err, "更新失败")
		return
	}

	// 7. 记录操作日志
	e.Logger.Infof("成功更新会员 %d 信息，新等级：%s，新到期时间：%d，剩余天数：%d",
		req.UserId, resp.MemberLevelName, resp.ExpireTime, resp.RemainingDays)

	// 8. 返回成功结果
	e.OK(resp, "更新成功")
}

// FreezePlatformMember 冻结会员
// @Summary 冻结会员
// @Tags 会员管理
// @Param data body dto.PlatformMemberFreezeReq true "冻结信息"
// @Success 200 {object} response.Response{data=dto.PlatformMemberFreezeResp}
// @Router /api/v1/platform-member/freeze [post]
// @Security Bearer
func (e PlatformMemberList) FreezePlatformMember(c *gin.Context) {
	// 1. 上下文判空保护
	if c == nil {
		e.Error(500, errors.New("context is nil"), "请求上下文异常")
		return
	}

	// 2. 服务初始化
	s := service.PlatformMemberListService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Logger.Errorf("服务初始化失败：%v", err)
		e.Error(500, err, "服务初始化失败")
		return
	}

	// 3. 解析请求参数
	var req dto.PlatformMemberFreezeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		e.Logger.Warnf("参数解析失败：%v", err)
		e.Error(400, err, "参数解析失败")
		return
	}

	// 4. 参数合法性校验
	if req.UserId <= 0 {
		e.Error(400, errors.New("用户 ID 格式错误"), "参数异常")
		return
	}

	// 校验冻结原因必填
	if req.FreezeReason == "" {
		e.Error(400, errors.New("冻结原因不能为空"), "参数异常")
		return
	}

	// 从 JWT 获取管理员 ID（简化处理，暂时设为 0）
	// TODO: 实现 JWT 管理员 ID 解析
	req.OperatorId = 0

	// 5. 记录操作日志
	e.Logger.Infof("管理员 %d 请求冻结会员 %d，原因：%s",
		req.OperatorId, req.UserId, req.FreezeReason)

	// 6. 调用服务层冻结
	resp, err := s.FreezeMember(&req)
	if err != nil {
		e.Logger.Errorf("冻结会员失败：%v", err)
		if err.Error() == "用户不存在" {
			e.Error(404, err, "用户不存在")
			return
		}
		if err.Error() == "用户账号状态异常" || err.Error() == "冻结原因不能为空" ||
			err.Error() == "用户未开通会员" || err.Error() == "会员已过期，无需冻结" ||
			err.Error() == "会员已冻结" {
			e.Error(400, err, "参数异常")
			return
		}
		e.Error(500, err, "冻结失败")
		return
	}

	// 7. 记录操作日志
	e.Logger.Infof("成功冻结会员 %d，冻结时间：%d", req.UserId, resp.FreezeTime)

	// 8. 返回成功结果
	e.OK(resp, "冻结成功")
}

// UnfreezePlatformMember 解冻会员
// @Summary 解冻会员
// @Tags 会员管理
// @Param data body dto.PlatformMemberUnfreezeReq true "解冻信息"
// @Success 200 {object} response.Response{data=dto.PlatformMemberUnfreezeResp}
// @Router /api/v1/platform-member/unfreeze [post]
// @Security Bearer
func (e PlatformMemberList) UnfreezePlatformMember(c *gin.Context) {
	// 1. 上下文判空保护
	if c == nil {
		e.Error(500, errors.New("context is nil"), "请求上下文异常")
		return
	}

	// 2. 服务初始化
	s := service.PlatformMemberListService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Logger.Errorf("服务初始化失败：%v", err)
		e.Error(500, err, "服务初始化失败")
		return
	}

	// 3. 解析请求参数
	var req dto.PlatformMemberUnfreezeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		e.Logger.Warnf("参数解析失败：%v", err)
		e.Error(400, err, "参数解析失败")
		return
	}

	// 4. 参数合法性校验
	if req.UserId <= 0 {
		e.Error(400, errors.New("用户 ID 格式错误"), "参数异常")
		return
	}

	// 校验解冻原因必填
	if req.UnfreezeReason == "" {
		e.Error(400, errors.New("解冻原因不能为空"), "参数异常")
		return
	}

	// 从 JWT 获取管理员 ID（简化处理，暂时设为 0）
	// TODO: 实现 JWT 管理员 ID 解析
	req.OperatorId = 0

	// 5. 记录操作日志
	e.Logger.Infof("管理员 %d 请求解冻会员 %d，原因：%s",
		req.OperatorId, req.UserId, req.UnfreezeReason)

	// 6. 调用服务层解冻
	resp, err := s.UnfreezeMember(&req)
	if err != nil {
		e.Logger.Errorf("解冻会员失败：%v", err)
		if err.Error() == "用户不存在" {
			e.Error(404, err, "用户不存在")
			return
		}
		if err.Error() == "用户账号状态异常" || err.Error() == "解冻原因不能为空" ||
			err.Error() == "用户未开通会员" || err.Error() == "会员未冻结，无需解冻" {
			e.Error(400, err, "参数异常")
			return
		}
		e.Error(500, err, "解冻失败")
		return
	}

	// 7. 记录操作日志
	e.Logger.Infof("成功解冻会员 %d，解冻时间：%d", req.UserId, resp.UnfreezeTime)

	// 8. 返回成功结果
	e.OK(resp, "解冻成功")
}

// SavePlatformMemberRightConfig 保存/更新会员权益配置
// @Summary 保存会员权益配置
// @Tags 会员管理
// @Param data body dto.PlatformMemberRightConfigReq true "权益配置信息"
// @Success 200 {object} response.Response{data=dto.PlatformMemberRightConfigResp}
// @Router /api/v1/platform-member/right-config [post]
// @Security Bearer
func (e PlatformMemberList) SavePlatformMemberRightConfig(c *gin.Context) {
	// 1. 上下文判空保护
	if c == nil {
		e.Error(500, errors.New("context is nil"), "请求上下文异常")
		return
	}

	// 2. 服务初始化
	s := service.PlatformMemberListService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Logger.Errorf("服务初始化失败：%v", err)
		e.Error(500, err, "服务初始化失败")
		return
	}

	// 3. 解析请求参数
	var req dto.PlatformMemberRightConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		e.Logger.Warnf("参数解析失败：%v", err)
		e.Error(400, err, "参数解析失败")
		return
	}

	// 4. 参数合法性校验
	if req.LevelId < 0 || req.LevelId > 3 {
		e.Error(400, errors.New("会员等级不合法"), "参数异常")
		return
	}

	// 从 JWT 获取管理员 ID（简化处理，暂时设为 0）
	// TODO: 实现 JWT 管理员 ID 解析
	req.OperatorId = 0

	// 5. 记录操作日志
	e.Logger.Infof("管理员 %d 请求配置会员等级 %d 权益，权益数量：%d",
		req.OperatorId, req.LevelId, len(req.Rights))

	// 6. 调用服务层保存配置
	resp, err := s.SaveRightConfig(&req)
	if err != nil {
		e.Logger.Errorf("保存权益配置失败：%v", err)
		if err.Error() == "会员等级不合法" {
			e.Error(400, err, "参数异常")
			return
		}
		e.Error(500, err, "保存失败")
		return
	}

	// 7. 记录操作日志
	e.Logger.Infof("成功保存会员等级 %d 权益配置，权益数量：%d", req.LevelId, len(req.Rights))

	// 8. 返回成功结果
	e.OK(resp, "配置成功")
}
