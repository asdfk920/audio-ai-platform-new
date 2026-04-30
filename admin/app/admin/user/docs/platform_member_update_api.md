# 平台会员更新 API 文档

## 接口说明

实现管理后台会员更新功能，支持管理员给用户开通、修改会员，包括开通、续费、升级、降级、延长有效期等多种操作。

## 接口地址

```
PUT /api/v1/platform-member
```

## 请求参数

### Header

```
Authorization: Bearer <admin_access_token>
Content-Type: application/json
```

### 请求 Body

```json
{
  "userId": 1001,
  "memberLevel": 2,
  "expireTime": 1735689600,
  "days": 365,
  "remark": "VIP 升级 SVIP，用户要求提升音质体验",
  "operationType": 3
}
```

### 参数说明

| 参数名 | 类型 | 必填 | 说明 | 示例 |
|--------|------|------|------|------|
| userId | int64 | 是 | 用户 ID | 1001 |
| memberLevel | int32 | 是 | 会员等级（0-普通 1-VIP 2-SVIP 3-终身） | 2 |
| expireTime | int64 | 是 | 会员过期时间戳（Unix 时间戳） | 1735689600 |
| days | int64 | 否 | 有效期天数（与 expireTime 二选一，续费时使用） | 365 |
| remark | string | 是 | 操作备注（必填，用于审计） | VIP 升级 SVIP |
| operationType | int32 | 否 | 操作类型（1-开通 2-续费 3-升级 4-降级 5-延长，不填自动判断） | 3 |

### 请求示例

```bash
curl -X PUT 'http://localhost:8000/api/v1/platform-member' \
  -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...' \
  -H 'Content-Type: application/json' \
  -d '{
    "userId": 1001,
    "memberLevel": 2,
    "expireTime": 1735689600,
    "remark": "VIP 升级 SVIP，用户要求提升音质体验"
  }'
```

## 成功响应

### 响应数据结构

```json
{
  "code": 200,
  "msg": "更新成功",
  "data": {
    "userId": 1001,
    "memberLevel": 2,
    "memberLevelName": "SVIP 会员",
    "memberStatus": 0,
    "expireTime": 1735689600,
    "remainingDays": 365,
    "operationType": 3,
    "oldLevel": 1,
    "oldExpireTime": 1672531200,
    "remark": "VIP 升级 SVIP，用户要求提升音质体验",
    "updateTime": 1704067200
  }
}
```

### 字段说明

| 字段名 | 类型 | 说明 |
|--------|------|------|
| userId | int64 | 用户 ID |
| memberLevel | int32 | 新会员等级 |
| memberLevelName | string | 会员等级名称 |
| memberStatus | int32 | 会员状态 0-正常 |
| expireTime | int64 | 新会员过期时间戳 |
| remainingDays | int64 | 剩余天数 |
| operationType | int32 | 操作类型 1-开通 2-续费 3-升级 4-降级 5-延长 |
| oldLevel | int32 | 原会员等级 |
| oldExpireTime | int64 | 原会员过期时间戳 |
| remark | string | 操作备注 |
| updateTime | int64 | 更新时间戳 |

## 错误响应

### 400 参数错误

```json
{
  "code": 400,
  "msg": "参数异常",
  "data": null
}
```

常见错误：
- 操作备注不能为空
- 到期时间必须晚于当前时间
- 禁止设置超过 10 年的有效期
- 用户账号状态异常

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
  "msg": "更新失败",
  "data": null
}
```

## 业务逻辑

### 1. 操作流程

```
管理员请求 → 校验权限 → 校验用户合法性 → 校验会员配置 → 
计算新周期 → 更新数据 → 记录日志 → 同步权益 → 返回结果
```

### 2. 参数校验

**必填参数校验**
- `userId`：非空，用户必须存在
- `memberLevel`：必须在 0-3 范围内
- `expireTime`：必须晚于当前时间
- `remark`：操作备注必填（审计要求）

**时间合法性校验**
- 到期时间必须晚于当前时间
- 禁止设置超过 10 年的有效期（防止恶意设置）

### 3. 操作类型自动判断

如果未指定 `operationType`，系统自动判断：

| 条件 | 操作类型 |
|------|---------|
| 原到期时间 = 0 | 1-开通 |
| 新等级 > 原等级 | 3-升级 |
| 新等级 < 原等级 | 4-降级 |
| 新到期时间 > 原到期时间 | 2-续费 |
| 其他 | 5-延长 |

### 4. 会员周期计算（核心逻辑）

**开通会员**
- 从当前时间开始计算
- `expireTime = now + days * 86400`

**续费/延长**
- 在原到期时间上叠加天数
- `expireTime = oldExpireTime + days * 86400`
- 仅当原会员未过期时生效

**升级/降级**
- 保留剩余时长
- 立即按新等级生效
- `expireTime = oldExpireTime`（保持不变）

### 5. 数据更新

**更新用户表（users）**
- 更新 `member_level` 字段

**更新会员表（user_member）**
- 更新 `expire_at`（到期时间）
- 更新 `status`（状态设为正常）
- 更新 `updated_at`（最后操作时间）
- 更新 `auto_renew`（默认不自动续费）

**记录操作日志（member_operate_log）**
- 管理员 ID
- 用户 ID
- 操作类型
- 原等级/新等级
- 原到期时间/新到期时间
- 操作备注
- 操作时间

### 6. 同步权益状态

- 异步刷新 Redis 缓存
- 会员权益实时生效
- 设备绑定上限立即更新

## 操作类型说明

| 类型 | 代码 | 说明 | 示例 |
|------|------|------|------|
| 开通 | 1 | 用户首次开通会员 | 普通会员 → VIP |
| 续费 | 2 | 延续会员有效期 | VIP 剩余 7 天 → 续费 365 天 |
| 升级 | 3 | 提升会员等级 | VIP → SVIP |
| 降级 | 4 | 降低会员等级 | SVIP → VIP |
| 延长 | 5 | 延长有效期（不改变等级） | VIP 剩余 7 天 → 延长 30 天 |

## 异常处理

| 异常类型 | 处理方式 |
|----------|----------|
| 未登录 | 返回 401 未授权 |
| 无权限 | 返回 403 权限不足 |
| 用户不存在 | 返回 404 用户不存在 |
| 会员等级不合法 | 返回 400 参数异常 |
| 到期时间不合法 | 返回 400 参数异常 |
| 用户账号状态异常 | 返回 400 参数异常 |
| 备注为空 | 返回 400 参数异常 |
| 设置超过 10 年 | 返回 400 参数异常 |

## 约束规则

### 1. 权限约束

- ✅ 只允许管理员操作
- ✅ 普通用户禁止访问
- ✅ 需要「会员管理 - 开通/修改」权限

### 2. 审计约束

- ✅ 必须填写备注（用于追溯）
- ✅ 每次修改都留痕
- ✅ 支持后台查看操作日志
- ✅ 记录操作管理员 ID

### 3. 数据约束

- ✅ 禁止设置过长时间（≤ 10 年）
- ✅ 到期时间必须晚于当前时间
- ✅ 会员等级必须在合法范围内
- ✅ 禁止越权修改高权限用户

### 4. 实时性约束

- ✅ 修改后立即刷新权益
- ✅ 权益立即生效
- ✅ 设备绑定上限实时更新

## 前端交互要点

### 1. 表单设计

**必填字段**
- 用户 ID（搜索选择）
- 会员等级（下拉选择）
- 到期时间（日期选择器）或 有效期天数（数字输入）
- 操作备注（文本输入，最少 5 字）

**可选字段**
- 操作类型（下拉选择，可自动判断）

### 2. 操作提示

**开通会员**
- 显示：从当前时间开始计算
- 提示：用户将立即获得会员权益

**续费会员**
- 显示：原到期时间 + 续费天数 = 新到期时间
- 提示：在原有效期基础上叠加

**升级会员**
- 显示：原等级 → 新等级
- 提示：会员权益立即升级

**降级会员**
- 显示：原等级 → 新等级
- 警告：降级后部分权益将失效

### 3. 确认对话框

在提交前显示确认信息：
```
操作类型：升级
用户：张三（1001）
原等级：VIP 会员
新等级：SVIP 会员
原到期时间：2024-01-01
新到期时间：2025-01-01
操作备注：VIP 升级 SVIP，用户要求提升音质体验

确认执行此操作吗？
```

### 4. 结果展示

操作成功后显示：
- ✅ 新会员等级
- ✅ 新到期时间
- ✅ 剩余天数
- ✅ 操作类型标签

## 数据表说明

### 主要数据表

- `users`：用户基础信息表
- `user_member`：会员信息表
- `member_operate_log`：会员操作日志表
- `member_level_config`：会员等级配置表

### 字段映射

| 请求字段 | 数据表 | 字段名 |
|----------|--------|--------|
| userId | users | user_id |
| memberLevel | users | member_level |
| expireTime | user_member | expire_at |
| operationType | member_operate_log | operate_type |
| remark | member_operate_log | remark |
| operatorId | member_operate_log | operate_admin |

## 操作日志查询

管理员可以在后台查看会员操作历史：

```sql
SELECT 
  operate_time,
  operate_admin,
  user_id,
  operate_type,
  old_level,
  new_level,
  old_expire_time,
  new_expire_time,
  remark
FROM member_operate_log
WHERE user_id = 1001
ORDER BY operate_time DESC
LIMIT 20;
```

## 最佳实践

### 1. 备注规范

建议备注包含以下信息：
- 操作原因（用户要求、活动赠送、补偿等）
- 相关单号（订单号、工单号等）
- 特殊说明

**示例**
- ✅ "VIP 升级 SVIP，用户要求提升音质体验，订单号：ORDER_20240101_001"
- ✅ "续费 365 天，双 11 活动赠送，活动编号：ACT_20231111"
- ✅ "延长 30 天，系统故障补偿，工单号：TICKET_20240101_001"
- ❌ "升级"（过于简单）
- ❌ ""（空备注）

### 2. 时间选择

- 开通/续费：建议选择整年/整月（365 天、30 天）
- 避免选择奇怪的天数（如 123 天）
- 到期时间建议设置为 23:59:59

### 3. 等级调整

- 升级：随时可以执行
- 降级：建议先与用户确认
- 补偿：使用延长操作，不影响原等级

### 4. 批量操作

对于批量开通/续费：
- 建议先导出用户列表
- 确认用户信息无误
- 逐个操作并记录备注
- 操作后抽查验证
