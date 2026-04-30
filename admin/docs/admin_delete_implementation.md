# 删除管理员接口实现文档

## 接口概述

删除管理员接口用于移除系统中不需要的后台管理员账户，包含关联检查、软删除处理等环节。

## API 接口

### 请求方式

```
POST /api/v1/admin/delete
```

或者

```
DELETE /api/v1/admin/{user_id}
```

### 请求参数

| 参数名 | 类型 | 必填 | 说明 | 默认值 | 校验规则 |
|--------|------|------|------|--------|----------|
| user_id | int | 是 | 管理员 ID | - | 必须大于 0 |
| confirm | boolean | 否 | 确认标识 | - | 必须为 true（如果提供） |
| reason | string | 否 | 删除原因 | - | 最大 255 字符 |

### 请求示例

```json
{
  "user_id": 10,
  "confirm": true,
  "reason": "该管理员已离职"
}
```

### 返回数据结构

```json
{
  "code": 200,
  "msg": "删除成功",
  "data": {
    "success": true,
    "admin_id": 10,
    "username": "admin001",
    "deleted_at": "2024-01-01 16:00:00",
    "message": "删除成功"
  }
}
```

### 返回字段说明

| 字段名 | 类型 | 说明 |
|--------|------|------|
| success | bool | 是否成功 |
| admin_id | int | 管理员 ID |
| username | string | 用户名 |
| deleted_at | string | 删除时间 |
| message | string | 操作提示 |

## 处理流程

### 第一步：管理员定位

1. **查询管理员记录**
   ```sql
   SELECT * FROM sys_user WHERE user_id = ?
   ```

2. **参数校验**
   - 验证 user_id 必须大于 0
   - 验证 confirm 字段（如果提供）必须为 true

### 第二步：系统管理员校验

1. **检查是否为超级管理员**
   - 检查 role_key 是否为"admin"
   - 若是超级管理员返回错误："系统内置管理员不可删除"

2. **保护系统安全**
   - 保证至少有一个超级管理员
   - 避免误删系统内置管理员

### 第三步：自身校验

1. **检查是否删除自己**
   - 比较操作人 ID（delete_by）与目标管理员 ID（user_id）
   - 若是同一人返回错误："不能删除自己的账户"

2. **防止权限真空**
   - 避免操作人删除自己导致权限真空
   - 如需离职应转移权限后由他人操作

### 第四步：关联检查（可选）

1. **检查进行中的任务**
   - 检查是否有待审批的订单
   - 检查是否有待处理的任务
   - 检查是否有未完成的流程

2. **业务关联检查**
   - 根据实际业务需求添加检查逻辑
   - 确保删除不会影响业务连续性

3. **提示处理**
   - 若有关联任务返回提示
   - 建议先处理完任务再删除

### 第五步：会话清理（可选）

1. **清除登录会话**
   - 删除用户的所有登录 token
   - 清除会话缓存

2. **清除权限缓存**
   - 删除权限缓存数据
   - 强制重新加载权限

3. **强制下线**
   - 使当前登录的管理员立即下线
   - 无法继续使用系统

### 第六步：数据处理（软删除）

1. **设置删除信息**
   ```go
   user.SetDeleteBy(req.DeleteBy)
   ```

2. **执行软删除**
   ```sql
   UPDATE sys_user SET 
       deleted_at = NOW(),
       deleted_by = ?
   WHERE user_id = ?
   ```

3. **保留数据记录**
   - 数据库记录保留
   - 仅标记删除状态
   - 不物理删除数据

### 第七步：角色关联保留

1. **保留关联记录**
   - GORM 的软删除会保留关联记录
   - sys_user_role 表中的记录保留

2. **审计追溯**
   - 保留历史记录用于审计
   - 可以追溯管理员的权限变更

3. **数据完整性**
   - 保持数据完整性
   - 不影响其他关联数据

### 第八步：日志记录（可选）

1. **记录操作日志**
   - 操作人：delete_by
   - 操作时间：当前时间
   - 操作类型：删除管理员

2. **记录详细信息**
   - 被删除的管理员信息（ID、用户名）
   - 删除原因（reason）
   - 操作 IP 地址

3. **审计日志**
   - 记录到审计日志表
   - 便于后续追溯和查询

### 第九步：返回结果

1. **构建返回数据**
   ```go
   result := &dto.SysAdminDeleteResponse{
       Success:   true,
       AdminId:   user.UserId,
       Username:  user.Username,
       DeletedAt: user.DeletedAt.Time.Format("2006-01-02 15:04:05"),
       Message:   "删除成功",
   }
   ```

2. **返回字段**
   - success: 是否成功（布尔值）
   - admin_id: 管理员 ID
   - username: 用户名
   - deleted_at: 删除时间
   - message: 操作提示

## 异常处理

### 1. 管理员不存在

**管理员记录不存在**
- 错误码：500
- 提示信息：管理员记录不存在
- 触发条件：根据 user_id 查询不到记录

**管理员已被删除**
- 错误码：500
- 提示信息：管理员已被删除
- 触发条件：管理员已被软删除

### 2. 超级管理员保护

**系统内置管理员不可删除**
- 错误码：500
- 提示信息：系统内置管理员不可删除
- 触发条件：尝试删除超级管理员（role_key="admin"）

### 3. 自身保护

**不能删除自己的账户**
- 错误码：500
- 提示信息：不能删除自己的账户
- 触发条件：操作人尝试删除自己

### 4. 确认标识异常

**请确认删除操作**
- 错误码：500
- 提示信息：请确认删除操作
- 触发条件：confirm 为 false

### 5. 权限异常

**权限不足**
- 错误码：500
- 提示信息：您没有权限删除该管理员
- 触发条件：操作人没有管理员管理权限

### 6. 系统异常

**删除管理员失败**
- 错误码：500
- 提示信息：删除管理员失败
- 触发条件：数据库操作失败

## 软删除说明

### 1. 删除方式

**软删除**
- 数据库记录保留
- 仅标记删除状态
- 不物理删除数据

**删除标记**
- deleted_at：删除时间
- deleted_by：删除操作人

### 2. 数据状态

**删除后的状态**
- 不显示在管理员列表中
- 可通过筛选条件查看已删除记录
- 无法登录后台系统

**数据库记录**
- 记录完整保留
- 关联记录保留
- 便于审计追溯

### 3. 恢复机制

**恢复方式**
- 通过后台管理员列表的已删除筛选
- 执行恢复操作清除 deleted_at
- 恢复后管理员可重新登录

**恢复权限**
- 需要管理员管理权限
- 超级管理员可恢复
- 系统管理员可恢复

## 删除后影响

### 1. 登录权限

**无法登录**
- 被删除的管理员将无法再登录后台
- 登录会话已全部清除
- 权限缓存已清除

**Token 失效**
- 所有有效的 token 失效
- 无法使用 API 接口
- 需要重新创建账户

### 2. 数据影响

**创建的数据保留**
- 该管理员创建的数据保留不受影响
- created_by 字段保留原管理员 ID
- 数据所有权不变

**更新的数据**
- updated_by 字段保留原管理员 ID
- 历史记录保持不变
- 数据完整性不受影响

### 3. 业务影响

**进行中的任务**
- 建议删除前确认无进行中任务
- 如有任务应转移给其他管理员
- 避免业务中断

**关联的数据**
- 关联的角色记录保留
- 关联的日志记录保留
- 关联的审计记录保留

### 4. 审计追溯

**操作记录**
- 删除操作记录在审计日志中
- 包含操作人、删除时间、删除原因
- 便于后续追溯

**历史记录**
- 管理员的历史操作记录保留
- 角色关联记录保留
- 登录日志记录保留

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
    DeletedAt     gorm.DeletedAt `gorm:"index"`
    DeletedBy     int       `gorm:"index"`
    SysRoles      []SysRole `gorm:"many2many:sys_user_role"`
    models.ControlBy
    models.ModelTime
}
```

### ControlBy 模型

```go
type ControlBy struct {
    CreateBy int `gorm:"index"`
    UpdateBy int `gorm:"index"`
    DeleteBy int `gorm:"index"`
}
```

## 请求参数 DTO

### SysAdminDeleteReq

```go
type SysAdminDeleteReq struct {
    UserId  int   `json:"user_id" binding:"required"`
    Confirm *bool `json:"confirm"`
    Reason  string `json:"reason"`
    ControlBy
}
```

### SysAdminDeleteResponse

```go
type SysAdminDeleteResponse struct {
    Success   bool   `json:"success"`
    AdminId   int    `json:"admin_id"`
    Username  string `json:"username"`
    DeletedAt string `json:"deleted_at"`
    Message   string `json:"message"`
}
```

## 最佳实践

### 1. 删除前检查

- 确认管理员无进行中任务
- 转移待处理的工作
- 通知相关人员进行工作交接

### 2. 权限管理

- 确保操作人有删除权限
- 超级管理员不可删除
- 不能删除自己的账户

### 3. 数据保护

- 使用软删除保护数据
- 保留审计追溯记录
- 便于后续恢复

### 4. 日志记录

- 记录删除操作
- 记录删除原因
- 便于问题追溯

### 5. 会话管理

- 清除所有登录会话
- 清除权限缓存
- 强制下线

### 6. 通知机制

- 删除前通知管理员
- 确认无进行中任务
- 安排工作交接

## 示例代码

### cURL 示例

```bash
curl -X POST "http://localhost:8000/api/v1/admin/delete" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 10,
    "confirm": true,
    "reason": "该管理员已离职"
  }'
```

### JavaScript 示例

```javascript
fetch('http://localhost:8000/api/v1/admin/delete', {
  method: 'POST',
  headers: {
    'Authorization': 'Bearer <token>',
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    user_id: 10,
    confirm: true,
    reason: '该管理员已离职'
  })
})
.then(response => response.json())
.then(data => console.log(data))
.catch(error => console.error('Error:', error));
```

### Python 示例

```python
import requests

url = "http://localhost:8000/api/v1/admin/delete"
headers = {
    "Authorization": "Bearer <token>",
    "Content-Type": "application/json"
}
data = {
    "user_id": 10,
    "confirm": True,
    "reason": "该管理员已离职"
}

response = requests.post(url, json=data, headers=headers)
print(response.json())
```

## 注意事项

### 1. 超级管理员保护

- 超级管理员不可删除
- 保证系统安全性
- 避免系统失控

### 2. 自身保护

- 不能删除自己的账户
- 避免权限真空
- 需要他人操作

### 3. 软删除

- 数据记录保留
- 便于审计追溯
- 支持恢复操作

### 4. 会话清理

- 清除所有登录会话
- 清除权限缓存
- 强制下线

### 5. 关联检查

- 检查进行中的任务
- 避免业务中断
- 确保数据完整性

### 6. 日志记录

- 记录删除操作
- 记录删除原因
- 便于问题追溯

## 常见问题

### Q1: 删除管理员后数据会丢失吗？

A: 不会，采用软删除方式，数据库记录保留，仅标记删除状态。

### Q2: 删除的管理员可以恢复吗？

A: 可以，通过后台管理员列表的已删除筛选找到记录，执行恢复操作。

### Q3: 为什么不能删除超级管理员？

A: 超级管理员是系统内置管理员，删除会导致系统失控，保证系统安全。

### Q4: 为什么不能删除自己的账户？

A: 避免权限真空，如需离职应转移权限后由他人操作删除。

### Q5: 删除管理员后登录会话会怎样？

A: 所有登录会话会被清除，管理员会被强制下线，无法继续使用系统。

### Q6: 删除管理员需要哪些权限？

A: 需要管理员管理权限，具体权限标识根据系统配置确定。

### Q7: confirm 字段必须传吗？

A: confirm 字段是选填的，但如果提供必须为 true，用于确认删除操作。

### Q8: 删除原因必须填写吗？

A: reason 字段是选填的，但建议填写删除原因便于审计追溯。

### Q9: 删除后角色关联会怎样？

A: 角色关联记录会保留用于审计追溯，但管理员不再拥有任何权限。

### Q10: 如何确认删除操作的影响？

A: 删除前检查管理员的进行中任务，确认无影响后再执行删除操作。
