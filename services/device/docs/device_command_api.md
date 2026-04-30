# 设备指令下发 API 文档

## 接口说明
- **接口地址**: `/api/v1/device/command`
- **请求方式**: POST
- **功能**: 用户向已绑定的设备下发控制指令，支持在线实时下发和离线缓存
- **认证**: JWT Token（用户登录态）

## 请求参数

### Header
```
Authorization: Bearer <user_access_token>
Content-Type: application/json
```

### Body
```json
{
  "device_sn": "SN1234567890",
  "command": "set_volume",
  "params": {
    "volume": 80
  }
}
```

### 参数说明
| 参数名 | 类型 | 必填 | 说明 | 示例 |
|--------|------|------|------|------|
| device_sn | string | 是 | 设备序列号 | "SN1234567890" |
| command | string | 是 | 指令类型 | "restart", "set_volume", "set_mode" |
| params | object | 否 | 指令参数 | {"volume": 80} |

### 支持的指令类型
| 指令类型 | 说明 | 参数 |
|----------|------|------|
| restart | 设备重启 | 无 |
| reboot | 设备重启（同 restart） | 无 |
| factory_reset | 恢复出厂设置 | 无 |
| set_volume | 设置音量 | {"volume": 0-100} |
| set_mode | 设置运行模式 | {"mode": "normal/sleep/party/quiet"} |
| play | 播放 | 无 |
| pause | 暂停 | 无 |
| stop | 停止 | 无 |
| next | 下一首 | 无 |
| prev | 上一首 | 无 |

## 成功响应（200）
```json
{
  "command_id": "cmd_a1b2c3d4e5f6",
  "device_sn": "SN1234567890",
  "command": "set_volume",
  "status": "sent",
  "sent_at": 1775692800,
  "message": "指令下发成功"
}
```

### 响应字段说明
| 字段名 | 类型 | 说明 |
|--------|------|------|
| command_id | string | 指令唯一ID |
| device_sn | string | 设备序列号 |
| command | string | 指令类型 |
| status | string | 下发状态：pending/sent/executed/failed |
| sent_at | int64 | 下发时间戳 |
| execute_at | int64 | 执行时间戳（可选） |
| execute_code | int32 | 执行结果码（可选） |
| message | string | 状态描述 |

## 错误响应

### 400 参数错误
```json
{
  "code": 400,
  "message": "设备 SN 不能为空"
}
```

### 401 未授权
```json
{
  "code": 401,
  "message": "用户身份验证失败"
}
```

### 403 无权限
```json
{
  "code": 403,
  "message": "无权限操作该设备"
}
```

### 404 资源不存在
```json
{
  "code": 404,
  "message": "设备不存在"
}
```

### 500 系统错误
```json
{
  "code": 500,
  "message": "系统内部错误"
}
```

## 业务处理流程

### 1. 用户身份验证
- 从 JWT Token 解析用户 ID
- 校验用户账号状态正常（未禁用）

### 2. 设备权限校验
- 校验设备存在且状态正常
- 校验设备已绑定当前用户
- 校验设备未被禁用

### 3. 指令参数校验
- 校验指令类型合法
- 校验参数格式正确
- 校验参数值在有效范围内

### 4. 指令下发处理
- 生成唯一指令 ID
- 将指令存入 Redis 待下发队列
- 通过 MQTT 推送给在线设备
- 记录指令下发日志

### 5. 离线设备处理
- 设备离线时指令缓存到 Redis
- 设备上线后自动拉取未执行指令
- 指令缓存有效期 24 小时

## 安全规则

### 权限控制
- 仅设备绑定用户可下发指令
- 支持管理员权限覆盖（可选扩展）
- 指令操作全程审计记录

### 防重放攻击
- 指令 ID 唯一性保证
- 同一指令短时间内不重复下发
- 敏感指令支持二次确认

### 数据安全
- 指令参数格式严格校验
- 参数值范围限制
- 异常指令自动拦截

## 离线指令处理

### 缓存机制
- 设备离线时指令存入 Redis
- 按设备 SN 分类存储指令队列
- 限制每个设备最多保留 100 条指令

### 自动补发
- 设备上线后主动拉取未执行指令
- 按指令创建时间顺序执行
- 超时未执行指令标记为失败

### 状态同步
- 设备执行后上报结果
- 更新指令状态为已执行/失败
- 同步更新设备影子状态

## 使用示例

### 设置音量
```bash
curl -X POST \
  http://localhost:8888/api/v1/device/command \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "Content-Type: application/json" \
  -d '{
    "device_sn": "SN1234567890",
    "command": "set_volume",
    "params": {
      "volume": 80
    }
  }'
```

### 设备重启
```bash
curl -X POST \
  http://localhost:8888/api/v1/device/command \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "Content-Type: application/json" \
  -d '{
    "device_sn": "SN1234567890",
    "command": "restart"
  }'
```

### 设置运行模式
```bash
curl -X POST \
  http://localhost:8888/api/v1/device/command \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "Content-Type: application/json" \
  -d '{
    "device_sn": "SN1234567890",
    "command": "set_mode",
    "params": {
      "mode": "sleep"
    }
  }'
```

## 注意事项

1. **设备在线状态**：设备在线时指令实时下发，离线时缓存等待
2. **指令执行顺序**：按创建时间顺序执行，支持优先级扩展
3. **指令超时处理**：24 小时内未执行指令自动标记为失败
4. **状态同步延迟**：设备执行结果上报可能存在延迟
5. **网络异常处理**：MQTT 推送失败不影响指令存储，设备上线后可补发

## 扩展功能

### 指令优先级
- 支持指令优先级设置
- 高优先级指令优先执行
- 紧急指令可插队处理

### 批量指令
- 支持批量下发多个指令
- 批量指令原子性保证
- 支持批量状态查询

### 指令历史
- 提供指令执行历史查询
- 支持按时间范围筛选
- 支持按执行状态过滤