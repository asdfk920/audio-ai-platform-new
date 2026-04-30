package service

import (
	"github.com/go-admin-team/go-admin-core/sdk/service"

	"go-admin/app/admin/models"
	"go-admin/app/admin/service/dto"
	cDto "go-admin/common/dto"
)

type SysPermission struct {
	service.Service
}

// GetPermissionList 获取权限列表（按模块分组）
func (e *SysPermission) GetPermissionList(c *dto.SysPermissionGetPageReq, list *[]dto.SysPermissionListItem, count *int64) error {
	var data models.SysMenu

	// 构建查询条件
	db := e.Orm.Model(&data).
		Scopes(cDto.MakeCondition(c.GetNeedSearch()))

	// 关键词筛选（权限名称或标识）
	if c.Keyword != "" {
		db = db.Where("menu_name LIKE ? OR permission LIKE ?", "%"+c.Keyword+"%", "%"+c.Keyword+"%")
	}

	// 模块筛选
	if c.Module != "" {
		db = db.Where("module = ?", c.Module)
	}

	// 类型筛选
	if c.Type != "" {
		db = db.Where("type = ?", c.Type)
	}

	// 状态筛选
	if c.Status != "" {
		db = db.Where("status = ?", c.Status)
	}

	// 统计总数
	if err := db.Count(count).Error; err != nil {
		e.Log.Errorf("db error:%s", err)
		return err
	}

	// 排序
	sortBy := "sort"
	if c.SortBy != "" {
		sortBy = c.SortBy
	}
	sortOrder := "asc"
	if c.SortOrder != "" {
		sortOrder = c.SortOrder
	}
	db = db.Order(sortBy + " " + sortOrder)

	// 分页
	page := c.GetPageIndex()
	pageSize := c.GetPageSize()
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	db = db.Offset(offset).Limit(pageSize)

	// 查询数据
	var menus []models.SysMenu
	if err := db.Find(&menus).Error; err != nil {
		e.Log.Errorf("db error:%s", err)
		return err
	}

	// 格式化返回
	*list = make([]dto.SysPermissionListItem, 0, len(menus))
	for _, menu := range menus {
		item := dto.SysPermissionListItem{
			PermissionId:   menu.MenuId,
			MenuId:         menu.MenuId,
			PermissionName: menu.MenuName,
			PermissionCode: menu.Permission,
			Description:    menu.Remark,
			Type:           menu.Type,
			TypeText:       permissionTypeText(menu.Type),
			Module:         menu.Module,
			ModuleText:     moduleText(menu.Module),
			Status:         menu.Status,
			StatusText:     statusText(menu.Status),
			CreatedAt:      menu.CreatedAt.Format("2006-01-02 15:04:05"),
		}

		// 统计被多少个角色引用
		var roleCount int64
		if err := e.Orm.Table("sys_role_menu").
			Where("menu_id = ?", menu.MenuId).
			Count(&roleCount).Error; err == nil {
			item.RoleCount = int(roleCount)
		}

		*list = append(*list, item)
	}

	return nil
}

// GetPermissionTree 获取权限树形结构
func (e *SysPermission) GetPermissionTree(req *dto.SysPermissionTreeReq) ([]dto.SysPermissionTreeItem, error) {
	var menus []models.SysMenu
	db := e.Orm.Model(&menus)

	// 模块筛选
	if req.Module != "" {
		db = db.Where("module = ?", req.Module)
	}

	// 状态筛选
	if req.Status != "" {
		db = db.Where("status = ?", req.Status)
	}

	// 查询所有菜单（权限）
	if err := db.Order("sort ASC").Find(&menus).Error; err != nil {
		return nil, err
	}

	// 按模块分组
	moduleMap := make(map[string][]models.SysMenu)
	for _, menu := range menus {
		moduleMap[menu.Module] = append(moduleMap[menu.Module], menu)
	}

	// 构建权限树
	tree := make([]dto.SysPermissionTreeItem, 0, len(moduleMap))
	for moduleCode, moduleMenus := range moduleMap {
		// 创建模块节点
		moduleNode := dto.SysPermissionTreeItem{
			Id:       -1, // 模块节点使用负数 ID
			Label:    moduleText(moduleCode),
			Type:     "module",
			Children: make([]*dto.SysPermissionTreeItem, 0),
		}

		// 添加权限子节点
		for _, menu := range moduleMenus {
			childNode := dto.SysPermissionTreeItem{
				Id:             menu.MenuId,
				Label:          menu.MenuName,
				Type:           "permission",
				PermissionCode: menu.Permission,
				Module:         menu.Module,
				Status:         menu.Status,
			}
			moduleNode.Children = append(moduleNode.Children, &childNode)
		}

		tree = append(tree, moduleNode)
	}

	return tree, nil
}

// permissionTypeText 权限类型文本转换
func permissionTypeText(permissionType string) string {
	switch permissionType {
	case "1":
		return "菜单权限"
	case "2":
		return "操作权限"
	case "3":
		return "数据权限"
	default:
		return "未知类型"
	}
}

// moduleText 模块文本转换
func moduleText(module string) string {
	switch module {
	case "user":
		return "用户模块"
	case "device":
		return "设备模块"
	case "content":
		return "内容模块"
	case "order":
		return "订单模块"
	case "system":
		return "系统模块"
	default:
		return module
	}
}

// statusText 状态文本转换
func statusText(status string) string {
	switch status {
	case "1":
		return "禁用"
	case "2":
		return "启用"
	default:
		return "未知状态"
	}
}
