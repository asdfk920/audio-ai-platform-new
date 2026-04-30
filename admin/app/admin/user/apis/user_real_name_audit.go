package apis

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-admin-team/go-admin-core/sdk/api"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth/user"
	"gorm.io/gorm"

	"go-admin/app/admin/user/models"
	"go-admin/app/admin/user/service"
)

type UserRealNameAudit struct {
	api.Api
}

// AuditRealName 审核实名认证
// @Summary 审核实名认证
// @Description 管理员审核用户实名认证申请
// @Tags 实名认证
// @Accept application/json
// @Product application/json
// @Param data body models.UserRealNameAuditReq true "审核请求参数"
// @Success 200 {object} response.Response
// @Router /api/v1/user-realname/audit [post]
// @Security Bearer
func (a UserRealNameAudit) AuditRealName(c *gin.Context) {
	var req models.UserRealNameAuditReq
	if err := c.ShouldBindJSON(&req); err != nil {
		a.Error(400, err, "参数解析失败")
		return
	}

	// 获取当前登录管理员 ID
	operatorID := int64(user.GetUserId(c))

	// 参数校验
	if req.UserID <= 0 {
		a.Error(400, errors.New("用户 ID 不能为空"), "参数错误")
		return
	}

	if req.AuditResult != 1 && req.AuditResult != 2 {
		a.Error(400, errors.New("审核结果参数错误（1 通过 2 驳回）"), "参数错误")
		return
	}

	if req.AuditResult == 2 && req.AuditRemark == "" {
		a.Error(400, errors.New("驳回时必须填写审核意见"), "参数错误")
		return
	}

	// 执行审核
	s := service.UserRealNameAuditService{}
	if err := a.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		a.Logger.Error(err)
		a.Error(500, err, "服务初始化失败")
		return
	}

	resp, err := s.AuditRealName(&req, operatorID)
	if err != nil {
		a.Logger.Error(err)
		a.Error(400, err, "审核失败")
		return
	}

	a.OK(resp, "审核成功")
}

// GetRealNameList 获取实名认证列表
// @Summary 获取实名认证列表
// @Description 管理员查询实名认证列表
// @Tags 实名认证
// @Accept application/json
// @Product application/json
// @Param page query int true "页码"
// @Param page_size query int true "每页数量"
// @Param auth_status query int false "审核状态"
// @Success 200 {object} response.Response
// @Router /api/v1/user-realname/list [get]
// @Security Bearer
func (a UserRealNameAudit) GetRealNameList(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "10")
	authStatus := c.Query("auth_status")

	var p, ps int
	if _, err := fmt.Sscanf(page, "%d", &p); err != nil || p <= 0 {
		p = 1
	}
	if _, err := fmt.Sscanf(pageSize, "%d", &ps); err != nil || ps <= 0 {
		ps = 10
	}

	var statusPtr *int
	if authStatus != "" {
		var status int
		if _, err := fmt.Sscanf(authStatus, "%d", &status); err == nil {
			statusPtr = &status
		}
	}

	s := service.UserRealNameAuditService{}
	if err := a.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		a.Logger.Error(err)
		a.Error(500, err, "服务初始化失败")
		return
	}

	list, err := s.GetRealNameList(p, ps, statusPtr)
	if err != nil {
		a.Logger.Error(err)
		a.Error(500, err, "查询失败")
		return
	}

	a.PageOK(list, int(list.Total), p, ps, "查询成功")
}

// GetRealNameDetail 获取实名认证详情
// @Summary 获取实名认证详情
// @Description 管理员查看用户实名认证详细信息
// @Tags 实名认证
// @Accept application/json
// @Product application/json
// @Param user_id path int true "用户 ID"
// @Success 200 {object} response.Response
// @Router /api/v1/user-realname/detail/:user_id [get]
// @Security Bearer
func (a UserRealNameAudit) GetRealNameDetail(c *gin.Context) {
	userIDStr := c.Param("user_id")
	var userID int64
	if _, err := fmt.Sscanf(userIDStr, "%d", &userID); err != nil || userID <= 0 {
		a.Error(400, errors.New("用户 ID 格式错误"), "参数错误")
		return
	}

	s := service.UserRealNameAuditService{}
	if err := a.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		a.Logger.Error(err)
		a.Error(500, err, "服务初始化失败")
		return
	}

	detail, err := s.GetRealNameDetail(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			a.Error(404, errors.New("该用户暂无实名认证记录"), "记录不存在")
			return
		}
		a.Logger.Error(err)
		a.Error(500, err, "查询失败")
		return
	}

	a.OK(detail, "查询成功")
}
