# 平台用户详情 API 文档

## 接口说明

实现管理后台用户详情查询功能，支持多维度关联查询，包括用户基础信息、会员信息、设备信息、实名认证信息、会话信息等。

## 接口地址

```
GET /api/v1/platform-user/{userId}
```

## 请求参数

### Header

```
Authorization: Bearer <admin_access_token>
```

### 路径参数

| 参数名 | 类型 | 必填 | 说明 | 示例 |
|--------|------|------|------|------|
| userId | int | 是 | 用户 ID | 1001 |

### 请求示例

```bash
curl -X GET 'http://localhost:8000/api/v1/platform-user/1001' \
  -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...'
```

## 成功响应

### 响应数据结构

```json
{
  "code": 200,
  "msg": "查询成功",
  "data": {
    "userId": 1001,
    "mobile": "138 1234 5678",
    "email": "zhangsan@example.com",
    "nickname": "张三",
    "avatar": "https://example.com/avatar/1001.jpg",
    "status": 1,
    "realNameStatus": 2,
    "registerTime": 1672531200,
    "lastLoginTime": 1704067200,
    "lastLoginIP": "192.168.1.100",
    "createdAt": "2023-01-01T00:00:00Z",
    "updatedAt": "2024-01-01T00:00:00Z",
    
    "memberLevel": 2,
    "memberLevelName": "SVIP 会员",
    "memberExpireAt": 1735689600,
    "memberStatus": 0,
    "memberCreatedAt": 1672531200,
    
    "bindDeviceCount": 3,
    "onlineDeviceCount": 2,
    "deviceList": [
      {
        "deviceSn": "SN12345678",
        "deviceName": "智能音箱 X1",
        "model": "X1",
        "online": true,
        "bindTime": 1704067200
      }
    ],
    
    "realNameInfo": {
      "status": 2,
      "realName": "张*",
      "idCard": "110101******1234",
      "submitTime": 1672531200,
      "auditTime": 1672617600,
      "auditor": "admin",
      "auditRemark": "审核通过"
    },
    
    "activeSessionCount": 2,
    "recentLogins": [
      {
        "loginTime": 1704067200,
        "device": "Chrome",
        "ip": "192.168.1.100",
        "location": "北京市"
      }
    ]
  }
}
```

### 字段说明

#### 基础信息
| 字段名 | 类型 | 说明 |
|--------|------|------|
| userId | int64 | 用户 ID |
| mobile | string | 手机号（脱敏） |
| email | string | 邮箱 |
| nickname | string | 昵称 |
| avatar | string | 头像 URL |
| status | int32 | 账号状态 0-禁用 1-正常 |
| realNameStatus | int32 | 实名状态 0-未提交 1-审核中 2-已通过 3-已驳回 |
| registerTime | int64 | 注册时间戳 |
| lastLoginTime | int64 | 最后登录时间戳 |
| lastLoginIP | string | 最后登录 IP |
| createdAt | time.Time | 创建时间 |
| updatedAt | time.Time | 更新时间 |

#### 会员信息
| 字段名 | 类型 | 说明 |
|--------|------|------|
| memberLevel | int32 | 会员等级 |
| memberLevelName | string | 会员等级名称 |
| memberExpireAt | int64 | 会员过期时间戳 |
| memberStatus | int32 | 会员状态 0-正常 1-过期 2-冻结 |
| memberCreatedAt | int64 | 会员开通时间戳 |

#### 设备信息
| 字段名 | 类型 | 说明 |
|--------|------|------|
| bindDeviceCount | int64 | 绑定设备总数 |
| onlineDeviceCount | int64 | 在线设备数 |
| deviceList | array | 设备列表（最近 10 个） |

##### DeviceList 数组项
| 字段名 | 类型 | 说明 |
|--------|------|------|
| deviceSn | string | 设备 SN |
| deviceName | string | 设备名称 |
| model | string | 设备型号 |
| online | bool | 是否在线 |
| bindTime | int64 | 绑定时间戳 |

#### 实名信息
| 字段名 | 类型 | 说明 |
|--------|------|------|
| realNameInfo | object | 实名认证信息 |
| └─ status | int32 | 实名状态 |
| └─ realName | string | 实名姓名（脱敏） |
| └─ idCard | string | 身份证号（脱敏） |
| └─ submitTime | int64 | 提交时间戳 |
| └─ auditTime | int64 | 审核时间戳 |
| └─ auditor | string | 审核人 |
| └─ auditRemark | string | 审核意见 |

#### 会话信息
| 字段名 | 类型 | 说明 |
|--------|------|------|
| activeSessionCount | int64 | 当前有效会话数 |
| recentLogins | array | 最近登录记录（最近 5 条） |

##### RecentLogins 数组项
| 字段名 | 类型 | 说明 |
|--------|------|------|
| loginTime | int64 | 登录时间戳 |
| device | string | 登录设备 |
| ip | string | 登录 IP |
| location | string | 登录地点 |

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

## 业务逻辑

### 1. 多维度关联查询

- **用户基础信息**：从 users 表查询核心字段
- **会员信息**：关联 user_member 表，查询会员等级、有效期、状态
- **设备信息**：关联 user_device_bind 表和 device 表，统计绑定设备数、在线设备数，查询设备列表
- **实名认证信息**：关联 user_real_name 表，查询实名状态、姓名、身份证号（脱敏）
- **会话信息**：关联 user_session 表和 sys_login_log 表，查询有效会话数、最近登录记录

### 2. 数据脱敏处理

- **手机号**：138 1234 5678（空格分隔）
- **姓名**：张*（只显示第一个字）
- **身份证号**：110101******1234（前 6 位 + 6 个星号 + 后 4 位）

### 3. 日志记录

- 记录管理员查询操作
- 记录查询的用户 ID
- 记录返回的设备数量、登录记录数量

## 异常处理

| 异常类型 | 处理方式 |
|----------|----------|
| 未登录/Token 失效 | 返回 401 未授权 |
| 无权限 | 返回 403 权限不足 |
| 用户不存在 | 返回 404 用户不存在 |
| 参数错误 | 返回 400 参数异常 |
| 数据库错误 | 返回 500 服务器错误 |

## 约束规则

1. **权限隔离**：仅管理员可调用，普通用户禁止访问
2. **敏感信息保护**：禁止返回密码、密钥、完整手机号、完整身份证号等敏感数据
3. **操作审计**：所有查询操作全程留痕，支持溯源
4. **性能优化**：关联查询使用 JOIN，限制返回数量（设备列表最多 10 条，登录记录最多 5 条）

## 管理后台扩展操作

在用户详情页，管理员可以执行以下操作（需配合其他接口）：

- 禁用/启用用户账号
- 重置用户密码
- 修改用户会员等级/有效期
- 强制解绑用户设备
- 审核用户实名认证
- 踢下线用户登录会话
