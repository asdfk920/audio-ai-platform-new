# 设备认证功能实现总结

## 📋 实现内容

### 1. DTO 定义
- **文件**: `internal/types/device_auth_types.go`
- **内容**:
  - `DeviceAuthReq`: 设备认证请求结构（sn, secret, version, ip）
  - `DeviceAuthResp`: 认证成功响应（token, expire, sn）
  - `DeviceAuthFailResp`: 认证失败响应（code, message）

### 2. 业务逻辑层
- **文件**: `internal/logic/deviceauthlogic.go`
- **核心功能**:
  - 参数校验（SN 长度 8-64，Secret 长度 16-64）
  - 设备锁定检查（5 次失败锁定 15 分钟）
  - 设备查询（根据 SN 查询设备）
  - 密钥验证（支持明文和 SHA-256 哈希比对）
  - 设备状态检查（status=1 为正常）
  - Token 生成（格式：device_{deviceID}_{timestamp}_{hash}）
  - 在线状态更新（更新 online_status=1, last_active_at, ip）
  - 失败记录管理（记录失败原因，成功后清除）

### 3. 数据访问层
- **文件**: `internal/repo/device_repo.go`
- **新增**:
  - `DeviceRow`: 设备表行模型（含 Secret 字段）
  - `GetDeviceBySN`: 根据 SN 查询设备（含密钥）

### 4. API 处理器
- **文件**: `internal/handler/deviceauthhandler.go`
- **功能**:
  - 解析 HTTP 请求
  - 参数校验
  - 调用认证逻辑
  - 返回 JSON 响应
  - 错误处理（401 未授权）

### 5. 路由注册
- **文件**: `internal/handler/routes.go`
- **路由**: `POST /api/v1/device/auth`
- **特点**: 无需 JWT 认证（设备首次接入）

### 6. 服务上下文
- **文件**: `device.go`
- **修改**: 初始化时调用 `handler.SetServiceContext(ctx)`

### 7. 数据库迁移
- **文件**: `internal/repo/migrations/device_auth.sql`
- **表**: `device_auth_failures`（认证失败记录表）
- **索引**: `idx_auth_failures_sn_time`（加速查询）

### 8. API 文档
- **文件**: `docs/device_auth_api.md`
- **内容**: 完整的接口文档、调用示例、错误码说明

---

## 🔐 安全特性

### 1. 防暴力破解
- 连续失败 5 次自动锁定
- 锁定时长 15 分钟
- 认证成功后自动清除失败记录

### 2. Token 安全
- 有效期 24 小时
- 基于设备 ID、时间戳、哈希生成
- 支持过期自动失效

### 3. 密钥保护
- 支持 SHA-256 加密存储
- 传输过程使用 HTTPS
- 密钥长度要求 16-64 位

### 4. 审计日志
- 所有认证请求记录日志
- 失败原因详细记录（device_not_found/secret_mismatch/device_disabled）
- 便于问题排查和安全审计

---

## 📊 数据库表结构

### device_auth_failures
```sql
CREATE TABLE device_auth_failures (
    id BIGSERIAL PRIMARY KEY,
    sn VARCHAR(64) NOT NULL,
    reason VARCHAR(50) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_auth_failures_sn_time 
ON device_auth_failures(sn, created_at);
```

### device（需要包含的字段）
```sql
- id: 设备 ID
- sn: 设备序列号（唯一）
- secret: 设备密钥（建议 SHA-256 加密存储）
- status: 设备状态（1=正常，2=禁用，3=未激活）
- online_status: 在线状态（0=离线，1=在线）
- last_active_at: 最后活跃时间
- ip: 设备 IP 地址
```

---

## 🚀 使用方式

### 1. 执行数据库迁移
```bash
psql -h localhost -p 5432 -U admin -d audio_platform -f internal/repo/migrations/device_auth.sql
```

### 2. 启动设备服务
```bash
cd services/device
go build -o device.exe device.go
./device.exe -f etc/device.yaml
```

### 3. 调用认证接口
```bash
curl -X POST http://localhost:8888/api/v1/device/auth \
  -H "Content-Type: application/json" \
  -d '{
    "sn": "SN1234567890",
    "secret": "your_device_secret_123456",
    "version": "1.0.0",
    "ip": "192.168.1.100"
  }'
```

### 4. 成功响应
```json
{
  "code": 200,
  "data": {
    "token": "device_123_1712563200_a1b2c3d4e5f6...",
    "expire": 86400,
    "sn": "SN1234567890"
  },
  "msg": "success"
}
```

---

## ⚠️ 注意事项

### 1. 设备 SN 标准化
- 系统会自动将 SN 转为大写并去除首尾空格
- 数据库存储建议使用大写格式

### 2. Secret 存储方式
- **明文存储**: 直接比对字符串
- **哈希存储**: 使用 SHA-256 哈希后比对
- 系统会自动检测并适配两种格式

### 3. 并发控制
- 同一设备并发认证会串行处理
- 使用数据库事务保证一致性

### 4. 监控告警
建议配置以下监控指标：
- 认证成功率
- 认证失败次数（按设备 SN）
- 设备锁定告警
- Token 刷新频率

---

## 📝 后续优化建议

1. **Token 存储**: 使用 Redis 存储 Token，支持主动失效
2. **多设备互斥**: 同一 SN 登录时强制下线旧设备
3. **地域限制**: 根据 IP 限制设备登录地域
4. **时间窗口**: 限制设备仅在特定时间段可接入
5. **双因素认证**: 重要设备增加二次认证
6. **审计报表**: 定期生成设备认证审计报告

---

## 🎯 错误码说明

| 错误信息 | HTTP 状态码 | 说明 |
|---------|-----------|------|
| sn 不能为空 | 400 | 设备序列号缺失 |
| sn 长度必须为 8-64 位 | 400 | SN 格式错误 |
| secret 不能为空 | 400 | 设备密钥缺失 |
| secret 长度必须为 16-64 位 | 400 | Secret 格式错误 |
| 设备不存在 | 401 | SN 对应的设备不存在 |
| 设备密钥错误 | 401 | Secret 不匹配 |
| 设备已禁用 | 401 | 设备状态为禁用 |
| 设备认证失败次数过多，已锁定 | 401 | 触发防暴力破解锁定 |

---

## 📚 相关文件清单

```
services/device/
├── internal/
│   ├── types/
│   │   └── device_auth_types.go          # DTO 定义
│   ├── logic/
│   │   └── deviceauthlogic.go            # 业务逻辑
│   ├── handler/
│   │   ├── deviceauthhandler.go          # API 处理器
│   │   └── routes.go                     # 路由注册
│   ├── repo/
│   │   ├── device_repo.go                # 数据访问
│   │   └── migrations/
│   │       └── device_auth.sql           # 数据库迁移
│   └── svc/
│       └── service_context.go            # 服务上下文
├── docs/
│   └── device_auth_api.md                # API 文档
└── device.go                             # 主程序入口
```

---

## ✅ 编译测试

```bash
cd services/device
go build -o device.exe device.go
# 编译成功，无错误
```

---

**实现完成时间**: 2026-04-08  
**版本**: v1.0.0  
**状态**: ✅ 已完成并编译通过
