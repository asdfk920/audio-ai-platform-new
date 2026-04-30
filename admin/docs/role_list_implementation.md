# 角色列表接口实现文档

## 接口概述

角色列表接口用于查看系统中所有的角色配置信息，包括角色名称、权限范围、关联管理员数量、创建时间等详细统计信息。

## API 接口

### 请求方式

```
GET /api/v1/role/list
```

### 请求参数

| 参数名 | 类型 | 必填 | 说明 | 默认值 |
|--------|------|------|------|--------|
| keyword | string | 否 | 关键词（角色名称或编码） | - |
| status | string | 否 | 状态（1:禁用 2:正常） | - |
| page | int | 否 | 页码 | 1 |
| page_size | int | 否 | 每页数量 | 20 |
| sort_by | string | 否 | 排序字段 | created_at |
| sort_order | string | 否 | 排序方式（asc/desc） | desc |

### 返回数据结构

```json
{
  "code": 200,
  "msg": "查询成功",
  "data": {
    "list": [
      {
        "role_id": 1,
        "role_name": "超级管理员",
        "role_code": "admin",
        "description": "系统超级管理员",
        "status": "2",
        "status_text": "正常",
        "permission_count": 50,
        "permission_list": ["user.view", "user.edit", "role.view"],
        "admin_count": 3,
        "created_at": "2024-01-01 10:00:00",
        "created_by": "system",
        "updated_at": "2024-01-01 10:00:00",
        "updated_by": "admin"
      }
    ],
    "count": 10,
    "pageIndex": 1,
    "pageSize": 20
  }
}
```

### 返回字段说明

| 字段名 | 类型 | 说明 |
|--------|------|------|
| role_id | int | 角色 ID |
| role_name | string | 角色名称 |
| role_code | string | 角色编码 |
| description | string | 角色描述 |
| status | string | 角色状态（1:禁用 2:正常） |
| status_text | string | 状态文本描述 |
| permission_count | int | 权限数量 |
| permission_list | []string | 权限标识列表 |
| admin_count | int | 关联管理员数量 |
| created_at | string | 创建时间 |
| created_by | string | 创建人 |
| updated_at | string | 更新时间 |
| updated_by | string | 更新人 |

## 处理流程

### 1. 参数解析

解析关键词、分页、排序等参数：
- `keyword`: 用于模糊匹配角色名称或编码
- `status`: 精确匹配角色状态
- `page` / `page_size`: 分页参数
- `sort_by` / `sort_order`: 排序参数

### 2. 权限校验

校验操作人是否有权限查看角色列表：
- 普通管理员只能查看，不能管理角色
- 超级管理员拥有所有权限

### 3. 构建查询条件

- 关键词模糊匹配：`role_name LIKE '%keyword%' OR role_key LIKE '%keyword%'`
- 状态精确匹配：`status = '1' or '2'`
- 其他条件精确匹配

### 4. 执行分页查询

- 统计符合条件的总条数
- 查询当前页数据
- 默认按创建时间倒序排列

### 5. 关联统计

对每个角色进行关联统计：
- **权限数量统计**：查询 `sys_role_menu` 表中该角色的菜单数量
- **权限列表获取**：查询角色关联菜单的权限标识
- **管理员数量统计**：查询 `sys_user_role` 表中该角色关联的用户数量

### 6. 格式化返回

- 转换枚举值为中文描述（如状态 1→禁用，2→正常）
- 格式化时间字段为 `YYYY-MM-DD HH:mm:ss`
- 获取创建人和更新人的用户名

## 数据模型

### SysRoleListItem

```go
type SysRoleListItem struct {
    RoleId         int      `json:"role_id"`          // 角色 ID
    RoleName       string   `json:"role_name"`        // 角色名称
    RoleCode       string   `json:"role_code"`        // 角色编码
    Description    string   `json:"description"`      // 角色描述
    Status         string   `json:"status"`           // 状态
    StatusText     string   `json:"status_text"`      // 状态文本
    PermissionCount int     `json:"permission_count"` // 权限数量
    PermissionList []string `json:"permission_list"`  // 权限标识列表
    AdminCount     int      `json:"admin_count"`      // 关联管理员数量
    CreatedAt      string   `json:"created_at"`       // 创建时间
    CreatedBy      string   `json:"created_by"`       // 创建人
    UpdatedAt      string   `json:"updated_at"`       // 更新时间
    UpdatedBy      string   `json:"updated_by"`       // 更新人
}
```

## 角色类型说明

| 角色类型 | 权限说明 | 可删除 | 可修改 |
|----------|----------|--------|--------|
| 超级管理员 | 系统所有权限 | 否 | 否 |
| 系统管理员 | 系统配置和用户管理 | 是 | 是 |
| 运营管理员 | 日常运营操作 | 是 | 是 |
| 审计管理员 | 仅可查看日志和报表 | 是 | 是 |
| 普通管理员 | 根据业务需求配置 | 是 | 是 |

## 权限范围

角色需绑定具体的权限标识列表：
- `user.view`: 用户查看权限
- `user.edit`: 用户编辑权限
- `device.admin`: 设备管理权限
- `role.view`: 角色查看权限
- `role.edit`: 角色编辑权限

权限分为：
1. **菜单权限**：控制页面可见性
2. **操作权限**：控制按钮可用性
3. **数据权限**：控制数据范围可见性

## 角色状态

| 状态值 | 状态文本 | 说明 |
|--------|----------|------|
| 1 | 禁用 | 不可被分配给新管理员，但不影响已关联的管理员使用 |
| 2 | 正常 | 可被正常分配和使用 |

## 排序规则

- 默认按创建时间倒序（最新创建在前）
- 支持按以下字段排序：
  - `role_name`: 角色名称
  - `admin_count`: 管理员数量
  - `created_at`: 创建时间
  - `updated_at`: 更新时间

## 实现文件

### API 层
- `app/admin/apis/sys_role_list.go`: 角色列表 API 接口

### 服务层
- `app/admin/service/sys_role.go`: 角色业务逻辑（GetRoleList 方法）

### DTO 层
- `app/admin/service/dto/sys_role.go`: 数据传输对象（SysRoleListItem, SysRoleListResponse）

### 模型层
- `app/admin/models/sys_role.go`: 角色数据模型

## 数据库表

### sys_role
角色主表，存储角色基本信息

### sys_role_menu
角色菜单关联表，存储角色与菜单的关联关系

### sys_user_role
用户角色关联表，存储用户与角色的关联关系

### sys_menu
菜单表，存储菜单信息及权限标识

## 异常处理

| 异常场景 | 错误提示 |
|----------|----------|
| 权限不足 | 您没有权限查看角色列表 |
| 数据库错误 | 查询失败，请联系管理员 |
| 参数错误 | 参数格式不正确 |

## 使用示例

### 查询所有角色

```bash
curl -X GET "http://localhost:8000/api/v1/role/list" \
  -H "Authorization: Bearer <token>"
```

### 按状态筛选

```bash
curl -X GET "http://localhost:8000/api/v1/role/list?status=2" \
  -H "Authorization: Bearer <token>"
```

### 关键词搜索

```bash
curl -X GET "http://localhost:8000/api/v1/role/list?keyword=管理员" \
  -H "Authorization: Bearer <token>"
```

### 分页查询

```bash
curl -X GET "http://localhost:8000/api/v1/role/list?page=1&page_size=10" \
  -H "Authorization: Bearer <token>"
```

### 排序查询

```bash
curl -X GET "http://localhost:8000/api/v1/role/list?sort_by=admin_count&sort_order=desc" \
  -H "Authorization: Bearer <token>"
```

## 注意事项

1. 禁用状态的角色不可被分配给新管理员，但不影响已关联的管理员使用
2. 超级管理员角色不可删除和修改
3. 权限统计实时查询，可能存在性能开销，建议合理设置分页大小
4. 关键词搜索同时匹配角色名称和角色编码
