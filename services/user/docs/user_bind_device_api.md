# 用户绑定设备接口文档

## 接口说明

- **接口地址**: `/api/v1/user/device/bind`
- **请求方式**: `POST`
- **功能**: 用户绑定设备，建立用户与设备的关联关系
- **权限要求**: 需要 JWT 登录认证

---

## 请求参数

### Header
```
Authorization: Bearer <access_token>
Content-Type: application/json
```

### Body
```json
{
  "device_sn": "SN1234567890"       // 必填：设备唯一序列号（最大 64 字符）
}
```

### 参数说明
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| device_sn | string | 是 | 设备唯一序列号，必须在平台已注册 |

---

## 后端处理全流程

### 1. 用户合法性校验
- 根据 JWT token 解析当前用户 ID
- 查询用户信息，校验用户存在
- 校验用户账号状态为启用（status=1）
- 校验用户未被封禁

### 2. 设备合法性校验
- 根据 device_sn 查询设备信息
- 校验设备存在
- 校验设备状态为启用（status=1）
- 校验设备未被禁用

### 3. 绑定状态校验
- 查询设备绑定关系表
- 校验该设备未被其他用户绑定
- 支持幂等操作（已绑定给当前用户时直接返回成功）

### 4. 配额校验
- 校验用户已绑定设备数未超过最大绑定上限
- 默认最大绑定数：10 台（可通过配置调整）

### 5. 建立绑定关系（事务）
- 开启数据库事务
- 向 `user_device_bind` 表插入绑定记录
- 更新 `device` 表：写入绑定用户 ID、绑定时间、绑定状态
- 更新 `user_profile` 表：累加用户已绑定设备数
- 写入绑定操作日志到 `user_device_bind_log` 表
- 提交事务

### 6. 同步设备状态（异步）
- 通过 MQTT/长连接向设备下发绑定通知（可选）
- 更新设备影子（Redis 缓存）
- 触发设备权限同步

### 7. 返回绑定结果

---

## 返回结果

### 成功响应（200）
```json
{
  "code": 0,
  "msg": "绑定成功",
  "data": {
    "user_id": 123,
    "device_sn": "SN1234567890",
    "device_name": "SN1234567890",
    "bind_time": "2026-04-08 15:30:45"
  }
}
```

### 失败响应

#### 400 - 参数错误
```json
{
  "code": 1008,
  "msg": "设备序列号不能为空"
}
```

#### 401 - 用户未登录/权限不足
```json
{
  "code": 1004,
  "msg": "登录已过期或无效，请重新登录"
}
```

#### 404 - 设备/用户不存在
```json
{
  "code": 1001,
  "msg": "用户不存在"
}
```
```json
{
  "code": 2001,
  "msg": "设备不存在"
}
```

#### 403 - 绑定失败
```json
{
  "code": 2008,
  "msg": "该设备已被用户 456 绑定"
}
```
```json
{
  "code": 2008,
  "msg": "已达到最大绑定设备数限制（10 台）"
}
```

#### 设备/用户状态异常
```json
{
  "code": 1025,
  "msg": "用户账号已被禁用"
}
```
```json
{
  "code": 2015,
  "msg": "设备已被禁用"
}
```
```json
{
  "code": 2016,
  "msg": "设备未激活，请先激活设备"
}
```

#### 500 - 系统内部错误
```json
{
  "code": 9001,
  "msg": "系统繁忙，请稍后重试"
}
```

---

## 数据库表结构

### user_device_bind（用户 - 设备关联表）
```sql
CREATE TABLE public.user_device_bind (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL COMMENT '用户 ID',
    device_id BIGINT NOT NULL COMMENT '设备 ID',
    sn VARCHAR(64) NOT NULL COMMENT '设备序列号',
    alias VARCHAR(32) COMMENT '设备别名',
    is_default BOOLEAN DEFAULT FALSE,
    bind_type SMALLINT DEFAULT 1 COMMENT '绑定类型 1=主动绑定',
    status SMALLINT DEFAULT 1 COMMENT '1=绑定中 0=已解绑',
    bound_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    unbound_at TIMESTAMP NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    operator VARCHAR(64) COMMENT '操作人 ID',
    UNIQUE KEY uk_device (device_id, status)
);
```

### device（设备表相关字段）
```sql
ALTER TABLE public.device ADD COLUMN bound_user_id BIGINT COMMENT '绑定用户 ID';
ALTER TABLE public.device ADD COLUMN bound_at TIMESTAMP COMMENT '绑定时间';
ALTER TABLE public.device ADD COLUMN bind_status SMALLINT DEFAULT 0 COMMENT '0=未绑定 1=已绑定';
```

### user_device_bind_log（绑定操作日志表）
```sql
CREATE TABLE public.user_device_bind_log (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    device_id BIGINT NOT NULL,
    sn VARCHAR(64) NOT NULL,
    operator VARCHAR(64) COMMENT '操作人 ID',
    action VARCHAR(20) NOT NULL COMMENT 'bind/unbind',
    action_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

---

## 安全与规则约束

### 1. 唯一性约束
- 一台设备仅允许绑定一个用户
- 通过数据库唯一索引和代码逻辑双重保证

### 2. 权限隔离
- 仅设备绑定用户/管理员可操作该设备
- 其他用户无权限操作已绑定设备

### 3. 操作审计
- 所有绑定操作全程留痕
- 记录操作人、操作时间、操作类型
- 支持溯源查询

### 4. 防刷限制
- 同一用户/设备短时间内频繁绑定请求，触发限流
- 建议配置：同一设备 1 分钟内最多绑定 5 次

### 5. HTTPS 传输
- 绑定请求全程加密
- 防止 SN、用户信息泄露

### 6. 事务回滚
- 绑定过程中任一环节失败，自动回滚
- 保证数据一致性

---

## 业务联动

### 用户端
- 刷新设备列表，展示已绑定设备
- 开通设备控制、状态查询等权限
- 可在「我的设备」页面查看和管理

### 后台端
- 设备管理页同步展示绑定用户信息
- 用户管理页同步展示已绑定设备数
- 支持管理员解绑操作

### 设备端
- 收到绑定通知后，切换为绑定状态
- 仅允许绑定用户操作
- 支持解绑通知

---

## 错误码说明

| 错误码 | 说明 | 解决方案 |
|--------|------|----------|
| 0 | 绑定成功 | - |
| 1001 | 用户不存在 | 检查用户 ID 是否正确 |
| 1004 | Token 无效或过期 | 重新登录获取新 token |
| 1008 | 参数错误 | 检查请求参数格式 |
| 1025 | 用户账号已禁用 | 联系管理员解封 |
| 2001 | 设备不存在 | 检查设备 SN 是否正确 |
| 2008 | 设备已被其他用户绑定 | 先解绑或更换设备 |
| 2015 | 设备已被禁用 | 联系管理员启用设备 |
| 2016 | 设备未激活 | 设备需先激活才能绑定 |
| 9001 | 数据库错误 | 联系技术支持 |

---

## 调用示例

### cURL
```bash
curl -X POST http://localhost:8888/api/v1/user/device/bind \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "Content-Type: application/json" \
  -d '{
    "device_sn": "SN1234567890"
  }'
```

### JavaScript
```javascript
async bindDevice(deviceSn) {
  const res = await fetch('/api/v1/user/device/bind', {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${accessToken}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      device_sn: deviceSn
    })
  });
  
  const result = await res.json();
  if (result.code === 0) {
    console.log('绑定成功', result.data);
  } else {
    console.error('绑定失败', result.msg);
  }
}
```

---

## 实现文件清单

```
services/user/
├── internal/
│   ├── types/
│   │   └── types.go                          # DTO 定义
│   ├── logic/
│   │   └── bind_user_device_logic.go         # 业务逻辑层
│   ├── handler/
│   │   └── bind_user_device_handler.go       # API 处理器
│   ├── repo/dao/
│   │   ├── device_bind_dao.go                # 设备绑定 DAO
│   │   └── user_device_bind_repo.go          # 绑定关系 Repo
│   └── config/
│       └── config.go                         # 配置（MaxDeviceBinds）
└── docs/
    └── user_bind_device_api.md               # API 文档
```

---

## 注意事项

1. **幂等性**: 同一设备重复绑定给同一用户时，直接返回成功
2. **事务安全**: 所有数据库操作在事务中执行，失败自动回滚
3. **配额限制**: 用户绑定设备数有上限，可通过配置调整
4. **日志审计**: 所有绑定操作都会记录日志，便于追溯
5. **异步同步**: 设备状态同步采用异步方式，不影响主流程
6. **并发控制**: 通过数据库约束和代码逻辑双重保证并发安全

---

**版本**: v1.0.0  
**更新时间**: 2026-04-08  
**状态**: ✅ 已完成并编译通过
