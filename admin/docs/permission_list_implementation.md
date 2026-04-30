# 权限列表接口实现文档

## 接口概述

权限列表接口用于查看系统中所有的权限标识，按模块和类型分组展示，供配置角色时选择关联权限。

## API 接口

### 1. 权限列表接口

#### 请求方式

```
GET /api/v1/permission/list
```

#### 请求参数

| 参数名 | 类型 | 必填 | 说明 | 默认值 | 校验规则 |
|--------|------|------|------|--------|----------|
| module | string | 否 | 权限所属模块（user/device/content/order/system） | - | - |
| type | string | 否 | 权限类型（1:菜单权限 2:操作权限 3:数据权限） | - | - |
| keyword | string | 否 | 关键词搜索权限名称或标识 | - | - |
| status | string | 否 | 权限状态（1:禁用 2:启用） | - | - |
| page | int | 否 | 页码 | 1 | - |
| page_size | int | 否 | 每页数量 | 20 | - |
| sort_by | string | 否 | 排序字段 | sort | - |
| sort_order | string | 否 | 排序方式 | asc | - |

#### 请求示例

```
GET /api/v1/permission/list?module=user&type=1&keyword=view&page=1&page_size=20
```

#### 返回数据结构

```json
{
  "code": 200,
  "msg": "查询成功",
  "data": {
    "list": [
      {
        "permission_id": 1,
        "menu_id": 1,
        "permission_name": "用户查看",
        "permission_code": "user.view",
        "description": "查看用户信息权限",
        "type": "1",
        "type_text": "菜单权限",
        "module": "user",
        "module_text": "用户模块",
        "status": "2",
        "status_text": "启用",
        "role_count": 5,
        "created_at": "2024-01-01 10:00:00"
      }
    ],
    "count": 50,
    "pageIndex": 1,
    "pageSize": 20
  }
}
```

#### 返回字段说明

| 字段名 | 类型 | 说明 |
|--------|------|------|
| permission_id | int | 权限 ID |
| menu_id | int | 菜单 ID |
| permission_name | string | 权限名称 |
| permission_code | string | 权限标识（如 user.view） |
| description | string | 权限描述 |
| type | string | 权限类型（1:菜单 2:操作 3:数据） |
| type_text | string | 权限类型文本 |
| module | string | 所属模块 |
| module_text | string | 所属模块文本 |
| status | string | 权限状态（1:禁用 2:启用） |
| status_text | string | 权限状态文本 |
| role_count | int | 被多少个角色引用 |
| created_at | string | 创建时间 |

### 2. 权限树形结构接口

#### 请求方式

```
GET /api/v1/permission/tree
```

#### 请求参数

| 参数名 | 类型 | 必填 | 说明 | 默认值 |
|--------|------|------|------|--------|
| module | string | 否 | 权限所属模块 | - |
| status | string | 否 | 权限状态 | - |

#### 请求示例

```
GET /api/v1/permission/tree?module=user
```

#### 返回数据结构

```json
{
  "code": 200,
  "msg": "查询成功",
  "data": [
    {
      "id": -1,
      "label": "用户模块",
      "type": "module",
      "children": [
        {
          "id": 1,
          "label": "用户查看",
          "type": "permission",
          "permission_code": "user.view",
          "module": "user",
          "status": "2"
        },
        {
          "id": 2,
          "label": "用户新增",
          "type": "permission",
          "permission_code": "user.add",
          "module": "user",
          "status": "2"
        }
      ]
    }
  ]
}
```

#### 返回字段说明

| 字段名 | 类型 | 说明 |
|--------|------|------|
| id | int | 节点 ID（模块节点为负数，权限节点为正数） |
| label | string | 节点标签 |
| type | string | 节点类型（module: 模块 permission: 权限） |
| children | array | 子节点列表（仅模块节点有） |
| permission_code | string | 权限标识（仅权限节点有） |
| module | string | 所属模块（仅权限节点有） |
| status | string | 状态（仅权限节点有） |

## 处理流程

### 权限列表接口流程

#### 第一步 参数解析

解析模块、类型、关键词等筛选参数：
- 解析 `module` 参数筛选所属模块
- 解析 `type` 参数筛选权限类型
- 解析 `keyword` 参数进行关键词搜索
- 解析 `status` 参数筛选权限状态
- 解析分页和排序参数

#### 第二步 权限校验

校验操作人是否有权限查看权限列表：
- 检查操作人是否登录
- 检查操作人是否有查看权限的权限
- （可选）根据角色过滤可看到的权限

#### 第三步 模块加载

加载所有权限模块的配置信息：
- 从数据库查询所有模块
- 或从配置文件中加载模块信息
- 模块包括：用户模块、设备模块、内容模块、订单模块、系统模块

#### 第四步 权限查询

查询符合条件的权限记录：
- 构建 GORM 查询条件
- 应用模块、类型、关键词、状态筛选
- 执行分页查询
- 按模块分组组装数据

#### 第五步 关联统计

统计每个权限被多少个角色引用：
- 查询 `sys_role_menu` 表
- 统计每个 `menu_id` 被引用的次数
- 将统计结果添加到返回数据中

#### 第六步 格式化返回

转换枚举值为中文描述：
- 权限类型转换（1→菜单权限，2→操作权限，3→数据权限）
- 模块转换（user→用户模块，device→设备模块等）
- 状态转换（1→禁用，2→启用）
- 组织模块分组结构

### 权限树接口流程

#### 第一步 参数解析

解析模块和状态筛选参数：
- 解析 `module` 参数
- 解析 `status` 参数

#### 第二步 查询权限

查询所有权限记录：
- 按 `sort` 字段排序
- 应用模块和状态筛选
- 查询所有符合条件的权限

#### 第三步 按模块分组

将权限按模块分组：
- 遍历所有权限
- 按 `module` 字段分组
- 形成模块→权限的映射关系

#### 第四步 构建权限树

构建树形结构：
- 创建模块节点（父节点）
- 创建权限节点（子节点）
- 将权限节点添加到对应模块节点下
- 返回树形结构数组

## 模块分组

### 用户模块（user）

包含用户相关权限：
- 用户查看（user.view）
- 用户新增（user.add）
- 用户编辑（user.edit）
- 用户删除（user.delete）
- 用户管理（user.manage）

### 设备模块（device）

包含设备相关权限：
- 设备管理（device.manage）
- 设备查看（device.view）
- 设备新增（device.add）
- 设备编辑（device.edit）
- 设备删除（device.delete）
- 设备指令（device.command）

### 内容模块（content）

包含内容相关权限：
- 内容管理（content.manage）
- 内容查看（content.view）
- 内容新增（content.add）
- 内容编辑（content.edit）
- 内容删除（content.delete）
- 内容审核（content.audit）
- 内容统计（content.statistics）

### 订单模块（order）

包含订单相关权限：
- 订单查看（order.view）
- 订单新增（order.add）
- 订单编辑（order.edit）
- 订单删除（order.delete）
- 订单处理（order.process）
- 退款管理（order.refund）

### 系统模块（system）

包含系统相关权限：
- 系统配置（system.config）
- 角色管理（system.role）
- 权限管理（system.permission）
- 日志查看（system.log）
- 菜单管理（system.menu）

## 权限类型

### 菜单权限（type=1）

- **功能**：控制后台菜单的可见性
- **作用**：用户只能看到有权限的菜单
- **示例**：用户管理菜单、设备管理菜单
- **权限标识**：通常为模块名称，如 `user`、`device`

### 操作权限（type=2）

- **功能**：控制具体操作按钮的可用性
- **作用**：用户只能执行有权限的操作
- **示例**：新增、编辑、删除、查看等按钮
- **权限标识**：模块。操作，如 `user.add`、`user.edit`

### 数据权限（type=3）

- **功能**：控制数据范围
- **作用**：用户只能看到授权范围内的数据
- **示例**：只看本部门数据、可看全部数据
- **权限标识**：模块。数据范围，如 `user.dept`、`user.all`

## 权限标识规范

### 命名规则

- **格式**：`模块。操作`
- **示例**：
  - `user.view` - 用户查看
  - `user.add` - 用户新增
  - `user.edit` - 用户编辑
  - `user.delete` - 用户删除
  - `permission.edit` - 权限编辑

### 命名约定

1. **模块名称**：使用小写英文单词
   - user（用户）
   - device（设备）
   - content（内容）
   - order（订单）
   - system（系统）

2. **操作名称**：使用小写英文单词
   - view（查看）
   - add（新增）
   - edit（编辑）
   - delete（删除）
   - manage（管理）
   - audit（审核）
   - statistics（统计）
   - process（处理）
   - refund（退款）

3. **点分格式**：便于程序解析和权限校验
   - 使用英文句点 `.` 分隔
   - 模块在前，操作在后
   - 简洁明了，易于理解

## 树形结构展示

### 节点类型

1. **模块节点**
   - `type`: "module"
   - `id`: 负数（区别于权限节点）
   - `label`: 模块中文名称
   - `children`: 权限子节点数组

2. **权限节点**
   - `type`: "permission"
   - `id`: 菜单 ID（正数）
   - `label`: 权限名称
   - `permission_code`: 权限标识
   - `module`: 所属模块
   - `status`: 权限状态

### 树形结构优势

- **层级清晰**：模块为父节点，权限为子节点
- **直观展示**：一目了然查看权限归属关系
- **便于操作**：支持展开/收起操作
- **易于选择**：配置角色时可按模块批量选择

## 数据模型

### SysPermissionGetPageReq

```go
type SysPermissionGetPageReq struct {
    dto.Pagination
    Module    string `form:"module"`
    Type      string `form:"type"`
    Keyword   string `form:"keyword"`
    Status    string `form:"status"`
    SortBy    string `form:"sort_by"`
    SortOrder string `form:"sort_order"`
}
```

### SysPermissionListItem

```go
type SysPermissionListItem struct {
    PermissionId   int    `json:"permission_id"`
    MenuId         int    `json:"menu_id"`
    PermissionName string `json:"permission_name"`
    PermissionCode string `json:"permission_code"`
    Description    string `json:"description"`
    Type           string `json:"type"`
    TypeText       string `json:"type_text"`
    Module         string `json:"module"`
    ModuleText     string `json:"module_text"`
    Status         string `json:"status"`
    StatusText     string `json:"status_text"`
    RoleCount      int    `json:"role_count"`
    CreatedAt      string `json:"created_at"`
}
```

### SysPermissionTreeItem

```go
type SysPermissionTreeItem struct {
    Id       int                       `json:"id"`
    Label    string                    `json:"label"`
    Type     string                    `json:"type"`
    Children []*SysPermissionTreeItem  `json:"children,omitempty"`
    PermissionCode string              `json:"permission_code,omitempty"`
    Module         string              `json:"module,omitempty"`
    Status         string              `json:"status,omitempty"`
}
```

## 实现文件

### API 层
- `app/admin/apis/sys_permission.go`: PermissionList, PermissionTree 接口

### 服务层
- `app/admin/service/sys_permission.go`: GetPermissionList, GetPermissionTree 方法

### DTO 层
- `app/admin/service/dto/sys_permission.go`: SysPermissionGetPageReq, SysPermissionListItem, SysPermissionTreeItem

### 模型层
- `app/admin/models/sys_menu.go`: SysMenu 模型（权限存储在 sys_menu 表中）

## 数据库表

### sys_menu
菜单权限表（存储权限信息）
- `menu_id`: 菜单 ID（主键）
- `menu_name`: 菜单名称（权限名称）
- `permission`: 权限标识
- `type`: 权限类型（1:菜单 2:操作 3:数据）
- `module`: 所属模块
- `status`: 状态（1:禁用 2:启用）
- `remark`: 备注（权限描述）
- `parent_id`: 父级 ID
- `sort`: 排序
- `created_at`: 创建时间
- `updated_at`: 更新时间

### sys_role_menu
角色菜单关联表
- `role_id`: 角色 ID
- `menu_id`: 菜单 ID（权限 ID）

## 使用示例

### 查询用户模块的菜单权限

```bash
curl -X GET "http://localhost:8000/api/v1/permission/list?module=user&type=1&status=2" \
  -H "Authorization: Bearer <token>"
```

### 搜索包含"查看"的权限

```bash
curl -X GET "http://localhost:8000/api/v1/permission/list?keyword=view" \
  -H "Authorization: Bearer <token>"
```

### 获取权限树形结构

```bash
curl -X GET "http://localhost:8000/api/v1/permission/tree?module=user" \
  -H "Authorization: Bearer <token>"
```

### 获取所有启用的权限

```bash
curl -X GET "http://localhost:8000/api/v1/permission/list?status=2&page_size=100" \
  -H "Authorization: Bearer <token>"
```

## 注意事项

1. **权限标识唯一性**：每个权限标识应全局唯一
2. **模块规范**：权限标识应遵循 `模块。操作` 格式
3. **类型区分**：正确区分菜单权限、操作权限、数据权限
4. **状态管理**：禁用的权限不会在角色配置中显示
5. **引用统计**：删除权限前需检查被多少个角色引用
6. **树形展示**：模块节点使用负数 ID 区别于权限节点
7. **排序规则**：按 sort 字段升序排列
8. **分页查询**：大数据量时使用分页避免性能问题

## 最佳实践

1. **权限规划**：提前规划好模块和权限标识
2. **命名规范**：遵循统一的命名规范
3. **类型明确**：正确设置权限类型
4. **描述完整**：填写详细的权限描述
5. **定期审查**：定期审查权限配置的合理性
6. **清理无用权限**：及时清理不再使用的权限
7. **权限测试**：新增权限后进行功能测试
8. **文档维护**：维护权限清单文档
