# 平台会员列表 API 文档

## 接口说明

实现管理后台会员列表查询功能，支持分页、多条件筛选、会员信息查询、设备绑定统计。

## 接口列表

### 1. 会员列表查询

#### 接口地址

```
GET /api/v1/platform-member/list
```

#### 请求参数

##### Header

```
Authorization: Bearer <admin_access_token>
```

##### Query 参数

| 参数名 | 类型 | 必填 | 说明 | 示例 |
|--------|------|------|------|------|
| page | int | 否 | 页码，默认 1 | 1 |
| pageSize | int | 否 | 每页条数，默认 10，最大 100 | 10 |
| mobile | string | 否 | 手机号（模糊搜索） | 13800138000 |
| nickname | string | 否 | 昵称（模糊搜索） | 张三 |
| memberLevel | int | 否 | 会员等级 | 2 |
| memberStatus | int | 否 | 会员状态 0-正常 1-过期 2-冻结 | 0 |
| startTimeStart | string | 否 | 开通会员时间开始（Unix 时间戳） | 1672531200 |
| startTimeEnd | string | 否 | 开通会员时间结束（Unix 时间戳） | 1704067199 |
| expireTimeStart | string | 否 | 过期时间开始（Unix 时间戳） | 1704067200 |
| expireTimeEnd | string | 否 | 过期时间结束（Unix 时间戳） | 1735689600 |

#### 请求示例

```bash
curl -X GET 'http://localhost:8000/api/v1/platform-member/list?page=1&pageSize=10&memberLevel=2&memberStatus=0' \
  -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...'
```

#### 成功响应

```json
{
  "code": 200,
  "msg": "查询成功",
  "data": {
    "list": [
      {
        "userId": 1001,
        "mobile": "138****5678",
        "nickname": "张三",
        "avatar": "https://example.com/avatar/1001.jpg",
        "memberLevel": 2,
        "memberLevelName": "SVIP 会员",
        "memberStatus": 0,
        "expireTime": 1735689600,
        "remainingDays": 365,
        "startTime": 1672531200,
        "bindDeviceCount": 3,
        "createTime": 1672531200,
        "updateTime": 1704067200
      }
    ],
    "total": 100,
    "page": 1,
    "pageSize": 10
  }
}
```

#### 字段说明

| 字段名 | 类型 | 说明 |
|--------|------|------|
| userId | int64 | 用户 ID |
| mobile | string | 手机号（脱敏：138****5678） |
| nickname | string | 用户昵称 |
| avatar | string | 头像 URL |
| memberLevel | int32 | 会员等级 |
| memberLevelName | string | 会员等级名称 |
| memberStatus | int32 | 会员状态 0-正常 1-过期 2-冻结 |
| expireTime | int64 | 会员过期时间戳 |
| remainingDays | int64 | 剩余天数 |
| startTime | int64 | 开通会员时间戳 |
| bindDeviceCount | int64 | 绑定设备数量 |
| createTime | int64 | 开通会员时间戳 |
| updateTime | int64 | 最后更新时间戳 |
| total | int64 | 总条数 |
| page | int | 当前页码 |
| pageSize | int | 每页条数 |

---

### 2. 会员详情查询

#### 接口地址

```
GET /api/v1/platform-member/{userId}
```

#### 请求参数

##### 路径参数

| 参数名 | 类型 | 必填 | 说明 | 示例 |
|--------|------|------|------|------|
| userId | int | 是 | 用户 ID | 1001 |

#### 请求示例

```bash
curl -X GET 'http://localhost:8000/api/v1/platform-member/1001' \
  -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...'
```

#### 成功响应

```json
{
  "code": 200,
  "msg": "查询成功",
  "data": {
    "userId": 1001,
    "mobile": "138****5678",
    "nickname": "张三",
    "avatar": "https://example.com/avatar/1001.jpg",
    "memberLevel": 2,
    "memberLevelName": "SVIP 会员",
    "memberStatus": 0,
    "expireTime": 1735689600,
    "remainingDays": 365,
    "startTime": 1672531200,
    "lastRenewalTime": 1704067200,
    "totalRenewal": 5,
    "bindDeviceCount": 3,
    "levelConfig": {
      "level": 2,
      "name": "SVIP 会员",
      "color": "#E6A23C",
      "description": "至尊会员服务"
    },
    "renewalRecords": [
      {
        "renewalTime": 1704067200,
        "days": 365,
        "amount": 29900
      }
    ]
  }
}
```

#### 字段说明

| 字段名 | 类型 | 说明 |
|--------|------|------|
| userId | int64 | 用户 ID |
| mobile | string | 手机号（脱敏） |
| nickname | string | 用户昵称 |
| avatar | string | 头像 URL |
| memberLevel | int32 | 会员等级 |
| memberLevelName | string | 会员等级名称 |
| memberStatus | int32 | 会员状态 |
| expireTime | int64 | 会员过期时间戳 |
| remainingDays | int64 | 剩余天数 |
| startTime | int64 | 开通会员时间戳 |
| lastRenewalTime | int64 | 最后续费时间戳 |
| totalRenewal | int64 | 累计续费次数 |
| bindDeviceCount | int64 | 绑定设备数量 |
| levelConfig | object | 会员等级配置 |
| └─ level | int32 | 等级 |
| └─ name | string | 等级名称 |
| └─ color | string | 颜色标签 |
| └─ description | string | 描述 |
| renewalRecords | array | 续费记录（最近 10 条） |
| └─ renewalTime | int64 | 续费时间戳 |
| └─ days | int64 | 续费天数 |
| └─ amount | int64 | 续费金额（分） |

---

### 3. 更新会员信息

#### 接口地址

```
PUT /api/v1/platform-member
```

#### 请求参数

##### Body 参数

```json
{
  "userId": 1001,
  "memberLevel": 2,
  "expireTime": 1735689600
}
```

| 参数名 | 类型 | 必填 | 说明 | 示例 |
|--------|------|------|------|------|
| userId | int64 | 是 | 用户 ID | 1001 |
| memberLevel | int32 | 是 | 会员等级（0-普通会员 1-VIP 2-SVIP 3-终身会员） | 2 |
| expireTime | int64 | 是 | 会员过期时间戳 | 1735689600 |

#### 请求示例

```bash
curl -X PUT 'http://localhost:8000/api/v1/platform-member' \
  -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...' \
  -H 'Content-Type: application/json' \
  -d '{
    "userId": 1001,
    "memberLevel": 2,
    "expireTime": 1735689600
  }'
```

#### 成功响应

```json
{
  "code": 200,
  "msg": "更新成功",
  "data": null
}
```

---

## 错误响应

### 400 参数错误

```json
{
  "code": 400,
  "msg": "参数异常",
  "data": null
}
```

### 401 未授权

```json
{
  "code": 401,
  "msg": "未授权",
  "data": null
}
```

### 403 权限不足

```json
{
  "code": 403,
  "msg": "权限不足",
  "data": null
}
```

### 404 用户不存在

```json
{
  "code": 404,
  "msg": "用户不存在",
  "data": null
}
```

### 500 服务器错误

```json
{
  "code": 500,
  "msg": "查询失败",
  "data": null
}
```

---

## 业务逻辑

### 1. 多表关联查询

- **主表**：从 `users` 表查询用户基础信息（user_id、mobile、nickname、avatar）
- **关联表**：从 `user_member` 表查询会员核心信息（会员等级、过期时间、会员状态、开通时间）
- **统计**：关联 `user_device_bind` 表，统计用户绑定设备数量

### 2. 动态条件构建

- **会员等级筛选**：精准匹配等级 ID
- **会员状态筛选**：正常 / 过期 / 冻结
- **时间范围筛选**：注册时间、开通会员时间、过期时间
- **关键词搜索**：手机号、昵称模糊匹配

### 3. 数据格式化

- **手机号脱敏**：`138****5678`
- **会员状态转换**：数字转换为文字标签（正常 / 过期 / 冻结）
- **剩余天数计算**：`（过期时间 - 当前时间）/ 86400`
- **时间戳转换**：Unix 时间戳转换为可读格式

### 4. 会员等级配置

| 等级 | 名称 | 颜色标签 | 描述 |
|------|------|----------|------|
| 0 | 普通会员 | #909399 | 基础会员服务 |
| 1 | VIP 会员 | #409EFF | 高级会员服务 |
| 2 | SVIP 会员 | #E6A23C | 至尊会员服务 |
| 3 | 终身会员 | #F56C6C | 终身尊享会员 |

---

## 异常处理

| 异常类型 | 处理方式 |
|----------|----------|
| 未登录/Token 过期 | 返回 401 未授权 |
| 无权限 | 返回 403 权限不足 |
| 参数错误 | 返回 400 请求参数异常 |
| 用户不存在 | 返回 404 用户不存在 |
| 无数据 | 返回空列表，不报错 |

---

## 约束规则

1. **权限隔离**：仅管理员可调用，普通用户禁止访问
2. **敏感信息保护**：禁止返回密码、密钥、完整手机号等敏感数据
3. **操作审计**：所有查询操作全程留痕，支持溯源
4. **性能优化**：
   - 必须分页，禁止全表查询
   - 多表关联使用 JOIN
   - 高频查询可加 Redis 缓存（过期时间 5-10 分钟）

---

## 前端交互要点

### 1. 列表功能

- 支持点击列表头进行排序（按开通时间、过期时间、剩余天数）
- 支持批量操作导出数据
- 支持点击用户 ID 跳转到用户详情页
- 支持点击会员等级跳转到会员等级配置页

### 2. 筛选功能

- 手机号模糊搜索
- 昵称模糊搜索
- 会员等级下拉选择
- 会员状态下拉选择
- 时间范围选择器（开通时间、过期时间）

### 3. 状态展示

- 会员等级标签（带颜色）
- 会员状态标签（正常/过期/冻结）
- 剩余天数高亮显示（小于 7 天显示警告色）

---

## 管理后台扩展操作

在会员列表页，管理员可以执行以下操作（需配合其他接口）：

- 修改用户会员等级
- 修改会员过期时间
- 批量调整会员有效期
- 导出会员数据
- 查看会员续费记录
- 查看会员权益使用记录
