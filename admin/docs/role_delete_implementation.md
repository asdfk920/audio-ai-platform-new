# 删除角色接口实现文档

## 接口概述

删除角色接口用于移除系统中不需要的角色，包含关联检查、软删除处理等环节。

## API 接口

### 请求方式

```
POST /api/v1/role/delete
```

### 请求参数

| 参数名 | 类型 | 必填 | 说明 | 默认值 | 校验规则 |
|--------|------|------|------|--------|----------|
| role_id | int | 是 | 角色 ID | - | 必须大于 0 |
| confirm | bool | 否 | 确认标识 | false | 建议为 true 确认删除 |
| reason | string | 否 | 删除原因 | - | 记录删除原因 |

### 请求示例

```json
{
  "role_id": 5,
  "confirm": true,
  "reason": "该角色已不再使用"
}
```

### 返回数据结构

```json
{
  "code": 200,
  "msg": "删除成功",
  "data": {
    "role_id": 5,
    "role_name": "运营管理员",
    "role_code": "operator",
    "backup_info": {
      "backup_time": "2024-01-01 14:30:00",
      "backup_data": {
        "role_id": 5,
        "role_name": "运营管理员",
        "role_code": "operator",
        "description": "负责运营管理工作",
        "permission_list": ["user.view", "user.edit", "content.view"],
        "status": "2",
        "deleted_at": "2024-01-01 14:30:00",
        "deleted_by": "admin",
        "reason": "该角色已不再使用"
      },
      "retention_day": 30
    },
    "affected_admins": 0,
    "message": "删除成功"
  }
}
```

### 返回字段说明

| 字段名 | 类型 | 说明 |
|--------|------|------|
| role_id | int | 角色 ID |
| role_name | string | 角色名称 |
| role_code | string | 角色编码 |
| backup_info | object | 备份信息 |
| backup_info.backup_time | string | 备份时间 |
| backup_info.backup_data | object | 备份数据 |
| backup_info.backup_data.role_id | int | 角色 ID |
| backup_info.backup_data.role_name | string | 角色名称 |
| backup_info.backup_data.role_code | string | 角色编码 |
| backup_info.backup_data.description | string | 角色描述 |
| backup_info.backup_data.permission_list | []string | 权限标识列表 |
| backup_info.backup_data.status | string | 状态 |
| backup_info.backup_data.deleted_at | string | 删除时间 |
| backup_info.backup_data.deleted_by | string | 删除人 |
| backup_info.backup_data.reason | string | 删除原因 |
| backup_info.retention_day | int | 保留期限（天） |
| affected_admins | int | 受影响的管理员数量 |
| message | string | 操作提示 |

## 处理流程

### 1. 角色定位

根据角色 ID 查询角色记录：
- 验证角色 ID 有效性（必须大于 0）
- 查询角色是否存在
- 若不存在返回错误提示

### 2. 系统角色校验

检查是否为不可删除的系统内置角色：
- 校验 `role_key = 'admin'` 或 `role_name = '超级管理员'`
- 若为超级管理员角色返回错误提示
- 系统内置角色不可删除

### 3. 关联检查

查询是否有管理员关联该角色：
- 查询 `sys_user_role` 表统计关联数量
- 若存在关联管理员（`admin_count > 0`）返回错误提示
- 提示"该角色已分配给管理员，请先移除关联后删除"

### 4. 备份记录

将角色信息和关联的权限列表备份：
- 查询角色的所有权限列表（`sys_menu` 关联）
- 获取权限标识列表
- 获取删除人信息
- 准备备份数据结构

### 5. 权限清理

删除角色与权限的关联关系：
- 清除角色的所有菜单关联（`sys_role_menu`）
- 清理权限缓存中该角色的缓存数据
- （可选）清除相关用户权限缓存

### 6. 软删除处理

将角色记录标记为已删除状态：
- 使用 GORM 的软删除功能（`Delete` 方法）
- GORM 自动设置 `deleted_at` 字段
- 手动更新 `deleted_by` 字段记录删除人
- 数据库记录保留但不显示在列表中

### 7. 日志记录

记录完整的删除操作信息：
- 操作人 ID（删除人）
- 操作时间
- 删除的角色信息（ID、名称、编码）
- 删除原因
- 便于审计追溯

### 8. 返回结果

返回删除操作结果和备份信息：
- 角色基本信息（ID、名称、编码）
- 备份信息（备份时间、备份数据、保留期限）
- 受影响的管理员数量
- 操作提示信息

## 删除条件

### 不可删除的角色

1. **超级管理员角色**
   - 角色标识：`role_key = 'admin'` 或 `role_name = '超级管理员'`
   - 原因：系统核心角色，删除会导致系统异常

2. **正在使用中的角色**
   - 定义：已关联管理员的角色（`sys_user_role` 表有记录）
   - 原因：删除会影响已关联的管理员登录和权限
   - 处理方式：先移除所有关联管理员后再删除

### 删除前检查清单

- [ ] 确认角色是否系统内置
- [ ] 确认是否有管理员关联
- [ ] 确认是否有进行中的任务依赖该角色权限
- [ ] 确认操作人权限是否足够
- [ ] 确认已通知受影响的管理员

## 软删除说明

### 软删除机制

- **删除方式**：采用软删除，数据库记录保留
- **删除标记**：通过 `deleted_at` 字段标记删除时间
- **删除人记录**：通过 `deleted_by` 字段记录删除操作人
- **数据显示**：删除后的角色不显示在列表中
- **数据查询**：可通过筛选条件查看已删除角色

### 数据保留

- **备份内容**：角色信息 + 权限列表完整备份
- **保留期限**：默认保留 30 天（可配置）
- **审计追溯**：保留角色编码用于审计追溯
- **自动清理**：备份数据保留一定周期后自动清理

## 删除前影响

### 已关联管理员

- **影响**：已关联该角色的管理员将无法登录或权限异常
- **处理方式**：
  - 删除前通知受影响的管理员
  - 及时调整管理员的角色关联
  - 安排权限调整方案

### 后台功能

- **影响**：依赖该角色权限的后台功能可能无法正常使用
- **处理方式**：
  - 提前评估影响范围
  - 准备替代权限方案
  - 在低峰期执行删除操作

## 数据模型

### SysRoleDeleteRequest

```go
type SysRoleDeleteRequest struct {
    RoleId  int    `json:"role_id" binding:"required"`
    Confirm bool   `json:"confirm"`
    Reason  string `json:"reason"`
    common.ControlBy
}
```

### SysRoleDeleteResponse

```go
type SysRoleDeleteResponse struct {
    RoleId         int         `json:"role_id"`
    RoleName       string      `json:"role_name"`
    RoleCode       string      `json:"role_code"`
    BackupInfo     *BackupInfo `json:"backup_info"`
    AffectedAdmins int         `json:"affected_admins"`
    Message        string      `json:"message"`
}
```

### BackupInfo

```go
type BackupInfo struct {
    BackupTime   string    `json:"backup_time"`
    BackupData   RoleBackup `json:"backup_data"`
    RetentionDay int       `json:"retention_day"`
}
```

### RoleBackup

```go
type RoleBackup struct {
    RoleId         int      `json:"role_id"`
    RoleName       string   `json:"role_name"`
    RoleCode       string   `json:"role_code"`
    Description    string   `json:"description"`
    PermissionList []string `json:"permission_list"`
    Status         string   `json:"status"`
    DeletedAt      string   `json:"deleted_at"`
    DeletedBy      string   `json:"deleted_by"`
    Reason         string   `json:"reason"`
}
```

## 异常处理

| 异常场景 | 错误码 | 错误提示 |
|----------|--------|----------|
| 角色 ID 无效 | 500 | 角色 ID 无效 |
| 角色不存在 | 500 | 角色记录不存在 |
| 超级管理员角色 | 500 | 系统内置角色不可删除 |
| 存在关联管理员 | 500 | 该角色已分配给管理员，请先移除关联后删除 |
| 权限清理失败 | 500 | 权限清理失败 |
| 删除角色失败 | 500 | 删除角色失败 |

## 实现文件

### API 层
- `app/admin/apis/sys_role_list.go`: DeleteRole 接口

### 服务层
- `app/admin/service/sys_role.go`: DeleteRole 方法

### DTO 层
- `app/admin/service/dto/sys_role.go`: SysRoleDeleteRequest, SysRoleDeleteResponse, BackupInfo, RoleBackup

### 模型层
- `app/admin/models/sys_role.go`: SysRole 模型（支持软删除）
- `common/models/by.go`: ControlBy（包含 DeleteBy 字段）
- `common/models/response.go`: ModelTime（包含 DeletedAt 字段）

## 数据库表

### sys_role
角色主表（支持软删除）
- `role_id`: 角色 ID
- `role_name`: 角色名称
- `role_key`: 角色编码
- `status`: 状态
- `remark`: 角色描述
- `delete_by`: 删除人
- `deleted_at`: 删除时间（GORM 软删除字段）

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

### 删除普通角色

```bash
curl -X POST "http://localhost:8000/api/v1/role/delete" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "role_id": 5,
    "confirm": true,
    "reason": "该角色已不再使用"
  }'
```

### 删除角色并记录详细原因

```bash
curl -X POST "http://localhost:8000/api/v1/role/delete" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "role_id": 5,
    "confirm": true,
    "reason": "业务调整，该角色对应的功能已下线"
  }'
```

## 注意事项

1. **超级管理员保护**：超级管理员角色不可删除
2. **关联检查**：有管理员关联的角色不可删除
3. **软删除机制**：删除后数据保留，可通过筛选查看
4. **备份保留**：备份数据保留 30 天，用于审计追溯
5. **权限清理**：删除前会自动清理角色的所有权限关联
6. **日志记录**：完整的删除操作记录，便于审计
7. **通知推送**：建议删除前通知受影响的管理员
8. **低峰期执行**：建议在用户活跃度低时执行删除

## 最佳实践

1. **提前通知**：删除前通知所有受影响的管理员
2. **权限转移**：为已关联管理员安排新的角色
3. **备份确认**：确认备份信息完整后再删除
4. **低峰期执行**：选择用户活跃度低的时间段删除
5. **记录原因**：详细记录删除原因便于追溯
6. **审计检查**：定期检查已删除角色的备份数据
7. **权限审查**：定期审查角色配置的合理性
8. **清理计划**：制定备份数据的定期清理计划
