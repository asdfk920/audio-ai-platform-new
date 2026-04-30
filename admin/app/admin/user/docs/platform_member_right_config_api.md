# 平台会员权益配置 API 文档

## 接口说明

实现管理后台会员权益配置功能，管理员可配置各会员等级的权益项，包括设备绑定上限、内容权限、高音质、OTA 升级等权益。

## 接口地址

```
POST /api/v1/platform-member/right-config
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
  "levelId": 2,
  "levelName": "SVIP 会员",
  "status": 1,
  "rights": [
    {
      "rightKey": "device_bind_limit",
      "rightName": "设备绑定上限",
      "rightValue": 10,
      "rightType": "int"
    },
    {
      "rightKey": "vip_content",
      "rightName": "付费内容权限",
      "rightValue": true,
      "rightType": "bool"
    },
    {
      "rightKey": "high_quality_audio",
      "rightName": "高音质",
      "rightValue": true,
      "rightType": "bool"
    },
    {
      "rightKey": "spatial_audio",
      "rightName": "空间音频",
      "rightValue": true,
      "rightType": "bool"
    },
    {
      "rightKey": "ota_upgrade",
      "rightName": "OTA 升级",
      "rightValue": true,
      "rightType": "bool"
    },
    {
      "rightKey": "download_speed",
      "rightName": "下载速度",
      "rightValue": "unlimited",
      "rightType": "string"
    },
    {
      "rightKey": "download_parallel",
      "rightName": "下载并发数",
      "rightValue": 16,
      "rightType": "int"
    },
    {
      "rightKey": "ad_free",
      "rightName": "免广告",
      "rightValue": true,
      "rightType": "bool"
    },
    {
      "rightKey": "cloud_storage",
      "rightName": "云存储空间",
      "rightValue": "100GB",
      "rightType": "string"
    },
    {
      "rightKey": "exclusive_service",
      "rightName": "专属客服",
      "rightValue": true,
      "rightType": "bool"
    },
    {
      "rightKey": "vip_avatar",
      "rightName": "会员头像",
      "rightValue": true,
      "rightType": "bool"
    },
    {
      "rightKey": "vip_badge",
      "rightName": "会员标识",
      "rightValue": true,
      "rightType": "bool"
    },
    {
      "rightKey": "early_access",
      "rightName": "提前收听",
      "rightValue": true,
      "rightType": "bool"
    }
  ],
  "remark": "配置 SVIP 会员权益，提升设备绑定上限至 10 台"
}
```

### 参数说明

| 参数名 | 类型 | 必填 | 说明 | 示例 |
|--------|------|------|------|------|
| levelId | int32 | 是 | 会员等级 ID（0-普通 1-VIP 2-SVIP 3-终身） | 2 |
| levelName | string | 否 | 会员等级名称 | SVIP 会员 |
| status | int32 | 是 | 状态（0-禁用 1-启用） | 1 |
| rights | array | 是 | 权益配置列表 | 见下方权益项说明 |
| remark | string | 否 | 备注 | 配置说明 |

### 权益项说明

**标准权益配置项（13 项）**

| 权益键 | 权益名称 | 类型 | 说明 | 示例值 |
|--------|---------|------|------|--------|
| device_bind_limit | 设备绑定上限 | int | 最大绑定设备数量 | 10 |
| vip_content | 付费内容权限 | bool | 是否可观看付费内容 | true |
| high_quality_audio | 高音质 | bool | 是否可使用高音质 | true |
| spatial_audio | 空间音频 | bool | 是否可使用空间音频 | true |
| ota_upgrade | OTA 升级 | bool | 是否支持 OTA 升级 | true |
| download_speed | 下载速度 | string | 下载速度限制 | unlimited/1MB/s |
| download_parallel | 下载并发数 | int | 同时下载任务数 | 16 |
| ad_free | 免广告 | bool | 是否免除广告 | true |
| cloud_storage | 云存储空间 | string | 云存储容量 | 100GB |
| exclusive_service | 专属客服 | bool | 是否专属客服 | true |
| vip_avatar | 会员头像 | bool | 是否会员头像框 | true |
| vip_badge | 会员标识 | bool | 是否会员标识 | true |
| early_access | 提前收听 | bool | 是否提前收听 | true |

### 请求示例

```bash
curl -X POST 'http://localhost:8000/api/v1/platform-member/right-config' \
  -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...' \
  -H 'Content-Type: application/json' \
  -d '{
    "levelId": 2,
    "levelName": "SVIP 会员",
    "status": 1,
    "rights": [
      {
        "rightKey": "device_bind_limit",
        "rightName": "设备绑定上限",
        "rightValue": 10,
        "rightType": "int"
      },
      {
        "rightKey": "vip_content",
        "rightName": "付费内容权限",
        "rightValue": true,
        "rightType": "bool"
      }
    ],
    "remark": "配置 SVIP 会员权益"
  }'
```

## 成功响应

### 响应数据结构

```json
{
  "code": 200,
  "msg": "配置成功",
  "data": {
    "levelId": 2,
    "levelName": "SVIP 会员",
    "status": 1,
    "rights": [
      {
        "rightKey": "device_bind_limit",
        "rightName": "设备绑定上限",
        "rightValue": 10,
        "rightType": "int"
      },
      {
        "rightKey": "vip_content",
        "rightName": "付费内容权限",
        "rightValue": true,
        "rightType": "bool"
      }
    ],
    "updateTime": 1704153600,
    "operatorId": 100,
    "operatorName": "管理员"
  }
}
```

### 字段说明

| 字段名 | 类型 | 说明 |
|--------|------|------|
| levelId | int32 | 会员等级 ID |
| levelName | string | 会员等级名称 |
| status | int32 | 状态（0-禁用 1-启用） |
| rights | array | 权益配置列表 |
| updateTime | int64 | 更新时间戳 |
| operatorId | int64 | 操作管理员 ID |
| operatorName | string | 操作人名称 |

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
- 会员等级不合法
- 非法的权益项
- device_bind_limit 必须大于等于 0
- download_parallel 必须大于等于 0

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

### 500 服务器错误

```json
{
  "code": 500,
  "msg": "保存失败",
  "data": null
}
```

## 业务逻辑

### 1. 操作流程

```
管理员请求 → 校验权限 → 校验配置参数 → 查询原有配置 → 
保存/更新配置 → 记录日志 → 刷新缓存 → 返回结果
```

### 2. 参数校验

**会员等级校验**
- ✅ levelId 必须在 0-3 范围内
- ✅ 等级名称可选，用于显示

**权益项校验**
- ✅ 权益键必须在系统允许范围内
- ✅ 数值类型必须正确（int/bool/string）
- ✅ 数值必须合法（设备数≥0、并发数≥0）

**状态校验**
- ✅ status 必须为 0 或 1

### 3. 核心逻辑

**查询原有配置**
- 查询会员等级配置表
- 获取原有权益配置
- 用于日志记录对比

**保存/更新配置**
- 存在则更新权益配置
- 不存在则新增配置
- 记录操作人和操作时间

**刷新全局缓存**
- 清除 Redis 中所有会员等级权益缓存
- 新配置立即生效
- 用户下次校验自动读取最新规则

**记录操作日志**
- 操作人
- 会员等级
- 修改前权益
- 修改后权益
- 操作时间
- 备注

### 4. 数据更新

**更新会员等级配置表（member_level_config）**
- 更新 `level_name`（等级名称）
- 更新 `status`（启用状态）
- 更新 `rights`（权益配置 JSON）
- 更新 `updated_at`（更新时间）

**记录操作日志（member_operate_log）**
- 创建权益配置操作记录
- 记录操作类型（4-权益配置）
- 记录新旧权益对比

## 会员等级说明

| 等级 ID | 等级名称 | 说明 |
|--------|---------|------|
| 0 | 普通会员 | 基础免费用户 |
| 1 | VIP 会员 | 初级付费会员 |
| 2 | SVIP 会员 | 高级付费会员 |
| 3 | 终身会员 | 永久会员 |

## 异常处理

| 异常类型 | 处理方式 |
|----------|----------|
| 未登录 | 返回 401 未授权 |
| 无权限 | 返回 403 权限不足 |
| 等级不存在 | 返回 400 参数异常 |
| 权益项非法 | 返回 400 参数异常 |
| 配置重复 | 自动更新配置 |
| 数值不合法 | 返回 400 参数异常 |

## 约束规则

### 1. 权限约束

- ✅ 只允许管理员操作
- ✅ 需要「会员管理 - 权益配置」权限
- ✅ 普通用户禁止访问

### 2. 审计约束

- ✅ 必须记录操作日志
- ✅ 记录新旧权益对比
- ✅ 支持后台查看操作日志
- ✅ 记录操作管理员 ID

### 3. 数据约束

- ✅ 禁止配置不合理数值
- ✅ 设备绑定数必须≥0
- ✅ 下载并发数必须≥0
- ✅ 权益键必须在允许范围内

### 4. 实时性约束

- ✅ 配置后立即全局生效
- ✅ 刷新 Redis 缓存
- ✅ 用户权益实时同步

## 前端交互要点

### 1. 表单设计

**等级选择**
- 下拉选择：普通会员/VIP 会员/SVIP 会员/终身会员
- 显示当前等级的权益配置

**权益配置**
- 开关型权益：使用 Switch 组件（是/否）
- 数值型权益：使用 InputNumber 组件
- 字符串型权益：使用 Input 组件

**状态设置**
- 启用/禁用该等级

**备注**
- 文本域，记录配置说明

### 2. 权益配置展示

**分组展示**
- 设备权限组：设备绑定上限
- 内容权限组：付费内容、高音质、空间音频、提前收听
- 下载权限组：下载速度、下载并发数
- 服务权限组：免广告、云存储、专属客服
- 身份标识组：会员头像、会员标识
- 系统权限组：OTA 升级

### 3. 配置对比

在保存前显示变更对比：
```
等级：SVIP 会员
状态：启用

权益变更：
- 设备绑定上限：5 → 10（+5）
- 下载并发数：8 → 16（+8）
- 云存储空间：50GB → 100GB（+50GB）

确认保存此配置吗？
```

### 4. 结果展示

操作成功后显示：
- ✅ 配置成功提示
- ✅ 等级名称
- ✅ 权益数量
- ✅ 更新时间

## 数据表说明

### 主要数据表

- `member_level_config`：会员等级配置表
- `member_operate_log`：会员操作日志表

### 字段映射

| 请求字段 | 数据表 | 字段名 |
|----------|--------|--------|
| levelId | member_level_config | level_id |
| levelName | member_level_config | level_name |
| status | member_level_config | status |
| rights | member_level_config | rights (JSON) |
| remark | member_operate_log | remark |
| operatorId | member_operate_log | operate_admin |

## 操作日志查询

管理员可以在后台查看权益配置历史：

```sql
SELECT 
  operate_time,
  operate_admin,
  operate_type,
  old_level,
  new_level,
  old_rights,
  new_rights,
  remark
FROM member_operate_log
WHERE operate_type = 4  -- 权益配置操作
ORDER BY operate_time DESC
LIMIT 20;
```

## 最佳实践

### 1. 备注规范

建议备注包含以下信息：
- 配置目的（提升用户体验、活动支持等）
- 变更内容（提升了哪些权益）
- 生效时间（立即生效/定时生效）

**示例**
- ✅ "配置 SVIP 会员权益，提升设备绑定上限至 10 台，支持双 11 活动"
- ✅ "优化 VIP 会员下载权限，并发数提升至 16，提升下载体验"
- ✅ "新增终身会员云存储空间至 500GB，提升会员价值"
- ❌ "配置权益"（过于简单）
- ❌ ""（空备注）

### 2. 权益配置策略

**普通会员（levelId=0）**
- 设备绑定上限：1-2 台
- 基础内容权限
- 标准音质
- 有广告
- 无云存储或小额度

**VIP 会员（levelId=1）**
- 设备绑定上限：3-5 台
- 付费内容权限
- 高音质
- 免广告
- 适量云存储（如 50GB）

**SVIP 会员（levelId=2）**
- 设备绑定上限：10 台
- 全部内容权限
- 高音质 + 空间音频
- 免广告
- 大额云存储（如 100GB）
- 专属客服
- 提前收听

**终身会员（levelId=3）**
- 所有 SVIP 权益
- 更高额度（如 500GB 云存储）
- 永久有效

### 3. 数值建议

| 权益项 | 普通 | VIP | SVIP | 终身 |
|--------|------|-----|------|------|
| device_bind_limit | 2 | 5 | 10 | 20 |
| download_parallel | 2 | 8 | 16 | 32 |
| cloud_storage | 10GB | 50GB | 100GB | 500GB |

### 4. 批量操作

对于批量配置：
- 建议逐个等级配置
- 配置后抽查验证
- 定期审计配置记录
- 评估权益使用率

## 缓存刷新机制

### 1. 缓存策略

**Redis 缓存键**
```
member_rights:{levelId}
```

**缓存内容**
```json
{
  "levelId": 2,
  "levelName": "SVIP 会员",
  "status": 1,
  "rights": [...],
  "updateTime": 1704153600
}
```

**过期时间**
- 建议设置 24 小时
- 配置更新时主动清除

### 2. 刷新流程

```
配置保存成功 → 异步刷新缓存 → 清除 Redis 旧数据 → 
用户请求时重新加载 → 写入新缓存
```

### 3. 实时生效

- 配置后立即清除缓存
- 用户下次请求自动读取最新配置
- 权益校验接口实时生效

## 关联功能

### 1. 会员等级管理

在会员等级管理页：
- 查看各等级权益配置
- 一键配置权益
- 复制其他等级配置
- 查看配置历史

### 2. 权益使用统计

在数据统计页：
- 各等级权益使用率
- 高频使用的权益项
- 用户满意度调查
- 权益成本分析

### 3. 用户权益查询

在用户详情页：
- 查看用户当前等级
- 查看用户可用权益
- 权益使用记录
- 权益剩余次数

## 审计要求

### 1. 操作留痕

每次权益配置操作必须记录：
- 操作时间
- 操作管理员
- 会员等级
- 修改前权益列表
- 修改后权益列表
- 备注说明

### 2. 权益追溯

权益配置必须可追溯：
- 配置变更历史
- 每次变更的详细说明
- 审批记录（如需要）
- 生效时间

### 3. 定期审计

建议定期审计：
- 每周审查权益配置记录
- 每月统计权益配置变更
- 每季度评估权益合理性
- 每年优化权益体系
