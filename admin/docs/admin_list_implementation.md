# 管理员列表接口实现文档

## 接口概述

管理员列表接口用于查看系统中所有的后台管理员账户信息，包括账户基本信息、关联角色、登录状态、创建时间等。

## API 接口

### 1. 管理员列表接口

#### 请求方式

```
GET /api/v1/admin/list
```

#### 请求参数

| 参数名 | 类型 | 必填 | 说明 | 默认值 | 校验规则 |
|--------|------|------|------|--------|----------|
| keyword | string | 否 | 关键词搜索管理员姓名或用户名 | - | - |
| role_id | int | 否 | 关联角色 ID 筛选 | - | - |
| status | string | 否 | 管理员状态（1:禁用 2:正常） | - | - |
| last_login_from | string | 否 | 最后登录时间起始（2006-01-02） | - | - |
| last_login_to | string | 否 | 最后登录时间结束（2006-01-02） | - | - |
| page | int | 否 | 页码 | 1 | - |
| page_size | int | 否 | 每页数量 | 20 | - |
| sort_by | string | 否 | 排序字段 | created_at | - |
| sort_order | string | 否 | 排序方式 | desc | - |

#### 返回数据结构

```json
{
  "code": 200,
  "msg": "查询成功",
  "data": {
    "total": 100,
    "page": 1,
    "page_size": 20,
    "list": [
      {
        "admin_id": 1,
        "user_id": 1,
        "username": "admin",
        "nickname": "超级管理员",
        "email": "admin@example.com",
        "phone": "138****5678",
        "avatar": "https://example.com/avatar.jpg",
        "role_list": [
          {
            "role_id": 1,
            "role_name": "超级管理员",
            "role_code": "admin"
          }
        ],
        "status": "2",
        "status_text": "正常",
        "last_login_time": "2024-01-01 10:00:00",
        "last_login_ip": "192.168.1.100",
        "login_count": 100,
        "created_at": "2024-01-01 00:00:00",
        "created_by": "system",
        "updated_at": "2024-01-01 00:00:00",
        "is_super": true
      }
    ]
  }
}
```

#### 返回字段说明

| 字段名 | 类型 | 说明 |
|--------|------|------|
| total | int | 总条数 |
| page | int | 当前页码 |
| page_size | int | 每页条数 |
| list | array | 管理员列表 |
| admin_id | int | 管理员 ID |
| user_id | int | 用户 ID |
| username | string | 用户名 |
| nickname | string | 昵称姓名 |
| email | string | 邮箱 |
| phone | string | 手机号（脱敏） |
| avatar | string | 头像 URL |
| role_list | array | 关联角色列表 |
| role_list[].role_id | int | 角色 ID |
| role_list[].role_name | string | 角色名称 |
| role_list[].role_code | string | 角色编码 |
| status | string | 管理员状态（1:禁用 2:正常） |
| status_text | string | 状态文本 |
| last_login_time | string | 最后登录时间 |
| last_login_ip | string | 最后登录 IP |
| login_count | int | 登录次数 |
| created_at | string | 创建时间 |
| created_by | string | 创建人 |
| updated_at | string | 更新时间 |
| is_super | bool | 是否超级管理员 |

### 2. 管理员详情接口

#### 请求方式

```
GET /api/v1/admin/{id}
```

#### 请求参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id | int | 是 | 管理员 ID |

#### 返回数据结构

```json
{
  "code": 200,
  "msg": "查询成功",
  "data": {
    "admin_id": 1,
    "user_id": 1,
    "username": "admin",
    "nickname": "超级管理员",
    "email": "admin@example.com",
    "phone": "13812345678",
    "avatar": "https://example.com/avatar.jpg",
    "role_list": [
      {
        "role_id": 1,
        "role_name": "超级管理员",
        "role_code": "admin"
      }
    ],
    "role_ids": [1],
    "status": "2",
    "status_text": "正常",
    "last_login_time": "2024-01-01 10:00:00",
    "last_login_ip": "192.168.1.100",
    "login_count": 100,
    "remark": "备注信息",
    "created_at": "2024-01-01 00:00:00",
    "created_by": "system",
    "updated_at": "2024-01-01 00:00:00",
    "updated_by": "admin",
    "is_super": true
  }
}
```

### 3. 创建管理员接口

#### 请求方式

```
POST /api/v1/admin/create
```

#### 请求参数

| 参数名 | 类型 | 必填 | 说明 | 校验规则 |
|--------|------|------|------|----------|
| username | string | 是 | 用户名 | 必填，最大 50 字符 |
| password | string | 是 | 密码 | 必填，6-20 字符 |
| nickname | string | 是 | 昵称 | 必填，最大 50 字符 |
| email | string | 否 | 邮箱 | 邮箱格式，最大 100 字符 |
| phone | string | 否 | 手机号 | 最大 20 字符 |
| avatar | string | 否 | 头像 URL | 最大 255 字符 |
| role_ids | array | 是 | 角色 ID 列表 | 必填，至少一个角色 |
| status | string | 否 | 状态 | 1:禁用 2:正常，默认 2 |
| remark | string | 否 | 备注 | 最大 255 字符 |

#### 请求示例

```json
{
  "username": "test",
  "password": "123456",
  "nickname": "测试管理员",
  "email": "test@example.com",
  "phone": "13812345678",
  "role_ids": [2, 3],
  "status": "2",
  "remark": "测试管理员"
}
```

#### 返回数据结构

```json
{
  "code": 200,
  "msg": "创建成功",
  "data": {
    "admin_id": 10,
    "user_id": 10,
    "username": "test",
    "nickname": "测试管理员",
    "email": "test@example.com",
    "phone": "13812345678",
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
    "status": "2",
    "status_text": "正常",
    "created_at": "2024-01-01 12:00:00",
    "is_super": false
  }
}
```

### 4. 更新管理员接口

#### 请求方式

```
PUT /api/v1/admin/update
```

#### 请求参数

| 参数名 | 类型 | 必填 | 说明 | 校验规则 |
|--------|------|------|------|----------|
| user_id | int | 是 | 用户 ID | 必填 |
| username | string | 是 | 用户名 | 必填，最大 50 字符 |
| nickname | string | 是 | 昵称 | 必填，最大 50 字符 |
| email | string | 否 | 邮箱 | 邮箱格式，最大 100 字符 |
| phone | string | 否 | 手机号 | 最大 20 字符 |
| avatar | string | 否 | 头像 URL | 最大 255 字符 |
| role_ids | array | 是 | 角色 ID 列表 | 必填，至少一个角色 |
| status | string | 否 | 状态 | 1:禁用 2:正常 |
| remark | string | 否 | 备注 | 最大 255 字符 |

#### 请求示例

```json
{
  "user_id": 10,
  "username": "test_updated",
  "nickname": "更新后的测试管理员",
  "email": "test_updated@example.com",
  "role_ids": [2],
  "status": "2"
}
```

#### 返回数据结构

```json
{
  "code": 200,
  "msg": "更新成功",
  "data": {
    "admin_id": 10,
    "user_id": 10,
    "username": "test_updated",
    "nickname": "更新后的测试管理员",
    "email": "test_updated@example.com",
    "role_list": [
      {
        "role_id": 2,
        "role_name": "系统管理员",
        "role_code": "system_admin"
      }
    ],
    "status": "2",
    "status_text": "正常",
    "updated_at": "2024-01-01 13:00:00"
  }
}
```

### 5. 删除管理员接口

#### 请求方式

```
POST /api/v1/admin/delete
```

#### 请求参数

| 参数名 | 类型 | 必填 | 说明 | 校验规则 |
|--------|------|------|------|----------|
| user_id | int | 是 | 用户 ID | 必填 |

#### 请求示例

```json
{
  "user_id": 10
}
```

#### 返回数据结构

```json
{
  "code": 200,
  "msg": "删除成功",
  "data": {
    "user_id": 10,
    "username": "test_updated"
  }
}
```

### 6. 更新管理员状态接口

#### 请求方式

```
PUT /api/v1/admin/status
```

#### 请求参数

| 参数名 | 类型 | 必填 | 说明 | 校验规则 |
|--------|------|------|------|----------|
| user_id | int | 是 | 用户 ID | 必填 |
| status | string | 是 | 状态 | 必填，1:禁用 2:正常 |

#### 请求示例

```json
{
  "user_id": 10,
  "status": "1"
}
```

#### 返回数据结构

```json
{
  "code": 200,
  "msg": "更新成功",
  "data": {
    "admin_id": 10,
    "user_id": 10,
    "username": "test",
    "nickname": "测试管理员",
    "status": "1",
    "status_text": "禁用",
    "updated_at": "2024-01-01 14:00:00"
  }
}
```

## 处理流程

### 管理员列表查询流程

1. **参数解析**
   - 解析关键词、角色、状态、时间范围等筛选参数
   - 解析分页和排序参数

2. **权限校验**
   - 校验操作人是否有权限查看管理员列表
   - 普通管理员只能查看不能管理管理员

3. **数据权限过滤**
   - 超级管理员可查看所有管理员
   - 系统管理员可查看系统管理员和普通管理员
   - 普通管理员只能查看自己

4. **构建查询条件**
   - 关键词模糊匹配用户名或昵称
   - 角色 ID 精确匹配
   - 状态精确匹配
   - 时间范围区间查询

5. **执行分页查询**
   - 统计符合条件的总条数
   - 查询当前页数据按创建时间倒序

6. **关联角色信息**
   - 关联角色表获取每个管理员的角色名称和角色编码

7. **敏感信息处理**
   - 脱敏显示手机号中间四位
   - 隐藏密码字段不返回
   - 隐藏密码加密值

8. **格式化返回**
   - 转换枚举值为中文描述
   - 格式化时间字段
   - 统计登录信息

### 创建管理员流程

1. **参数校验**
   - 验证必填字段是否完整
   - 验证用户名格式
   - 验证密码强度（6-20 位）
   - 验证昵称格式

2. **用户名唯一性校验**
   - 校验用户名是否已存在
   - 若重复返回错误提示

3. **密码加密**
   - 使用 bcrypt 对密码进行加密
   - 加密强度为 DefaultCost

4. **创建用户**
   - 将用户信息写入数据库
   - 包括用户名、昵称、邮箱、手机号等
   - 默认状态为启用

5. **关联角色**
   - 将角色 ID 列表写入用户角色关联表
   - 建立多对多关系

6. **返回详情**
   - 返回创建成功的用户详情
   - 包含关联的角色信息

### 更新管理员流程

1. **参数校验**
   - 验证用户 ID 有效性
   - 验证必填字段
   - 验证至少选择一个角色

2. **查询用户**
   - 根据用户 ID 定位目标记录
   - 若不存在返回错误

3. **用户名唯一性校验**
   - 校验新用户名是否与其他用户重复（排除自己）
   - 若重复返回错误提示

4. **更新用户信息**
   - 更新用户表中的修改字段
   - 更新 update_by 和 updated_at

5. **更新角色关联**
   - 先删除原有关联
   - 再新增新的权限关联
   - 原子操作保证一致性

6. **返回详情**
   - 返回更新后的用户详情

### 删除管理员流程

1. **参数校验**
   - 验证用户 ID 有效性

2. **查询用户**
   - 根据用户 ID 定位目标记录

3. **超级管理员校验**
   - 检查是否为超级管理员
   - 超级管理员不可删除

4. **软删除**
   - 将用户记录标记为已删除状态
   - 更新 deleted_at 和 deleted_by

5. **返回结果**
   - 返回删除操作结果

### 更新管理员状态流程

1. **参数校验**
   - 验证用户 ID 有效性
   - 验证状态值有效性

2. **查询用户**
   - 根据用户 ID 定位目标记录

3. **更新状态**
   - 更新用户状态字段
   - 更新 update_by 和 updated_at

4. **返回详情**
   - 返回更新后的用户详情

## 管理员类型

1. **超级管理员**
   - 拥有所有权限
   - 不可删除
   - role_key = "admin"

2. **系统管理员**
   - 拥有系统管理权限
   - 可管理系统配置、角色等

3. **运营管理员**
   - 拥有运营相关权限
   - 可管理内容、用户等

4. **普通管理员**
   - 拥有受限的业务权限
   - 只能查看和管理授权范围内的数据

## 列表展示字段

管理员列表页面展示以下字段：

- **用户名**：登录账号
- **姓名**：昵称/真实姓名
- **手机号**：脱敏显示（中间四位隐藏）
- **角色**：关联的角色列表
- **状态**：正常/禁用
- **最后登录**：最后登录时间
- **登录次数**：累计登录次数
- **创建时间**：账户创建时间
- **操作列**：查看详情、编辑、禁用/启用、删除等功能按钮

## 排序规则

- **默认排序**：按创建时间倒序（最新创建在前）
- **支持排序字段**：
  - created_at：创建时间
  - last_login_time：最后登录时间
  - login_count：登录次数
  - username：用户名

## 状态说明

1. **正常状态（status=2）**
   - 可正常登录和使用后台
   - 权限正常生效

2. **禁用状态（status=1）**
   - 无法登录后台
   - 账户信息保留
   - 权限暂时失效

3. **删除状态**
   - 不显示在列表中
   - 采用软删除方式
   - 数据库记录保留

## 数据模型

### SysUser 模型

```go
type SysUser struct {
    UserId        int       `gorm:"primaryKey;autoIncrement"`
    Username      string    `gorm:"size:64"`
    Password      string    `gorm:"size:128"`
    NickName      string    `gorm:"size:128"`
    Phone         string    `gorm:"size:20"`
    RoleId        int       `gorm:"size:20"`
    Salt          string    `gorm:"size:255"`
    Avatar        string    `gorm:"size:255"`
    Sex           string    `gorm:"size:255"`
    Email         string    `gorm:"size:128"`
    DeptId        int       `gorm:"size:20"`
    PostId        int       `gorm:"size:20"`
    Remark        string    `gorm:"size:255"`
    Status        string    `gorm:"size:4"`
    LoginIp       string    `gorm:"size:50"`
    LoginCount    int64     `gorm:"size:20"`
    LastLoginTime time.Time `gorm:"comment:最后登录时间"`
    RoleKey       string    `gorm:"-"` // 角色标识
    SysRoles      []SysRole `gorm:"many2many:sys_user_role"`
    models.ControlBy
    models.ModelTime
}
```

### 管理员列表项

```go
type SysAdminListItem struct {
    AdminId       int                `json:"admin_id"`
    UserId        int                `json:"user_id"`
    Username      string             `json:"username"`
    Nickname      string             `json:"nickname"`
    Email         string             `json:"email"`
    Phone         string             `json:"phone"` // 脱敏
    PhoneRaw      string             `json:"-"`     // 原始
    Avatar        string             `json:"avatar"`
    RoleList      []SysAdminRoleItem `json:"role_list"`
    Status        string             `json:"status"`
    StatusText    string             `json:"status_text"`
    LastLoginTime string             `json:"last_login_time"`
    LastLoginIp   string             `json:"last_login_ip"`
    LoginCount    int                `json:"login_count"`
    CreatedAt     string             `json:"created_at"`
    CreatedBy     string             `json:"created_by"`
    UpdatedAt     string             `json:"updated_at"`
    IsSuper       bool               `json:"is_super"`
}
```

## 异常处理

### 常见错误

1. **管理员不存在**
   - 错误码：500
   - 提示信息：管理员不存在

2. **用户名已存在**
   - 错误码：500
   - 提示信息：用户名已存在

3. **密码格式错误**
   - 错误码：500
   - 提示信息：密码长度不能少于 6 位

4. **超级管理员不可删除**
   - 错误码：500
   - 提示信息：超级管理员不可删除

5. **至少选择一个角色**
   - 错误码：500
   - 提示信息：至少选择一个角色

6. **状态值无效**
   - 错误码：500
   - 提示信息：状态值无效

## 安全说明

1. **密码加密**
   - 使用 bcrypt 加密算法
   - 加密强度为 DefaultCost
   - 密码字段不返回给前端

2. **手机号脱敏**
   - 列表接口返回脱敏后的手机号
   - 详情接口返回完整手机号
   - 隐藏中间四位数字

3. **权限控制**
   - 超级管理员不可删除
   - 普通管理员只能查看自己
   - 操作需要相应的权限

4. **软删除**
   - 删除操作采用软删除
   - 数据库记录保留
   - 可通过筛选条件查看

## 最佳实践

1. **用户名规范**
   - 使用字母、数字、下划线组合
   - 长度控制在 6-20 位
   - 避免使用敏感词汇

2. **密码强度**
   - 至少 6 位字符
   - 建议包含字母和数字
   - 定期更换密码

3. **角色分配**
   - 根据职责分配角色
   - 遵循最小权限原则
   - 定期审查角色权限

4. **状态管理**
   - 离职人员及时禁用
   - 临时禁用而非删除
   - 定期清理无用账户

5. **登录监控**
   - 记录登录时间和 IP
   - 统计登录次数
   - 异常登录告警

6. **审计日志**
   - 记录创建人和创建时间
   - 记录更新人和更新时间
   - 记录删除操作

## 示例代码

### 查询管理员列表

```bash
curl -X GET "http://localhost:8000/api/v1/admin/list?page=1&page_size=20" \
  -H "Authorization: Bearer <token>"
```

### 按角色筛选

```bash
curl -X GET "http://localhost:8000/api/v1/admin/list?role_id=2" \
  -H "Authorization: Bearer <token>"
```

### 按状态筛选

```bash
curl -X GET "http://localhost:8000/api/v1/admin/list?status=2" \
  -H "Authorization: Bearer <token>"
```

### 关键词搜索

```bash
curl -X GET "http://localhost:8000/api/v1/admin/list?keyword=admin" \
  -H "Authorization: Bearer <token>"
```

### 创建管理员

```bash
curl -X POST "http://localhost:8000/api/v1/admin/create" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "newadmin",
    "password": "123456",
    "nickname": "新管理员",
    "email": "newadmin@example.com",
    "role_ids": [2, 3]
  }'
```

### 更新管理员

```bash
curl -X PUT "http://localhost:8000/api/v1/admin/update" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 10,
    "nickname": "更新后的昵称",
    "role_ids": [2]
  }'
```

### 禁用管理员

```bash
curl -X PUT "http://localhost:8000/api/v1/admin/status" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 10,
    "status": "1"
  }'
```

### 删除管理员

```bash
curl -X POST "http://localhost:8000/api/v1/admin/delete" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 10
  }'
```

## 注意事项

1. **超级管理员保护**
   - 超级管理员不可删除
   - 超级管理员不可禁用
   - 超级管理员权限不可修改

2. **角色关联**
   - 每个管理员至少有一个角色
   - 支持多个角色
   - 角色变更即时生效

3. **密码安全**
   - 密码加密存储
   - 不返回密码字段
   - 不支持密码查询

4. **数据一致性**
   - 用户和角色关联使用事务
   - 保证数据一致性
   - 避免脏数据

5. **性能优化**
   - 列表接口使用分页
   - 避免全表扫描
   - 合理使用索引

6. **敏感信息**
   - 手机号脱敏处理
   - 密码不返回
   - 保护用户隐私
