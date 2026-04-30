# 修改角色接口实现文档

## 接口概述

修改角色接口用于更新已有角色的配置信息，包括角色名称、权限范围、角色描述、角色状态等。

## API 接口

### 请求方式

```
PUT /api/v1/role/update
```

### 请求参数

| 参数名 | 类型 | 必填 | 说明 | 默认值 | 校验规则 |
|--------|------|------|------|--------|----------|
| role_id | int | 是 | 角色 ID | - | 必须大于 0 |
| role_name | string | 是 | 角色名称 | - | 不超过 50 字符 |
| role_code | string | 是 | 角色编码 | - | 不超过 30 字符，全局唯一 |
| description | string | 否 | 角色描述 | - | 不超过 200 字符 |
| permission_list | []string | 是 | 权限标识列表 | - | 至少包含一个权限 |
| status | string | 否 | 角色状态 | - | 1:禁用 2:正常 |

### 请求示例

```json
{
  "role_id": 5,
  "role_name": "高级运营管理员",
  "role_code": "senior_operator",
  "description": "负责高级运营管理工作",
  "permission_list": ["user.view", "user.edit", "content.view", "content.edit", "report.view"],
  "status": "2"
}
```

### 返回数据结构

```json
{
  "code": 200,
  "msg": "更新成功",
  "data": {
    "role_id": 5,
    "role_name": "高级运营管理员",
    "role_code": "senior_operator",
    "updated_fields": ["role_name", "permission_list", "remark"],
    "updated_at": "2024-01-01 14:30:00",
    "affected_admins": 3,
    "message": "更新成功"
  }
}
```

### 返回字段说明

| 字段名 | 类型 | 说明 |
|--------|------|------|
| role_id | int | 角色 ID |
| role_name | string | 角色名称 |
| role_code | string | 角色编码 |
| updated_fields | []string | 修改的字段列表 |
| updated_at | string | 更新时间 |
| affected_admins | int | 受影响的管理员数量 |
| message | string | 操作提示 |

## 处理流程

### 1. 参数解析

根据角色 ID 定位目标记录，解析请求体中的修改字段：
- 验证角色 ID 有效性
- 验证必填字段完整性
- 验证字段长度限制

### 2. 角色校验

校验目标角色的合法性和可修改性：
- 查询角色是否存在
- 校验是否为超级管理员角色（权限不可修改）
- 校验角色是否已被删除

### 3. 名称和编码校验

若修改角色名称或编码，进行唯一性校验：
- 查询新名称是否与其他角色重复（排除当前角色）
- 查询新编码是否与其他角色重复（排除当前角色）
- 若重复则返回错误提示

### 4. 权限校验

验证权限标识的有效性：
- 遍历权限列表，查询 `sys_menu` 表验证每个权限标识是否存在
- 若包含无效权限，返回错误提示
- （可选）校验操作人是否具备修改该角色的权限

### 5. 影响评估

评估权限变更对已关联管理员的影响：
- 统计已关联该角色的管理员数量
- 用于返回受影响的管理员数量
- 提示用户可能的影响范围

### 6. 数据更新

开启事务，更新角色表中的修改字段：
- 记录修改的字段列表
- 更新角色名称、编码、描述、状态等字段
- 更新 `updated_at` 时间和 `updated_by` 更新人
- 保存角色记录

### 7. 权限重置

若修改了权限列表，采用原子操作保证一致性：
- **先删除**：清除角色原有的所有菜单关联
- **后新增**：根据新的权限列表重新关联菜单
- **事务保证**：在同一个事务中完成，失败则回滚
- 避免出现权限丢失或权限残留

### 8. 缓存清理

清除与角色相关的缓存数据：
- 清除用户权限缓存
- 清除菜单权限缓存
- 强制用户重新获取权限
- 确保权限变更即时生效

### 9. 日志记录

记录完整的修改操作信息：
- 操作人 ID
- 操作时间
- 修改的角色信息（ID、名称、编码）
- 修改前后的字段值对比
- 便于审计追溯

### 10. 通知推送

若权限范围缩减，向相关管理员推送通知：
- 通知权限变更事实
- 说明被移除的权限
- 建议重新登录生效
- （可选实现）

## 可修改字段

| 字段 | 可修改 | 说明 |
|------|--------|------|
| role_name | ✅ | 角色名称 |
| role_code | ✅ | 角色编码 |
| description | ✅ | 角色描述 |
| permission_list | ✅ | 权限标识列表 |
| status | ✅ | 角色状态 |

## 不可修改字段

| 字段 | 不可修改 | 说明 |
|------|----------|------|
| role_id | ❌ | 角色 ID（主键） |
| created_at | ❌ | 创建时间 |
| created_by | ❌ | 创建人 |

## 特殊角色处理

### 超级管理员角色

- **角色标识**：`role_key = 'admin'` 或 `role_name = '超级管理员'`
- **修改限制**：只能修改名称和描述，不能修改权限范围
- **原因**：超级管理员拥有系统所有权限，修改权限可能导致系统异常

### 正在使用中的角色

- **定义**：已关联管理员的角色
- **修改影响**：权限变更会影响所有已关联的管理员
- **处理方式**：
  - 允许修改
  - 返回受影响的管理员数量
  - 建议在低峰期修改
  - 提前通知相关管理员

## 权限变更影响

### 权限缩减

- **影响**：已关联的管理员将失去被移除权限的操作能力
- **建议**：
  - 提前通知受影响的管理员
  - 在低峰期执行修改
  - 修改后手动刷新权限缓存
  - 紧急修改后可通知用户重新登录

### 权限新增

- **影响**：已关联的管理员自动获得新增权限
- **生效时间**：立即生效（缓存清理后）
- **无需额外操作**

### 原子性保证

权限列表修改采用事务保证：
- **先删后增**：在一个事务中完成
- **失败回滚**：修改失败时恢复原权限列表
- **避免残留**：不会出现权限丢失或权限残留

## 数据模型

### SysRoleUpdateRequest

```go
type SysRoleUpdateRequest struct {
    RoleId         int      `json:"role_id" binding:"required"`
    RoleName       string   `json:"role_name" binding:"required,max=50"`
    RoleCode       string   `json:"role_code" binding:"required,max=30"`
    Description    string   `json:"description" binding:"max=200"`
    PermissionList []string `json:"permission_list" binding:"required,min=1"`
    Status         string   `json:"status" binding:"oneof=1 2"`
    common.ControlBy
}
```

### SysRoleUpdateResponse

```go
type SysRoleUpdateResponse struct {
    RoleId         int      `json:"role_id"`
    RoleName       string   `json:"role_name"`
    RoleCode       string   `json:"role_code"`
    UpdatedFields  []string `json:"updated_fields"`
    UpdatedAt      string   `json:"updated_at"`
    AffectedAdmins int      `json:"affected_admins"`
    Message        string   `json:"message"`
}
```

## 异常处理

| 异常场景 | 错误码 | 错误提示 |
|----------|--------|----------|
| 角色 ID 无效 | 500 | 角色 ID 无效 |
| 角色名称为空 | 500 | 角色名称不能为空 |
| 角色名称超长 | 500 | 角色名称不能超过 50 字符 |
| 角色编码超长 | 500 | 角色编码不能超过 30 字符 |
| 角色描述超长 | 500 | 角色描述不能超过 200 字符 |
| 角色不存在 | 500 | 角色记录不存在 |
| 角色编码重复 | 500 | 该角色编码已被使用 |
| 角色名称重复 | 500 | 该角色名称已被使用 |
| 权限列表为空 | 500 | 至少需选择一个权限 |
| 包含无效权限 | 500 | 包含无效的权限标识 |
| 更新失败 | 500 | 更新角色失败 |
| 权限重置失败 | 500 | 权限重置失败 |
| 权限关联失败 | 500 | 权限关联失败 |

## 实现文件

### API 层
- `app/admin/apis/sys_role_list.go`: UpdateRole 接口

### 服务层
- `app/admin/service/sys_role.go`: UpdateRole 方法

### DTO 层
- `app/admin/service/dto/sys_role.go`: SysRoleUpdateRequest, SysRoleUpdateResponse

### 模型层
- `app/admin/models/sys_role.go`: SysRole 模型

## 数据库表

### sys_role
角色主表
- `role_id`: 角色 ID
- `role_name`: 角色名称
- `role_key`: 角色编码
- `status`: 状态
- `remark`: 角色描述
- `update_by`: 更新人
- `updated_at`: 更新时间

### sys_menu
菜单权限表
- `menu_id`: 菜单 ID
- `permission`: 权限标识

### sys_role_menu
角色菜单关联表
- `role_id`: 角色 ID
- `menu_id`: 菜单 ID

### sys_user_role
用户角色关联表
- `user_id`: 用户 ID
- `role_id`: 角色 ID

## 使用示例

### 修改角色基本信息

```bash
curl -X PUT "http://localhost:8000/api/v1/role/update" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "role_id": 5,
    "role_name": "高级客服专员",
    "role_code": "senior_service",
    "description": "负责高级客户服务工作",
    "permission_list": ["user.view", "ticket.view", "ticket.edit", "report.view"],
    "status": "2"
  }'
```

### 修改角色权限

```bash
curl -X PUT "http://localhost:8000/api/v1/role/update" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "role_id": 5,
    "role_name": "运营管理员",
    "role_code": "operator",
    "permission_list": ["user.view", "user.edit", "content.view", "content.edit", "report.view", "device.view"],
    "status": "2"
  }'
```

### 禁用角色

```bash
curl -X PUT "http://localhost:8000/api/v1/role/update" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "role_id": 5,
    "role_name": "临时管理员",
    "role_code": "temp_admin",
    "permission_list": ["user.view"],
    "status": "1"
  }'
```

## 注意事项

1. **角色编码唯一性**：修改后的角色编码必须全局唯一
2. **角色名称唯一性**：修改后的角色名称不能与其他角色重复
3. **权限有效性**：确保传入的权限标识在系统中存在
4. **事务保证**：权限修改采用事务保证原子性
5. **缓存清理**：修改后会自动清理相关缓存
6. **影响评估**：返回受影响的管理员数量，建议提前通知
7. **超级管理员限制**：超级管理员角色的权限范围不可修改
8. **低峰期修改**：建议在用户活跃度低时修改权限

## 最佳实践

1. **提前通知**：修改权限前通知受影响的管理员
2. **低峰期执行**：选择用户活跃度低的时间段修改
3. **小步修改**：避免一次性修改大量权限
4. **测试验证**：修改后验证权限是否正确生效
5. **记录日志**：详细记录修改原因和修改内容
6. **权限审查**：定期审查角色权限配置的合理性
