package service

import (
	"errors"
	"time"

	"go-admin/app/admin/user/models"

	"github.com/go-admin-team/go-admin-core/sdk/service"

	"gorm.io/gorm"
)

// UserRealNameAuditService 实名认证审核服务
type UserRealNameAuditService struct {
	service.Service
}

// AuditRealName 审核实名认证
func (s *UserRealNameAuditService) AuditRealName(req *models.UserRealNameAuditReq, operatorID int64) (*models.UserRealNameAuditResp, error) {
	// 开启事务
	var resp *models.UserRealNameAuditResp
	err := s.Orm.Transaction(func(tx *gorm.DB) error {
		// 查询实名记录
		var auth models.UserRealNameAuth
		err := tx.Where("user_id = ?", req.UserID).
			Order("created_at DESC").
			First(&auth).Error

		if err == gorm.ErrRecordNotFound {
			return errors.New("该用户暂无实名认证记录")
		}
		if err != nil {
			return errors.New("查询实名记录失败")
		}

		// 校验状态必须是待人工审核（20）
		if auth.AuthStatus != 20 {
			return errors.New("当前实名记录状态不可审核")
		}

		// 根据审核结果执行操作
		var newStatus int16
		var userRealNameStatus int16
		var message string

		if req.AuditResult == 1 {
			// 审核通过
			newStatus = 21         // 人工通过
			userRealNameStatus = 2 // 已通过
			message = "实名认证审核已通过"
		} else if req.AuditResult == 2 {
			// 审核驳回
			newStatus = 22         // 人工驳回
			userRealNameStatus = 3 // 已驳回
			message = "实名认证审核已驳回"
		} else {
			return errors.New("审核结果参数错误")
		}

		// 更新实名审核记录
		now := time.Now()
		updates := map[string]interface{}{
			"auth_status":   newStatus,
			"reviewer_note": req.AuditRemark,
			"reviewed_at":   now,
			"reviewed_by":   operatorID,
			"updated_at":    now,
		}

		err = tx.Model(&auth).Where("id = ?", auth.ID).Updates(updates).Error
		if err != nil {
			return errors.New("更新审核记录失败")
		}

		// 更新用户实名状态
		err = tx.Model(&models.SysUser{}).
			Where("id = ?", req.UserID).
			Updates(map[string]interface{}{
				"real_name_status": userRealNameStatus,
				"real_name_at":     now,
				"updated_at":       now,
			}).Error
		if err != nil {
			return errors.New("更新用户状态失败")
		}

		// 记录操作日志
		log := models.UserRealNameAuditLog{
			UserID:     req.UserID,
			AuthID:     auth.ID,
			OperatorID: operatorID,
			Action:     "audit",
			OldStatus:  auth.AuthStatus,
			NewStatus:  newStatus,
			Remark:     req.AuditRemark,
			CreatedAt:  now,
		}
		err = tx.Create(&log).Error
		if err != nil {
			// 日志记录失败不影响主流程
		}

		resp = &models.UserRealNameAuditResp{
			UserID:         req.UserID,
			RealNameStatus: userRealNameStatus,
			Message:        message,
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetRealNameList 获取实名认证列表
func (s *UserRealNameAuditService) GetRealNameList(page, pageSize int, authStatus *int) (*models.UserRealNameListResp, error) {
	var list []models.UserRealNameListItem
	var count int64

	offset := (page - 1) * pageSize

	// 构建查询
	query := s.Orm.Debug().
		Table("user_real_name_auth ura").
		Joins("JOIN sys_user u ON ura.user_id = u.id").
		Where("u.deleted_at IS NULL")

	// 按状态筛选
	if authStatus != nil && *authStatus > 0 {
		query = query.Where("ura.auth_status = ?", *authStatus)
	}

	// 查询总数
	err := query.Count(&count).Error
	if err != nil {
		return nil, errors.New("查询总数失败")
	}

	// 查询列表
	err = query.Select(`
		ura.id,
		ura.user_id,
		u.nickname,
		u.email,
		u.mobile,
		ura.real_name_masked,
		ura.id_number_last4,
		ura.cert_type,
		ura.auth_status,
		ura.created_at
	`).
		Order("ura.created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Scan(&list).Error

	if err != nil {
		return nil, errors.New("查询列表失败")
	}

	// 补充状态文本
	for i := range list {
		list[i].AuthStatusText = getAuthStatusText(list[i].AuthStatus)
	}

	return &models.UserRealNameListResp{
		List:  list,
		Total: count,
	}, nil
}

// GetRealNameDetail 获取实名认证详情
func (s *UserRealNameAuditService) GetRealNameDetail(userID int64) (*models.UserRealNameAuth, error) {
	var auth models.UserRealNameAuth

	err := s.Orm.
		Where("user_id = ?", userID).
		Order("created_at DESC").
		First(&auth).Error

	if err == gorm.ErrRecordNotFound {
		return nil, errors.New("该用户暂无实名认证记录")
	}
	if err != nil {
		return nil, errors.New("查询详情失败")
	}

	return &auth, nil
}

// getAuthStatusText 获取状态文本
func getAuthStatusText(status int16) string {
	switch status {
	case 10:
		return "待三方核验"
	case 11:
		return "三方核验通过"
	case 12:
		return "三方核验失败"
	case 20:
		return "待人工审核"
	case 21:
		return "人工审核通过"
	case 22:
		return "人工审核驳回"
	default:
		return "未知状态"
	}
}
