# 创建管理员接口实现文档

## 接口概述

创建管理员接口用于新增后台管理系统用户，配置账户信息、关联角色等，供运营人员登录管理后台使用。

## API 接口

### 请求方式

```
POST /api/v1/admin/create
```

### 请求参数

| 参数名 | 类型 | 必填 | 说明 | 默认值 | 校验规则 |
|--------|------|------|------|--------|----------|
| username | string | 是 | 用户名 | - | 6 到 20 位英文字母或数字 |
| password | string | 是 | 密码 | - | 8 到 20 位，必须包含大小写字母和数字 |
| nickname | string | 是 | 昵称姓名 | - | 最大 50 字符 |
| email | string | 否 | 邮箱地址 | - | 邮箱格式，最大 100 字符 |
| phone | string | 否 | 手机号 | - | 11 位数字 |
| avatar | string | 否 | 头像 URL | - | 最大 255 字符 |
| role_ids | array | 是 | 关联角色 ID 列表 | - | 至少选择一个角色 |
| status | string | 否 | 管理员状态 | 2（正常） | 1:禁用 2:正常 |
| remark | string | 否 | 备注 | - | 最大 255 字符 |

### 请求示例

```json
{
  "username": "admin001",
  "password": "Admin123",
  "nickname": "运营管理员",
  "email": "admin001@example.com",
  "phone": "13812345678",
  "role_ids": [2, 3],
  "status": "2",
  "remark": "新创建的运营管理员"
}
```

### 返回数据结构

```json
{
  "code": 200,
  "msg": "创建成功",
  "data": {
    "admin_id": 10,
    "user_id": 10,
    "username": "admin001",
    "nickname": "运营管理员",
    "email": "admin001@example.com",
    "phone": "13812345678",
    "avatar": "https://example.com/avatar.jpg",
    "role_list": [
      {
        "role_id": 2,
        "role_name": "系统管理员",
        "role_code": "system_admin"
      },
      {
        "role_id": 3,
        "role_name": "运营管理员",
        "role_code": "operator"
      }
    ],
    "role_ids": [2, 3],
    "status": "2",
    "status_text": "正常",
    "remark": "新创建的运营管理员",
    "created_at": "2024-01-01 12:00:00",
    "created_by": "admin",
    "is_super": false
  }
}
```

### 返回字段说明

| 字段名 | 类型 | 说明 |
|--------|------|------|
| admin_id | int | 管理员 ID |
| user_id | int | 用户 ID |
| username | string | 用户名 |
| nickname | string | 昵称姓名 |
| email | string | 邮箱 |
| phone | string | 手机号 |
| avatar | string | 头像 URL |
| role_list | array | 关联角色列表 |
| role_list[].role_id | int | 角色 ID |
| role_list[].role_name | string | 角色名称 |
| role_list[].role_code | string | 角色编码 |
| role_ids | array | 角色 ID 列表 |
| status | string | 管理员状态（1:禁用 2:正常） |
| status_text | string | 状态文本 |
| remark | string | 备注 |
| created_at | string | 创建时间 |
| created_by | string | 创建人 |
| is_super | bool | 是否超级管理员 |

## 处理流程

### 第一步：参数校验

1. **必填字段校验**
   - 验证 username（用户名）不为空
   - 验证 password（密码）不为空
   - 验证 nickname（昵称）不为空
   - 验证 role_ids（角色列表）不为空

2. **用户名格式校验**
   - 长度必须在 6 到 20 位之间
   - 只能包含英文字母和数字
   - 不允许包含特殊字符

3. **密码强度校验**
   - 长度必须在 8 到 20 位之间
   - 必须包含大写字母（A-Z）
   - 必须包含小写字母（a-z）
   - 必须包含数字（0-9）

4. **邮箱格式校验**
   - 如果提供邮箱，验证邮箱格式
   - 必须包含@符号
   - @符号不能在开头或结尾

5. **手机号格式校验**
   - 如果提供手机号，验证为 11 位数字
   - 不允许包含字母或特殊字符

### 第二步：唯一性校验

1. **用户名唯一性校验**
   ```sql
   SELECT COUNT(*) FROM sys_user WHERE username = ?
   ```
   - 校验用户名是否已存在
   - 若已存在返回错误："该用户名已被使用请更换"

2. **手机号唯一性校验**
   ```sql
   SELECT COUNT(*) FROM sys_user WHERE phone = ? AND deleted_at IS NULL
   ```
   - 校验手机号是否已被注册
   - 若已注册返回错误："该手机号已被其他账户使用"

3. **邮箱唯一性校验**
   ```sql
   SELECT COUNT(*) FROM sys_user WHERE email = ? AND deleted_at IS NULL
   ```
   - 校验邮箱是否已被使用
   - 若已使用返回错误："该邮箱已被其他账户使用"

### 第三步：Licence 校验

1. **查询当前管理员数量**
   ```sql
   SELECT COUNT(*) FROM sys_user WHERE deleted_at IS NULL
   ```

2. **校验是否超过授权上限**
   - 系统预设 licence 上限（如 100）
   - 实际应从系统配置或 licence 文件读取
   - 若已达上限返回错误："管理员数量已达授权上限请联系商务续费"

### 第四步：角色校验

1. **查询角色信息**
   ```sql
   SELECT * FROM sys_role WHERE role_id IN (?, ?, ...)
   ```

2. **校验角色是否存在**
   - 验证查询到的角色数量与请求的角色数量一致
   - 若不一致返回错误："所选角色不存在或已被禁用"

3. **校验角色状态**
   - 检查每个角色的状态字段
   - 若角色状态为禁用（status=1）返回错误："所选角色不存在或已被禁用"

4. **权限校验（可选）**
   - 校验操作人是否有权限分配这些角色
   - 普通管理员不能分配比自己权限更高的角色

### 第五步：密码加密

1. **使用 bcrypt 加密算法**
   ```go
   hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
   ```

2. **加密参数**
   - 使用 DefaultCost（默认强度）
   - 可根据安全要求调整加密强度

3. **加密结果**
   - 生成 bcrypt 加密后的密码哈希值
   - 加密值存入数据库，不存储明文密码

### 第六步：数据入库

1. **构建用户对象**
   ```go
   user := models.SysUser{
       Username: req.Username,
       Password: string(hashedPassword),
       NickName: req.Nickname,
       Email:    req.Email,
       Phone:    req.Phone,
       Avatar:   req.Avatar,
       Status:   req.Status,
       Remark:   req.Remark,
   }
   user.CreateBy = req.CreateBy
   ```

2. **设置默认值**
   - 如果 status 为空，默认设置为"2"（正常）

3. **写入数据库**
   ```sql
   INSERT INTO sys_user (username, password, nick_name, email, phone, avatar, status, remark, create_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
   ```

### 第七步：角色关联

1. **建立多对多关系**
   ```go
   err := e.Orm.Model(&user).Association("SysRoles").Append(roles)
   ```

2. **写入关联表**
   ```sql
   INSERT INTO sys_user_role (user_id, role_id, created_at) VALUES (?, ?, ?)
   ```

3. **记录关联信息**
   - 记录关联创建时间
   - 记录关联操作人

### 第八步：初始化权限缓存（可选）

1. **加载角色权限**
   - 根据关联的角色加载权限列表
   - 构建权限缓存数据结构

2. **写入权限缓存**
   - 将权限数据写入缓存系统
   - 设置缓存过期时间

3. **缓存键设计**
   - 使用 user_id 作为缓存键
   - 便于后续查询和更新

### 第九步：日志记录（可选）

1. **记录操作日志**
   - 操作人：创建者的用户 ID
   - 操作时间：当前时间
   - 操作类型：创建管理员

2. **记录详细信息**
   - 被创建的管理员信息（ID、用户名、昵称）
   - 关联的角色列表
   - 操作 IP 地址

3. **审计日志**
   - 记录到审计日志表
   - 便于后续追溯和查询

### 第十步：返回结果

1. **构建返回数据**
   - 返回创建成功的管理员详情
   - 包含基本信息和关联角色信息

2. **返回字段**
   - success: 是否成功（布尔值）
   - admin_id: 管理员 ID
   - username: 用户名
   - nickname: 昵称姓名
   - role_list: 关联角色列表
   - created_at: 创建时间
   - message: 操作提示

## 异常处理

### 1. 用户名异常

**用户名已存在**
- 错误码：500
- 提示信息：该用户名已被使用请更换
- 触发条件：数据库中已存在相同用户名

**用户名格式错误**
- 错误码：500
- 提示信息：
  - "用户名长度必须在 6 到 20 位之间"
  - "用户名只能包含英文字母和数字"
- 触发条件：用户名长度不符合要求或包含非法字符

### 2. 密码异常

**密码强度不足**
- 错误码：500
- 提示信息：密码必须包含大小写字母和数字，长度 8 到 20 位
- 触发条件：
  - 密码长度小于 8 位或大于 20 位
  - 密码不包含大写字母
  - 密码不包含小写字母
  - 密码不包含数字

**密码为空**
- 错误码：500
- 提示信息：密码不能为空
- 触发条件：password 字段为空

### 3. 手机号异常

**手机号已注册**
- 错误码：500
- 提示信息：该手机号已被其他账户使用
- 触发条件：数据库中已存在相同手机号

**手机号格式错误**
- 错误码：500
- 提示信息：手机号格式不正确
- 触发条件：手机号不是 11 位数字

### 4. 邮箱异常

**邮箱已使用**
- 错误码：500
- 提示信息：该邮箱已被其他账户使用
- 触发条件：数据库中已存在相同邮箱

**邮箱格式错误**
- 错误码：500
- 提示信息：邮箱格式不正确
- 触发条件：邮箱格式不符合规范

### 5. 角色异常

**角色不存在**
- 错误码：500
- 提示信息：所选角色不存在或已被禁用
- 触发条件：
  - 请求的角色 ID 在数据库中不存在
  - 角色状态为禁用

**未选择角色**
- 错误码：500
- 提示信息：至少选择一个角色
- 触发条件：role_ids 为空数组

### 6. Licence 异常

**Licence 已达上限**
- 错误码：500
- 提示信息：管理员数量已达授权上限请联系商务续费
- 触发条件：当前管理员数量 >= licence 授权上限

### 7. 权限异常

**权限不足**
- 错误码：500
- 提示信息：您没有权限创建该管理员
- 触发条件：操作人没有管理员管理权限

### 8. 系统异常

**数据库错误**
- 错误码：500
- 提示信息：创建管理员失败
- 触发条件：数据库操作失败

**密码加密失败**
- 错误码：500
- 提示信息：密码加密失败
- 触发条件：bcrypt 加密算法执行失败

**角色关联失败**
- 错误码：500
- 提示信息：角色关联失败
- 触发条件：写入角色关联表失败

## 密码安全

### 1. 密码加密

- **加密算法**：bcrypt
- **加密强度**：DefaultCost（默认强度）
- **加密流程**：
  1. 接收明文密码
  2. 使用 bcrypt 生成哈希值
  3. 将哈希值存入数据库
  4. 不存储明文密码

### 2. 密码强度要求

- **长度要求**：8 到 20 位
- **字符要求**：
  - 必须包含大写字母（A-Z）
  - 必须包含小写字母（a-z）
  - 必须包含数字（0-9）
- **建议**：
  - 可考虑增加特殊字符要求
  - 可考虑增加密码复杂度评分

### 3. 密码策略（可选）

- **密码过期**：设置密码有效期，定期要求修改
- **密码历史**：记录历史密码，禁止使用最近使用过的密码
- **密码尝试限制**：限制连续登录失败次数
- **首次登录修改密码**：新创建的管理员首次登录强制修改密码

## 数据模型

### SysUser 模型

```go
type SysUser struct {
    UserId        int       `gorm:"primaryKey;autoIncrement"`
    Username      string    `gorm:"size:64"`
    Password      string    `gorm:"size:128"`  // bcrypt 加密值
    NickName      string    `gorm:"size:128"`
    Phone         string    `gorm:"size:20"`
    Email         string    `gorm:"size:128"`
    Avatar        string    `gorm:"size:255"`
    Status        string    `gorm:"size:4"`    // 1:禁用 2:正常
    Remark        string    `gorm:"size:255"`
    LoginIp       string    `gorm:"size:50"`
    LoginCount    int64     `gorm:"size:20"`
    LastLoginTime time.Time
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
    RoleKey   string    `gorm:"size:128"`  // 角色标识
    Status    string    `gorm:"size:1"`    // 1:禁用 2:正常
    Remark    string    `gorm:"size:255"`
    SysMenus  []SysMenu `gorm:"many2many:sys_role_menu"`
    models.ControlBy
    models.ModelTime
}
```

### SysUserRole 关联表

```go
type SysUserRole struct {
    UserId    int       `gorm:"primaryKey"`
    RoleId    int       `gorm:"primaryKey"`
    CreatedAt time.Time
}
```

## 请求参数 DTO

### SysAdminCreateReq

```go
type SysAdminCreateReq struct {
    Username string   `json:"username" binding:"required,min=6,max=20"`
    Password string   `json:"password" binding:"required,min=8,max=20"`
    Nickname string   `json:"nickname" binding:"required,max=50"`
    Email    string   `json:"email" binding:"omitempty,email,max=100"`
    Phone    string   `json:"phone" binding:"omitempty,len=11"`
    Avatar   string   `json:"avatar" binding:"omitempty,max=255"`
    RoleIds  []int    `json:"role_ids" binding:"required,min=1"`
    Status   string   `json:"status" binding:"omitempty,oneof=1 2"`
    Remark   string   `json:"remark" binding:"omitempty,max=255"`
    ControlBy
}
```

## 创建后操作

### 1. 完善个人资料

- 新管理员创建后可补充完善个人资料
- 可上传头像
- 可修改昵称
- 可修改联系方式

### 2. 通知管理员

- 通知被创建的管理员账户信息
- 发送账户信息邮件或短信
- 告知初始密码（如为系统生成）
- 提供登录地址和操作指南

### 3. 首次登录修改密码

- 新管理员首次登录强制修改初始密码
- 验证原密码
- 设置新密码（需符合密码强度要求）
- 更新密码修改标记

### 4. 密码过期策略

- 设置密码有效期（如 90 天）
- 密码即将过期时提醒修改
- 密码过期后强制修改
- 记录密码修改历史

## 最佳实践

### 1. 用户名规范

- 使用有意义的用户名
- 避免使用个人信息
- 遵循统一的命名规范
- 示例：
  - 运营管理员：operator001
  - 客服管理员：service001
  - 内容管理员：content001

### 2. 密码安全

- 使用强密码
- 定期更换密码
- 不与他人共享密码
- 不使用相同密码

### 3. 角色分配

- 遵循最小权限原则
- 根据职责分配角色
- 定期审查角色权限
- 及时回收不需要的权限

### 4. 信息管理

- 填写详细的备注信息
- 记录创建目的
- 记录使用场景
- 便于后续管理和审计

### 5. 审计追溯

- 记录所有创建操作
- 记录操作人和操作时间
- 记录关联的角色信息
- 便于问题追溯

## 示例代码

### cURL 示例

```bash
curl -X POST "http://localhost:8000/api/v1/admin/create" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "operator001",
    "password": "Operator123",
    "nickname": "运营管理员",
    "email": "operator001@example.com",
    "phone": "13812345678",
    "role_ids": [2, 3],
    "status": "2",
    "remark": "负责运营管理"
  }'
```

### JavaScript 示例

```javascript
fetch('http://localhost:8000/api/v1/admin/create', {
  method: 'POST',
  headers: {
    'Authorization': 'Bearer <token>',
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    username: 'operator001',
    password: 'Operator123',
    nickname: '运营管理员',
    email: 'operator001@example.com',
    phone: '13812345678',
    role_ids: [2, 3],
    status: '2',
    remark: '负责运营管理'
  })
})
.then(response => response.json())
.then(data => console.log(data))
.catch(error => console.error('Error:', error));
```

### Python 示例

```python
import requests

url = "http://localhost:8000/api/v1/admin/create"
headers = {
    "Authorization": "Bearer <token>",
    "Content-Type": "application/json"
}
data = {
    "username": "operator001",
    "password": "Operator123",
    "nickname": "运营管理员",
    "email": "operator001@example.com",
    "phone": "13812345678",
    "role_ids": [2, 3],
    "status": "2",
    "remark": "负责运营管理"
}

response = requests.post(url, json=data, headers=headers)
print(response.json())
```

## 注意事项

### 1. 超级管理员保护

- 超级管理员不可删除
- 超级管理员不可禁用
- 超级管理员权限不可修改
- 创建超级管理员需要特殊权限

### 2. 角色关联

- 每个管理员至少有一个角色
- 支持多个角色
- 角色变更即时生效
- 角色删除前检查关联

### 3. 密码安全

- 密码加密存储
- 不返回密码字段
- 不支持密码查询
- 密码重置需要特殊流程

### 4. 数据一致性

- 用户和角色关联使用事务
- 保证数据一致性
- 避免脏数据
- 失败时回滚

### 5. 性能优化

- 合理使用索引
- 避免全表扫描
- 批量操作优化
- 缓存权限数据

### 6. 敏感信息

- 手机号脱敏处理
- 密码不返回
- 保护用户隐私
- 符合数据安全规范

## 常见问题

### Q1: 用户名长度要求是多少？

A: 用户名长度必须在 6 到 20 位之间，只能包含英文字母和数字。

### Q2: 密码有什么要求？

A: 密码长度必须在 8 到 20 位之间，必须包含大写字母、小写字母和数字。

### Q3: 可以创建没有角色的管理员吗？

A: 不可以，每个管理员必须至少关联一个角色。

### Q4: Licence 上限是多少？

A: Licence 上限根据系统授权确定，默认为 100。如需增加请联系商务续费。

### Q5: 创建的管理员可以立即登录吗？

A: 可以，创建成功后管理员可以立即使用用户名和密码登录。

### Q6: 如何通知新管理员账户信息？

A: 可通过邮件、短信或其他方式通知新管理员用户名和初始密码。

### Q7: 支持批量创建管理员吗？

A: 当前版本不支持批量创建，可后续扩展此功能。

### Q8: 创建失败后如何排查问题？

A: 查看返回的错误信息，根据错误提示检查请求参数是否符合要求。
