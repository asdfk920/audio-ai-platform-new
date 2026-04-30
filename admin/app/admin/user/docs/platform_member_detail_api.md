# 平台会员详情 API 文档

## 接口说明

实现管理后台会员详情查询功能，支持多维度关联查询，包括用户基础信息、会员核心信息、会员权益、订单记录、管理员操作记录等全量信息。

## 接口地址

```
GET /api/v1/platform-member/{userId}
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
curl -X GET 'http://localhost:8000/api/v1/platform-member/1001' \
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
    "mobile": "138****5678",
    "nickname": "张三",
    "avatar": "https://example.com/avatar/1001.jpg",
    "userStatus": 1,
    "memberLevel": 2,
    "memberLevelName": "SVIP 会员",
    "memberStatus": 0,
    "expireTime": 1735689600,
    "remainingDays": 365,
    "startTime": 1672531200,
    "lastRenewalTime": 1704067200,
    "isAutoRenew": false,
    "isExpiringSoon": false,
    "deviceBindLimit": 10,
    "currentBindCount": 3,
    "availableBenefits": [
      {
        "benefitName": "无损音质",
        "benefitCode": "lossless_audio",
        "status": 0,
        "totalCount": 0,
        "usedCount": 0,
        "remainingCount": 0,
        "expireTime": 0
      },
      {
        "benefitName": "无限播放",
        "benefitCode": "unlimited_play",
        "status": 0,
        "totalCount": 0,
        "usedCount": 0,
        "remainingCount": 0,
        "expireTime": 0
      },
      {
        "benefitName": "绑定 10 台设备",
        "benefitCode": "bind_10_devices",
        "status": 0,
        "totalCount": 10,
        "usedCount": 3,
        "remainingCount": 7,
        "expireTime": 1735689600
      }
    ],
    "usedBenefits": [],
    "orderRecords": [
      {
        "orderId": "ORDER_20240101_001",
        "payAmount": 29900,
        "payType": 1,
        "orderTime": 1704067200,
        "orderStatus": 1,
        "orderType": 2,
        "memberLevel": 2,
        "memberDays": 365
      }
    ],
    "operateRecords": [
      {
        "operateAdmin": 1,
        "operateAdminName": "超级管理员",
        "operateType": 1,
        "oldLevel": 1,
        "newLevel": 2,
        "oldExpireTime": 1672531200,
        "newExpireTime": 1735689600,
        "operateTime": 1704067200,
        "remark": "VIP 升级 SVIP"
      }
    ],
    "bindDeviceCount": 3,
    "levelConfig": {
      "level": 2,
      "name": "SVIP 会员",
      "color": "#E6A23C",
      "description": "至尊会员服务",
      "deviceBindLimit": 10
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

### 字段说明

#### 1. 用户基础信息

| 字段名 | 类型 | 说明 |
|--------|------|------|
| userId | int64 | 用户 ID |
| mobile | string | 手机号（脱敏：138****5678） |
| nickname | string | 用户昵称 |
| avatar | string | 头像 URL |
| userStatus | int32 | 用户账号状态 0-禁用 1-正常 |

#### 2. 会员核心信息

| 字段名 | 类型 | 说明 |
|--------|------|------|
| memberLevel | int32 | 会员等级 0-普通 1-VIP 2-SVIP 3-终身 |
| memberLevelName | string | 会员等级名称 |
| memberStatus | int32 | 会员状态 0-正常 1-过期 2-冻结 3-未开通 |
| expireTime | int64 | 会员过期时间戳 |
| remainingDays | int64 | 剩余天数 |
| startTime | int64 | 开通会员时间戳 |
| lastRenewalTime | int64 | 最后续费时间戳 |
| isAutoRenew | bool | 是否自动续费 |
| isExpiringSoon | bool | 是否即将到期（7 天内） |

#### 3. 会员权益信息

| 字段名 | 类型 | 说明 |
|--------|------|------|
| deviceBindLimit | int64 | 绑定设备上限 |
| currentBindCount | int64 | 当前绑定数量 |
| availableBenefits | array | 可用权益列表 |
| usedBenefits | array | 已用权益列表 |

**权益信息字段**

| 字段名 | 类型 | 说明 |
|--------|------|------|
| benefitName | string | 权益名称 |
| benefitCode | string | 权益编码 |
| status | int32 | 状态 0-可用 1-已用完 2-已过期 |
| totalCount | int64 | 总次数 |
| usedCount | int64 | 已用次数 |
| remainingCount | int64 | 剩余次数 |
| expireTime | int64 | 过期时间戳 |

#### 4. 购买/订单记录

| 字段名 | 类型 | 说明 |
|--------|------|------|
| orderRecords | array | 订单记录（最近 10 条） |

**订单记录字段**

| 字段名 | 类型 | 说明 |
|--------|------|------|
| orderId | string | 订单 ID |
| payAmount | int64 | 支付金额（分） |
| payType | int32 | 支付方式 1-微信 2-支付宝 3-银行卡 |
| orderTime | int64 | 订单时间戳 |
| orderStatus | int32 | 订单状态 0-待支付 1-已支付 2-已取消 3-已退款 |
| orderType | int32 | 订单类型 1-新购 2-续费 3-升级 |
| memberLevel | int32 | 会员等级 |
| memberDays | int64 | 会员天数 |

#### 5. 管理员操作记录

| 字段名 | 类型 | 说明 |
|--------|------|------|
| operateRecords | array | 操作记录（最近 20 条） |

**操作记录字段**

| 字段名 | 类型 | 说明 |
|--------|------|------|
| operateAdmin | int64 | 操作管理员 ID |
| operateAdminName | string | 操作管理员姓名 |
| operateType | int32 | 操作类型 1-修改等级 2-冻结 3-解冻 4-修改过期时间 |
| oldLevel | int32 | 原等级 |
| newLevel | int32 | 新等级 |
| oldExpireTime | int64 | 原过期时间 |
| newExpireTime | int64 | 新过期时间 |
| operateTime | int64 | 操作时间戳 |
| remark | string | 备注 |

#### 6. 其他统计

| 字段名 | 类型 | 说明 |
|--------|------|------|
| bindDeviceCount | int64 | 绑定设备数量 |
| levelConfig | object | 会员等级配置 |
| └─ level | int32 | 等级 |
| └─ name | string | 等级名称 |
| └─ color | string | 颜色标签 |
| └─ description | string | 描述 |
| └─ deviceBindLimit | int64 | 绑定设备上限 |
| renewalRecords | array | 续费记录（最近 10 条） |
| └─ renewalTime | int64 | 续费时间戳 |
| └─ days | int64 | 续费天数 |
| └─ amount | int64 | 续费金额（分） |

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

### 1. 多表关联查询

- **用户基础信息**：从 `users` 表查询用户 ID、昵称、脱敏手机号、账号状态
- **会员主信息**：从 `user_member` 表查询等级、状态、开通时间、过期时间、剩余天数、是否自动续费
- **会员等级配置**：从 `member_level_config` 表查询等级名称、权益列表、绑定设备上限、描述
- **订单/支付记录**：从 `user_member_order` 表查询购买记录、支付方式、金额、订单时间
- **会员权益使用情况**：从 `user_benefit` 表查询已用次数、剩余次数、权益生效状态
- **操作日志**：从 `member_operate_log` 表查询管理员历史修改记录

### 2. 数据格式化与脱敏

- **手机号脱敏**：`138****5678`
- **状态转为中文标签**：正常/已过期/已冻结/未开通
- **计算剩余天数**：`(过期时间 - 当前时间) / 86400`
- **判断即将到期**：剩余天数 <= 7 天

### 3. 会员权益分类

- **可用权益**：状态为 0（可用）的权益
- **已用权益**：状态为 1（已用完）或 2（已过期）的权益

### 4. 会员等级权益配置

| 等级 | 名称 | 绑定设备上限 | 主要权益 |
|------|------|-------------|---------|
| 0 | 普通会员 | 1 | 基础音质、每日 10 次播放 |
| 1 | VIP 会员 | 3 | 高品质音质、每日 100 次播放、绑定 3 台设备 |
| 2 | SVIP 会员 | 10 | 无损音质、无限播放、绑定 10 台设备、离线下载 |
| 3 | 终身会员 | 20 | 无损音质、无限播放、绑定 20 台设备、离线下载、专属客服 |

## 异常处理

| 异常类型 | 处理方式 |
|----------|----------|
| 管理员未登录 | 返回 401 未授权 |
| 无权限 | 返回 403 权限不足 |
| 用户不存在 | 返回 404 用户不存在 |
| 用户未开通会员 | 返回未开通状态（memberStatus=3），不报错 |
| 禁止返回敏感数据 | 不返回密码、密钥等敏感数据 |

## 扩展操作

在会员详情页，管理员可以直接执行以下操作（需配合其他接口）：

### 1. 修改会员等级/时长

- 修改用户会员等级
- 修改会员过期时间
- 批量调整会员有效期

### 2. 冻结/解冻会员

- 冻结会员（设置 memberStatus=2）
- 解冻会员（设置 memberStatus=0）

### 3. 查看关联信息

- 查看用户详情
- 查看订单记录
- 查看续费记录
- 查看权益使用记录
- 查看绑定设备列表

### 4. 操作审计

- 查看管理员操作历史
- 追溯会员变更原因

## 前端交互要点

### 1. 状态展示

- **会员等级标签**：带颜色（VIP-蓝色、SVIP-金色、终身会员 - 红色）
- **会员状态标签**：正常（绿色）、已过期（灰色）、已冻结（红色）、未开通（灰色）
- **即将到期提示**：剩余天数 <= 7 天显示警告色（橙色）

### 2. 权益展示

- **可用权益**：绿色标签，显示剩余次数
- **已用权益**：灰色标签，显示已用完/已过期
- **权益进度条**：可视化展示使用进度

### 3. 操作按钮

- **修改会员信息**：弹窗修改等级、过期时间
- **冻结/解冻**：根据当前状态显示对应操作
- **查看用户详情**：跳转到用户详情页
- **查看订单记录**：弹窗显示完整订单列表
- **导出会员数据**：导出会员详细信息

## 数据表说明

### 主要数据表

- `users`：用户基础信息表
- `user_member`：会员信息表
- `member_level_config`：会员等级配置表
- `user_member_order`：会员订单表
- `user_benefit`：用户权益使用表
- `member_operate_log`：会员操作日志表
- `member_renewal_record`：会员续费记录表
- `user_device_bind`：用户设备绑定表

### 字段映射

| 响应字段 | 数据表 | 字段名 |
|----------|--------|--------|
| userId | users | user_id |
| mobile | users | mobile |
| nickname | users | nickname |
| userStatus | users | status |
| memberLevel | users | member_level |
| memberStatus | user_member | status |
| expireTime | user_member | expire_at |
| startTime | user_member | created_at |
| lastRenewalTime | user_member | updated_at |
| isAutoRenew | user_member | auto_renew |
