# 创建角色接口实现文档

## 接口概述

创建角色接口用于新增系统角色，配置角色名称、权限范围、角色描述等信息，供管理员分配给后台用户使用。

## API 接口

### 请求方式

```
POST /api/v1/role/create
```

### 请求参数

| 参数名 | 类型 | 必填 | 说明 | 默认值 | 校验规则 |
|--------|------|------|------|--------|----------|
| role_name | string | 是 | 角色名称 | - | 不超过 50 字符 |
| role_code | string | 否 | 角色编码 | 自动生成 | 不超过 30 字符，全局唯一 |
| description | string | 否 | 角色描述 | - | 不超过 200 字符 |
| permission_list | []string | 是 | 权限标识列表 | - | 至少包含一个权限 |
| status | string | 否 | 角色状态 | 2（启用） | 1:禁用 2:正常 |
| source_role_id | int | 否 | 源角色 ID | - | 用于权限继承 |

### 请求示例

```json
{
  "role_name": "运营管理员",
  "role_code": "operator",
  "description": "负责日常运营操作",
  "permission_list": ["user.view", "content.view", "content.edit"],
  "status": "2"
}
```

### 返回数据结构

```json
{
  "code": 200,
  "msg": "创建成功",
  "data": {
    "role_id": 5,
    "role_name": "运营管理员",
    "role_code": "operator",
    "permission_list": ["user.view", "content.view", "content.edit"],
    "created_at": "2024-01-01 12:00:00",
    "message": "创建成功"
  }
}
```

### 返回字段说明

| 字段名 | 类型 | 说明 |
|--------|------|------|
| role_id | int | 角色 ID |
| role_name | string | 角色名称 |
| role_code | string | 角色编码 |
| permission_list | []string | 权限标识列表 |
| created_at | string | 创建时间 |
| message | string | 操作提示 |

## 处理流程

### 1. 参数校验

验证必填字段和格式规范：
- **角色名称**：不能为空，不超过 50 字符
- **角色描述**：不超过 200 字符
- **权限列表**：不能为空，至少包含一个权限
- **角色编码**：不超过 30 字符

### 2. 编码生成

若未传入角色编码，根据角色名称自动生成：
- 中文字符：转换为拼音首字母（简化实现：直接移除）
- 英文字符：转换为小写
- 特殊字符：移除
- 长度限制：不超过 30 字符

示例：
- "超级管理员" → "admin"（简化处理）
- "运营管理员" → "operator"
- "Admin" → "admin"

### 3. 唯一性校验

检查角色编码和角色名称是否重复：
- 查询 `sys_role` 表，校验 `role_key` 是否已存在
- 查询 `sys_role` 表，校验 `role_name` 是否已存在
- 若重复则返回错误提示

### 4. 权限校验

验证权限标识的有效性：
- 遍历权限列表，查询 `sys_menu` 表验证每个权限标识是否存在
- 若包含无效权限，返回错误提示
- （可选）校验操作人是否具备创建该角色的权限

### 5. 数据入库

将角色信息写入数据库：
- 创建 `sys_role` 记录
- 字段包括：角色名称、角色编码、角色描述、状态、创建人、创建时间
- 默认状态为启用（"2"）

### 6. 权限关联

建立角色与权限的关联关系：
- 根据权限标识列表查询对应的菜单记录
- 将菜单关联到角色（写入 `sys_role_menu` 关联表）
- 建立多对多关系

### 7. 日志记录

记录创建操作信息：
- 操作人 ID
- 创建的角色信息（ID、名称、编码）
- 关联的权限列表
- 创建时间

### 8. 返回结果

返回创建成功的角色信息：
- 角色 ID
- 角色名称
- 角色编码
- 权限标识列表
- 创建时间
- 操作提示

## 数据模型

### SysRoleCreateReq

```go
type SysRoleCreateReq struct {
    RoleName       string   `json:"role_name" binding:"required,max=50"`
    RoleCode       string   `json:"role_code" binding:"max=30"`
    Description    string   `json:"description" binding:"max=200"`
    PermissionList []string `json:"permission_list" binding:"required,min=1"`
    Status         string   `json:"status" binding:"oneof=1 2"`
    SourceRoleId   int      `json:"source_role_id"`
    common.ControlBy
}
```

### SysRoleCreateResponse

```go
type SysRoleCreateResponse struct {
    RoleId         int      `json:"role_id"`
    RoleName       string   `json:"role_name"`
    RoleCode       string   `json:"role_code"`
    PermissionList []string `json:"permission_list"`
    CreatedAt      string   `json:"created_at"`
    Message        string   `json:"message"`
}
```

## 异常处理

| 异常场景 | 错误码 | 错误提示 |
|----------|--------|----------|
| 角色名称为空 | 500 | 角色名称不能为空 |
| 角色名称超长 | 500 | 角色名称不能超过 50 字符 |
| 角色描述超长 | 500 | 角色描述不能超过 200 字符 |
| 角色编码超长 | 500 | 角色编码不能超过 30 字符 |
| 角色编码重复 | 500 | 该角色编码已存在 |
| 角色名称重复 | 500 | 该角色名称已存在 |
| 权限列表为空 | 500 | 至少需选择一个权限 |
| 包含无效权限 | 500 | 包含无效的权限标识 |
| 数据库错误 | 500 | 创建角色失败 |
| 权限关联失败 | 500 | 权限关联失败 |

## 权限继承

支持从已有角色继承权限配置：

### 实现方式

1. 前端传入 `source_role_id` 参数指定源角色
2. 后端查询源角色的权限列表
3. 将源角色的权限列表复制到新角色
4. 可在此基础上增删权限形成新角色

### 注意事项

- 继承关系仅复制权限列表，不形成父子关联
- 源角色后续变更不影响已创建的新角色
- 继承的权限可以在创建时调整

### 示例

```json
{
  "role_name": "高级运营",
  "source_role_id": 3,
  "permission_list": ["user.view", "user.edit", "content.view", "content.edit", "report.view"]
}
```

## 默认角色

系统初始化时自动创建以下默认角色：

| 角色名称 | 角色编码 | 权限范围 | 可删除 | 可修改 |
|----------|----------|----------|--------|--------|
| 超级管理员 | admin | 所有权限 | 否 | 否 |
| 系统管理员 | system_admin | 大部分管理权限 | 是 | 是 |
| 普通管理员 | user | 基础查看权限 | 是 | 是 |

## 创建后操作

角色创建完成后可以执行以下操作：

1. **编辑角色**：完善角色描述
2. **调整权限**：增加或移除权限
3. **禁用角色**：暂时不使用该角色
4. **分配角色**：将角色分配给管理员
5. **新建管理员**：创建管理员时选择该角色

## 实现文件

### API 层
- `app/admin/apis/sys_role_list.go`: CreateRole 接口

### 服务层
- `app/admin/service/sys_role.go`: CreateRole 方法、generateRoleCode 辅助函数

### DTO 层
- `app/admin/service/dto/sys_role.go`: SysRoleCreateReq, SysRoleCreateResponse

### 模型层
- `app/admin/models/sys_role.go`: SysRole 模型

## 数据库表

### sys_role
角色主表，存储角色基本信息
- `role_id`: 角色 ID（主键）
- `role_name`: 角色名称
- `role_key`: 角色编码
- `status`: 状态（1:禁用 2:正常）
- `remark`: 角色描述
- `create_by`: 创建人
- `created_at`: 创建时间

### sys_menu
菜单权限表，存储权限标识
- `menu_id`: 菜单 ID
- `permission`: 权限标识
- `menu_name`: 菜单名称

### sys_role_menu
角色菜单关联表
- `role_id`: 角色 ID
- `menu_id`: 菜单 ID

## 使用示例

### 创建基础角色

```bash
curl -X POST "http://localhost:8000/api/v1/role/create" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "role_name": "客服专员",
    "description": "负责客户服务工作",
    "permission_list": ["user.view", "ticket.view", "ticket.edit"]
  }'
```

### 创建带自定义编码的角色

```bash
curl -X POST "http://localhost:8000/api/v1/role/create" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "role_name": "财务管理员",
    "role_code": "finance_admin",
    "description": "负责财务管理",
    "permission_list": ["finance.view", "finance.edit", "report.view"],
    "status": "2"
  }'
```

### 继承权限创建角色

```bash
curl -X POST "http://localhost:8000/api/v1/role/create" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "role_name": "高级运营",
    "source_role_id": 3,
    "description": "高级运营管理员",
    "permission_list": ["user.view", "user.edit", "content.view", "content.edit", "report.view"]
  }'
```

## 注意事项

1. **角色编码唯一性**：角色编码必须全局唯一，建议传入有意义的编码
2. **权限有效性**：确保传入的权限标识在系统中存在
3. **权限范围**：普通管理员不能创建比自己权限更大的角色
4. **默认状态**：未指定状态时默认为启用（"2"）
5. **中文编码**：中文字符会自动转换，建议手动指定英文编码
6. **操作日志**：创建操作会自动记录到操作日志表
