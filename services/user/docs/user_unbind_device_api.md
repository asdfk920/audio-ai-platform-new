# 用户解绑设备接口文档

## 接口说明

- **接口地址**: `/api/v1/user/device/unbind`
- **请求方式**: `POST`
- **功能**: 用户解绑设备，解除用户与设备的关联关系
- **权限要求**: 需要 JWT 登录认证，仅绑定用户或管理员可操作

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
  "device_sn": "SN1234567890"      // 必填：设备唯一序列号（最大 64 字符）
}
```

### 参数说明
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| device_sn | string | 是 | 设备唯一序列号，必须是当前用户已绑定的设备 |

---

## 后端处理全流程

### 1. 用户身份与权限校验
- 根据 JWT token 解析当前用户 ID
- 查询用户信息，校验用户存在
- 校验用户账号状态正常（status=1）

### 2. 设备合法性校验
- 根据 device_sn 查询设备信息
- 校验设备存在
- 校验设备状态正常（非禁用、非报废）

### 3. 绑定关系校验
- 查询该设备当前绑定用户是否为当前操作人
- 校验设备处于绑定中状态（status=1）
- 无权限则直接拒绝（设备归属其他用户）

### 4. 解除绑定关系（事务）
- 开启数据库事务
- 更新 `user_device_bind` 表：将状态改为已解绑（status=0），记录解绑时间
- 清空 `device` 表中的绑定用户 ID、绑定时间，将绑定状态改为未绑定（bind_status=0）
- 更新 `user_profile` 表：减少用户已绑定设备数
- 写入解绑操作日志到 `user_device_bind_log` 表
- 提交事务（失败自动回滚）

### 5. 同步设备与用户状态（异步）
- 通过 MQTT/长连接向设备下发解绑通知
- 更新设备影子（Redis 缓存），清除绑定用户信息
- 触发设备权限同步，清除用户设备权限

### 6. 返回结果

---

## 返回结果

### 成功响应（200）
```json
{
  "code": 0,
  "msg": "解绑成功",
  "data": {
    "device_sn": "SN1234567890",
    "user_id": 123,
    "status": 0
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

#### 403 - 无权解绑
```json
{
  "code": 2017,
  "msg": "无权解绑该设备，设备归属其他用户"
}
```

#### 设备状态异常
```json
{
  "code": 2015,
  "msg": "设备已被禁用，无法解绑"
}
```
```json
{
  "code": 2016,
  "msg": "设备未激活，无法解绑"
}
```
```json
{
  "code": 2018,
  "msg": "设备已报废，无法解绑"
}
```

#### 设备未绑定
```json
{
  "code": 2009,
  "msg": "该设备未绑定，无需解绑"
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

## 数据库操作

### 1. 更新绑定关系表
```sql
UPDATE public.user_device_bind
SET status = 0,
    unbound_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE user_id = $1 AND device_id = $2 AND status = 1;
```

### 2. 清空设备绑定信息
```sql
UPDATE public.device
SET bound_user_id = NULL,
    bound_at = NULL,
    bind_status = 0,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1;
```

### 3. 更新用户绑定数
```sql
UPDATE public.user_profile
SET device_count = GREATEST(device_count - 1, 0),
    last_unbind_time = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE user_id = $1;
```

### 4. 写入操作日志
```sql
INSERT INTO public.user_device_bind_log
  (user_id, device_id, sn, operator, action, action_time)
VALUES
  ($1, $2, $3, $4, 'unbind', CURRENT_TIMESTAMP);
```

---

## 安全与规则约束

### 1. 权限隔离
- 仅绑定用户与管理员可执行解绑
- 设备归属其他用户时，禁止解绑

### 2. 状态校验
- 设备已解绑、已禁用、已报废时，不予处理
- 防止重复解绑操作

### 3. 事务回滚
- 解绑过程异常需回滚
- 保证数据一致性

### 4. 操作审计
- 所有解绑操作全程留痕
- 记录操作人、操作时间、操作类型

### 5. 设备可重新绑定
- 解绑后设备可被其他用户重新绑定
- 设备状态恢复为未绑定

### 6. HTTPS 传输
- 解绑请求全程加密
- 防止用户信息泄露

---

## 业务联动

### 用户端
- 刷新设备列表，移除已解绑设备
- 清除设备控制、状态查询等权限
- 用户绑定设备数自动减少

### 后台端
- 设备管理页同步更新绑定状态为「未绑定」
- 用户管理页同步更新已绑定设备数
- 可查看解绑操作日志

### 设备端
- 收到解绑通知后，清除本地绑定信息
- 切换为未绑定状态
- 可被其他用户重新绑定

---

## 错误码说明

| 错误码 | 说明 | 解决方案 |
|--------|------|----------|
| 0 | 解绑成功 | - |
| 1001 | 用户不存在 | 检查用户 ID 是否正确 |
| 1004 | Token 无效或过期 | 重新登录获取新 token |
| 1008 | 参数错误 | 检查请求参数格式 |
| 2001 | 设备不存在 | 检查设备 SN 是否正确 |
| 2009 | 设备未绑定 | 设备当前未绑定状态 |
| 2015 | 设备已被禁用 | 联系管理员启用设备 |
| 2016 | 设备未激活 | 设备需先激活 |
| 2017 | 无权解绑 | 设备归属其他用户 |
| 2018 | 设备已报废 | 设备已报废无法操作 |
| 9001 | 数据库错误 | 联系技术支持 |

---

## 调用示例

### cURL
```bash
curl -X POST http://localhost:8888/api/v1/user/device/unbind \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "Content-Type: application/json" \
  -d '{
    "device_sn": "SN1234567890"
  }'
```

### JavaScript
```javascript
async unbindDevice(deviceSn) {
  const res = await fetch('/api/v1/user/device/unbind', {
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
    console.log('解绑成功', result.data);
    // 刷新设备列表
  } else {
    console.error('解绑失败', result.msg);
  }
}
```

---

## 实现文件清单

```
services/user/
├── internal/
│   ├── types/
│   │   └── types.go                          # DTO 定义（UnbindUserDeviceReq/Resp）
│   ├── logic/
│   │   └── unbind_user_device_logic.go       # 业务逻辑层（新建）
│   ├── handler/
│   │   └── unbind_user_device_handler.go     # API 处理器（已存在）
│   └── repo/dao/
│       └── device_bind_dao.go                # 解绑 DAO 方法（已更新）
└── docs/
    └── user_unbind_device_api.md             # API 文档（新建）
```

---

## 注意事项

1. **权限校验**: 必须校验设备归属当前用户，防止越权操作
2. **事务安全**: 所有数据库操作在事务中执行，失败自动回滚
3. **状态同步**: 解绑后设备状态自动恢复为未绑定
4. **日志审计**: 所有解绑操作都会记录日志，便于追溯
5. **异步通知**: 设备状态通知采用异步方式，不影响主流程
6. **并发控制**: 通过数据库约束和代码逻辑双重保证并发安全
7. **幂等性**: 对已解绑设备重复解绑时，返回友好提示

---

## 与绑定接口的区别

| 项目 | 绑定接口 | 解绑接口 |
|------|----------|----------|
| 操作方向 | 建立关联 | 解除关联 |
| 设备状态 | 未绑定 → 已绑定 | 已绑定 → 未绑定 |
| 用户绑定数 | +1 | -1 |
| 权限校验 | 设备未被绑定 | 设备归属当前用户 |
| 设备状态要求 | 启用、已激活 | 非禁用、非报废 |

---

**版本**: v1.0.0  
**更新时间**: 2026-04-08  
**状态**: ✅ 已完成并编译通过
