# 修改管理员接口实现文档

## 接口概述

修改管理员接口用于更新已有管理员账户的配置信息，包括基本信息、关联角色、状态等，不涉及密码修改。

## API 接口

### 请求方式

```
PUT /api/v1/admin/update
```

### 请求参数

| 参数名 | 类型 | 必填 | 说明 | 默认值 | 校验规则 |
|--------|------|------|------|--------|----------|
| user_id | int | 是 | 管理员 ID | - | 必须大于 0 |
| nickname | string | 是 | 昵称姓名 | - | 最大 50 字符 |
| email | string | 否 | 邮箱地址 | - | 邮箱格式，最大 100 字符 |
| phone | string | 否 | 手机号 | - | 11 位数字 |
| avatar | string | 否 | 头像 URL | - | 最大 255 字符 |
| role_ids | array | 是 | 关联角色 ID 列表 | - | 至少选择一个角色 |
| status | string | 否 | 管理员状态 | - | 1:禁用 2:正常 |
| remark | string | 否 | 备注 | - | 最大 255 字符 |

### 可修改字段

- ✅ **nickname**：昵称姓名
- ✅ **email**：邮箱地址
- ✅ **phone**：手机号
- ✅ **avatar**：头像 URL
- ✅ **role_ids**：关联角色 ID 列表
- ✅ **status**：管理员状态
- ✅ **remark**：备注

### 不可修改字段

- ❌ **username**：用户名（不可修改）
- ❌ **password**：密码（不可修改，使用单独接口）
- ❌ **admin_id**：管理员 ID（不可修改）
- ❌ **created_at**：创建时间（不可修改）
- ❌ **created_by**：创建人（不可修改）

### 请求示例

```json
{
  "user_id": 10,
  "nickname": "更新后的昵称",
  "email": "updated@example.com",
  "phone": "13987654321",
  "role_ids": [2, 4],
  "status": "2",
  "remark": "更新备注信息"
}
```

### 返回数据结构

```json
{
  "code": 200,
  "msg": "更新成功",
  "data": {
    "admin_id": 10,
    "user_id": 10,
    "username": "admin001",
    "nickname": "更新后的昵称",
    "email": "updated@example.com",
    "phone": "13987654321",
    "avatar": "https://example.com/avatar.jpg",
    "role_list": [
      {
        "role_id": 2,
        "role_name": "系统管理员",
        "role_code": "system_admin"
      },
      {
        "role_id": 4,
        "role_name": "内容管理员",
        "role_code": "content_admin"
      }
    ],
    "role_ids": [2, 4],
    "status": "2",
    "status_text": "正常",
    "remark": "更新备注信息",
    "updated_at": "2024-01-01 15:30:00",
    "updated_by": "admin",
    "is_super": false,
    "affected_roles": {
      "added": [
        {
          "role_id": 4,
          "role_name": "内容管理员",
          "role_code": "content_admin"
        }
      ],
      "removed": [
        {
          "role_id": 3,
          "role_name": "运营管理员",
          "role_code": "operator"
        }
      ]
    }
  }
}
```

### 返回字段说明

| 字段名 | 类型 | 说明 |
|--------|------|------|
| admin_id | int | 管理员 ID |
| user_id | int | 用户 ID |
| username | string | 用户名（不可修改） |
| nickname | string | 昵称姓名 |
| email | string | 邮箱 |
| phone | string | 手机号 |
| avatar | string | 头像 URL |
| role_list | array | 关联角色列表 |
| role_ids | array | 角色 ID 列表 |
| status | string | 管理员状态 |
| status_text | string | 状态文本 |
| remark | string | 备注 |
| updated_at | string | 更新时间 |
| updated_by | string | 更新人 |
| is_super | bool | 是否超级管理员 |
| affected_roles | object | 角色变更信息 |
| affected_roles.added | array | 新增的角色列表 |
| affected_roles.removed | array | 移除的角色列表 |

## 处理流程

### 第一步：参数解析

1. **解析请求参数**
   - 从请求体中提取 user_id
   - 解析可修改字段：nickname、email、phone、avatar、role_ids、status、remark
   - 获取操作人 ID（update_by）

2. **参数验证**
   - 验证 user_id 必须大于 0
   - 验证 nickname 不能为空
   - 验证 role_ids 至少有一个角色

### 第二步：管理员校验

1. **查询目标管理员**
   ```sql
   SELECT * FROM sys_user WHERE user_id = ?
   ```

2. **校验管理员是否存在**
   - 若查询失败返回错误："管理员不存在"

3. **超级管理员校验**
   - 检查 role_key 是否为"admin"
   - 若是超级管理员返回错误："系统内置管理员不可修改基本信息"

4. **检查是否被删除**
   - 检查 deleted_at 字段
   - 若已被删除返回错误："管理员记录不存在"

### 第三步：信息校验

1. **手机号唯一性校验**
   - 如果修改了手机号（与原始值不同）
   - 校验新手机号是否被其他管理员使用
   ```sql
   SELECT COUNT(*) FROM sys_user WHERE phone = ? AND user_id != ? AND deleted_at IS NULL
   ```
   - 若已使用返回错误："该手机号已被其他管理员使用"

2. **邮箱唯一性校验**
   - 如果修改了邮箱（与原始值不同）
   - 校验新邮箱是否被其他管理员使用
   ```sql
   SELECT COUNT(*) FROM sys_user WHERE email = ? AND user_id != ? AND deleted_at IS NULL
   ```
   - 若已使用返回错误："该邮箱已被其他管理员使用"

### 第四步：角色校验

1. **查询角色信息**
   ```sql
   SELECT * FROM sys_role WHERE role_id IN (?, ?, ...)
   ```

2. **校验角色是否存在**
   - 验证查询到的角色数量与请求的角色数量一致
   - 若不一致返回错误："所选角色不存在或已被禁用"

3. **校验角色状态**
   - 检查每个角色的 status 字段
   - 若角色状态为禁用（status=1）返回错误："所选角色不存在或已被禁用"

### 第五步：自身权限校验

1. **检查是否修改自己的信息**
   - 比较操作人 ID（update_by）与目标管理员 ID（user_id）
   - 如果是同一人，执行自身权限校验

2. **不能禁用自己的账户**
   - 检查 status 是否设置为"1"（禁用）
   - 若是返回错误："不能禁用自己的账户"

3. **不能将自己降权**（可选）
   - 获取当前用户的角色列表
   - 比较新旧角色的权限范围
   - 若新角色的权限范围更小返回错误："不能将自己修改为权限更小的角色"

### 第六步：记录原始角色信息

1. **查询原始角色**
   ```sql
   SELECT * FROM sys_role WHERE role_id IN (SELECT role_id FROM sys_user_role WHERE user_id = ?)
   ```

2. **保存原始角色列表**
   - 用于后续计算角色变更
   - 用于日志记录

### 第七步：数据更新

1. **更新用户信息**
   ```sql
   UPDATE sys_user SET 
       nick_name = ?,
       email = ?,
       phone = ?,
       avatar = ?,
       status = ?,
       remark = ?,
       update_by = ?,
       updated_at = NOW()
   WHERE user_id = ?
   ```

2. **不修改的字段**
   - username：用户名保持不变
   - password：密码保持不变
   - created_at：创建时间保持不变
   - created_by：创建人保持不变

### 第八步：角色重置

1. **清除原有角色关联**
   ```sql
   DELETE FROM sys_user_role WHERE user_id = ?
   ```

2. **新增新的角色关联**
   ```sql
   INSERT INTO sys_user_role (user_id, role_id, created_at) VALUES (?, ?, ?)
   ```

3. **原子性保证**
   - 使用 GORM 的 Association 操作
   - 先 Clear() 再 Append()
   - 若失败会回滚到原始状态

### 第九步：缓存清理（可选）

1. **清除权限缓存**
   - 删除用户 ID 对应的权限缓存
   - 强制重新加载权限

2. **清除登录会话**
   - 删除用户的登录会话
   - 强制重新登录获取权限

### 第十步：计算角色变更

1. **计算新增的角色**
   - 对比新角色列表与原始角色列表
   - 找出新增的角色 ID
   - 构建新增角色信息对象

2. **计算移除的角色**
   - 对比原始角色列表与新角色列表
   - 找出被移除的角色 ID
   - 构建移除角色信息对象

3. **构建返回结构**
   ```json
   {
     "affected_roles": {
       "added": [...],
       "removed": [...]
     }
   }
   ```

### 第十一步：日志记录（可选）

1. **记录操作日志**
   - 操作人：update_by
   - 操作时间：当前时间
   - 操作类型：修改管理员

2. **记录字段变更**
   - 记录修改前后的字段值对比
   - 记录角色变更信息

3. **审计日志**
   - 记录到审计日志表
   - 便于后续追溯和查询

### 第十二步：返回结果

1. **构建返回数据**
   - 返回修改成功的管理员详情
   - 包含 updated_fields 修改的字段列表
   - 包含 affected_roles 角色变更信息

2. **返回字段**
   - success: 是否成功（布尔值）
   - admin_id: 管理员 ID
   - updated_fields: 修改的字段列表
   - updated_at: 更新时间
   - affected_roles: 变更的角色列表
   - message: 操作提示

## 异常处理

### 1. 管理员不存在

**管理员记录不存在**
- 错误码：500
- 提示信息：管理员不存在
- 触发条件：根据 user_id 查询不到记录

**管理员已被删除**
- 错误码：500
- 提示信息：管理员记录不存在
- 触发条件：管理员已被软删除

### 2. 超级管理员保护

**系统内置管理员不可修改**
- 错误码：500
- 提示信息：系统内置管理员不可修改基本信息
- 触发条件：尝试修改超级管理员（role_key="admin"）

### 3. 手机号异常

**手机号已被使用**
- 错误码：500
- 提示信息：该手机号已被其他管理员使用
- 触发条件：新手机号已被其他管理员使用

### 4. 邮箱异常

**邮箱已被使用**
- 错误码：500
- 提示信息：该邮箱已被其他管理员使用
- 触发条件：新邮箱已被其他管理员使用

### 5. 角色异常

**角色不存在或已禁用**
- 错误码：500
- 提示信息：所选角色不存在或已被禁用
- 触发条件：
  - 请求的角色 ID 不存在
  - 角色状态为禁用

**未选择角色**
- 错误码：500
- 提示信息：至少选择一个角色
- 触发条件：role_ids 为空数组

### 6. 权限异常

**不能禁用自己的账户**
- 错误码：500
- 提示信息：不能禁用自己的账户
- 触发条件：操作人尝试禁用自己

**不能将自己降权**
- 错误码：500
- 提示信息：不能将自己修改为权限更小的角色
- 触发条件：操作人尝试减少自己的角色权限

**权限不足**
- 错误码：500
- 提示信息：您没有权限分配该角色
- 触发条件：操作人没有权限分配某些角色

### 7. 系统异常

**数据库错误**
- 错误码：500
- 提示信息：更新管理员失败
- 触发条件：数据库操作失败

**角色关联失败**
- 错误码：500
- 提示信息：角色关联失败
- 触发条件：写入角色关联表失败

**获取原角色信息失败**
- 错误码：500
- 提示信息：获取原角色信息失败
- 触发条件：查询原始角色失败

## 角色变更影响

### 1. 权限变更

**新增角色**
- 管理员获得新角色的所有权限
- 可以访问新角色授权的菜单和功能
- 可以执行新角色授权的操作

**移除角色**
- 管理员失去被移除角色的权限
- 无法访问被移除角色授权的菜单
- 无法执行被移除角色授权的操作

### 2. 即时生效

- 角色变更后权限即时生效
- 建议让管理员重新登录
- 或清除会话强制重新获取权限

### 3. 影响评估

**新增角色的影响**
- 管理员权限范围扩大
- 可以执行更多操作
- 一般不需要特别通知

**移除角色的影响**
- 管理员权限范围缩小
- 可能影响正在进行的工作
- 建议提前通知管理员

### 4. 最佳实践

- 角色变更前评估影响范围
- 提前通知管理员权限变更
- 在低峰期执行重大权限变更
- 变更后验证权限是否正确

## 原子性保证

### 1. 事务机制

- 使用 GORM 的 Association 操作
- Clear() 和 Append() 在同一事务中
- 失败时自动回滚

### 2. 数据一致性

- 先删除原有角色关联
- 再新增新的角色关联
- 保证角色关联的完整性

### 3. 避免权限丢失

- 若新增角色失败，回滚删除操作
- 原始角色关联保持不变
- 避免出现权限真空

### 4. 避免权限残留

- 确保清除所有原有角色
- 只保留新的角色关联
- 避免权限混乱

## 数据模型

### SysUser 模型

```go
type SysUser struct {
    UserId        int       `gorm:"primaryKey;autoIncrement"`
    Username      string    `gorm:"size:64"`
    Password      string    `gorm:"size:128"`
    NickName      string    `gorm:"size:128"`
    Phone         string    `gorm:"size:20"`
    Email         string    `gorm:"size:128"`
    Avatar        string    `gorm:"size:255"`
    Status        string    `gorm:"size:4"`
    Remark        string    `gorm:"size:255"`
    UpdateBy      int       `gorm:"index"`
    SysRoles      []SysRole `gorm:"many2many:sys_user_role"`
    models.ControlBy
    models.ModelTime
}
```

### SysRole 模型

```go
type SysRole struct {
    RoleId    int       `gorm:"primaryKey;autoIncrement"`
    RoleName  string    `gorm:"size:128"`
    RoleKey   string    `gorm:"size:128"`
    Status    string    `gorm:"size:1"`
    Remark    string    `gorm:"size:255"`
    SysMenus  []SysMenu `gorm:"many2many:sys_role_menu"`
    models.ControlBy
    models.ModelTime
}
```

## 请求参数 DTO

### SysAdminUpdateReq

```go
type SysAdminUpdateReq struct {
    UserId   int    `json:"user_id" binding:"required"`
    Nickname string `json:"nickname" binding:"required,max=50"`
    Email    string `json:"email" binding:"omitempty,email,max=100"`
    Phone    string `json:"phone" binding:"omitempty,len=11"`
    Avatar   string `json:"avatar" binding:"omitempty,max=255"`
    RoleIds  []int  `json:"role_ids" binding:"required,min=1"`
    Status   string `json:"status" binding:"omitempty,oneof=1 2"`
    Remark   string `json:"remark" binding:"omitempty,max=255"`
    ControlBy
}
```

## 角色变更返回结构

### AffectedRoles

```go
type AffectedRoles struct {
    Added   []SysAdminRoleItem `json:"added"`
    Removed []SysAdminRoleItem `json:"removed"`
}

type SysAdminRoleItem struct {
    RoleId   int    `json:"role_id"`
    RoleName string `json:"role_name"`
    RoleCode string `json:"role_code"`
}
```

## 最佳实践

### 1. 修改前评估

- 评估修改的影响范围
- 特别是角色变更的影响
- 提前通知相关人员

### 2. 权限管理

- 遵循最小权限原则
- 定期审查权限配置
- 及时回收不需要的权限

### 3. 审计追溯

- 记录所有修改操作
- 记录修改前后的值
- 便于问题追溯

### 4. 缓存管理

- 修改后及时清除缓存
- 强制重新获取权限
- 保证权限一致性

### 5. 会话管理

- 重要权限变更后清除会话
- 强制重新登录
- 避免权限混乱

## 示例代码

### cURL 示例

```bash
curl -X PUT "http://localhost:8000/api/v1/admin/update" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 10,
    "nickname": "更新后的昵称",
    "email": "updated@example.com",
    "phone": "13987654321",
    "role_ids": [2, 4],
    "status": "2"
  }'
```

### JavaScript 示例

```javascript
fetch('http://localhost:8000/api/v1/admin/update', {
  method: 'PUT',
  headers: {
    'Authorization': 'Bearer <token>',
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    user_id: 10,
    nickname: '更新后的昵称',
    email: 'updated@example.com',
    phone: '13987654321',
    role_ids: [2, 4],
    status: '2'
  })
})
.then(response => response.json())
.then(data => console.log(data))
.catch(error => console.error('Error:', error));
```

### Python 示例

```python
import requests

url = "http://localhost:8000/api/v1/admin/update"
headers = {
    "Authorization": "Bearer <token>",
    "Content-Type": "application/json"
}
data = {
    "user_id": 10,
    "nickname": "更新后的昵称",
    "email": "updated@example.com",
    "phone": "13987654321",
    "role_ids": [2, 4],
    "status": "2"
}

response = requests.put(url, json=data, headers=headers)
print(response.json())
```

## 注意事项

### 1. 用户名不可修改

- 用户名是管理员的唯一标识
- 修改用户名可能导致权限混乱
- 如需修改用户名，应删除后重新创建

### 2. 密码不可修改

- 密码修改使用单独接口
- 密码修改需要额外验证
- 密码修改需要记录日志

### 3. 超级管理员保护

- 不能修改超级管理员信息
- 保证系统安全性
- 避免误操作

### 4. 自身权限保护

- 不能禁用自己的账户
- 不能将自己降权
- 避免操作人失去权限

### 5. 角色变更影响

- 角色变更会影响权限
- 建议提前通知
- 变更后验证权限

### 6. 缓存清理

- 修改后清除权限缓存
- 清除登录会话
- 保证权限一致性

## 常见问题

### Q1: 为什么用户名不能修改？

A: 用户名是管理员的唯一标识，修改可能导致权限关联混乱。如需修改，建议删除后重新创建。

### Q2: 如何修改密码？

A: 密码修改使用单独的接口，不在修改管理员接口中处理。

### Q3: 修改角色后需要重新登录吗？

A: 建议重新登录以确保权限即时生效。系统会清除缓存和会话。

### Q4: 可以修改自己的角色吗？

A: 可以，但不能将自己降权或禁用自己。

### Q5: 修改失败后角色关联会回滚吗？

A: 会的，使用事务保证原子性，失败时会回滚到原始状态。

### Q6: 如何查看角色变更详情？

A: 返回结果中的 affected_roles 字段包含新增和移除的角色列表。

### Q7: 修改管理员信息需要哪些权限？

A: 需要管理员管理权限，具体权限标识根据系统配置确定。

### Q8: 修改后其他管理员能看到变更吗？

A: 可以，修改后的信息会即时显示在管理员列表中。
