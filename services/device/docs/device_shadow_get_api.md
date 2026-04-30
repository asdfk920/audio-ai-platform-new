# 设备影子查询接口文档

## 接口说明

- **接口地址**: `/api/v1/device/shadow`
- **请求方式**: `GET`
- **功能**: 用户获取已绑定设备的实时影子数据（在线状态、电量、运行参数等）
- **权限要求**: 需要 JWT 登录认证，仅能查询自己绑定的设备
- **使用场景**: App 端实时查看设备状态、设备控制面板、状态监控

---

## 请求参数

### Header
```
Authorization: Bearer <access_token>
```

### Query Parameters
```
GET /api/v1/device/shadow?device_sn=SN1234567890
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| device_sn | string | 是 | 设备序列号（最小 8 字符，最大 64 字符） |

---

## 后端处理全流程

### 1. 用户请求获取设备影子
- 用户端发起请求，携带设备 SN
- 后端从 Token 解析出 user_id

### 2. 校验用户合法性
- 查询用户信息，校验用户存在
- 校验用户账号状态正常（status=1）
- 用户不存在或异常则拒绝

### 3. 校验设备合法性
- 根据 device_sn 查询设备表
- 校验设备存在
- 校验设备状态正常（未被禁用、报废）

### 4. 校验设备绑定关系
- 查询 `user_device_bind` 表
- 校验该设备当前确实绑定给当前用户
- 无权限则直接拒绝（403）

### 5. 查询 Redis 设备影子
- Redis Key 格式：`device:shadow:{sn}`
- 查询 Hash 结构存储的设备影子数据
- 若 Redis 存在数据，直接取出

### 6. Redis 未命中则查数据库
- 若 Redis 无数据，从设备表读取基础信息
- 作为兜底数据返回

### 7. 组装设备影子数据
- 在线状态（online）
- 运行状态（run_state: playing/paused/stopped）
- 电量（battery: 0-100）
- 音量（volume: 0-100）
- 固件版本（firmware_version）
- 设备 IP（ip）
- 最后更新时间（last_update_at）
- 最后活跃时间（last_active_at）
- 工作模式（mode）

### 8. 返回设备影子完整信息
- 返回组装好的影子数据
- 包含 Redis 实时数据 + 数据库兜底数据

### 9. 记录查询日志
- 记录用户查询设备影子的操作日志
- 用于审计和性能分析

---

## 返回结果

### 成功响应（200）
```json
{
  "code": 0,
  "msg": "获取成功",
  "data": {
    "shadow": {
      "sn": "SN1234567890",
      "device_id": 123,
      "product_key": "X1 Pro",
      "online": true,
      "run_state": "playing",
      "firmware_version": "FW_1.2.3",
      "battery": 85,
      "volume": 60,
      "ip": "192.168.1.100",
      "last_update_at": 1775692800,
      "last_active_at": "2026-04-08 18:20:10",
      "mode": "normal"
    }
  }
}
```

### 无权限访问（403）
```json
{
  "code": 1007,
  "msg": "无权访问该设备"
}
```

### 设备不存在（404）
```json
{
  "code": 1006,
  "msg": "设备不存在"
}
```

### Token 无效（401）
```json
{
  "code": 1004,
  "msg": "登录已过期或无效，请重新登录"
}
```

### 参数错误（400）
```json
{
  "code": 1008,
  "msg": "设备 SN 不能为空"
}
```

---

## 设备影子字段说明

### DeviceShadowItem 字段说明

| 字段 | 类型 | 说明 | 来源 |
|------|------|------|------|
| sn | string | 设备序列号 | 数据库 |
| device_id | int64 | 设备 ID | 数据库 |
| product_key | string | 产品型号 | 数据库 |
| online | bool | **在线状态**（true=在线，false=离线） | Redis > 数据库 |
| run_state | string | **运行状态**（playing/paused/stopped） | Redis |
| firmware_version | string | 固件版本 | 数据库 |
| battery | int32 | **电量**（0-100，0=没电，100=满电） | Redis |
| volume | int32 | **音量**（0-100，0=静音，100=最大） | Redis |
| ip | string | 设备 IP 地址 | Redis > 数据库 |
| last_update_at | int64 | **最后更新时间戳**（Unix 时间戳） | Redis |
| last_active_at | string | 最后活跃时间（格式化时间） | 数据库 |
| mode | string | 工作模式（normal/eco/performance） | 默认值 |

---

## 数据来源优先级

### 实时数据（Redis 优先）
- **在线状态**: Redis > 数据库（online_status）
- **运行状态**: 仅 Redis（设备上报）
- **电量**: 仅 Redis（设备上报）
- **音量**: 仅 Redis（设备上报）
- **IP 地址**: Redis > 数据库
- **最后更新时间**: 仅 Redis

### 基础数据（数据库）
- **设备序列号**: 数据库（device 表）
- **设备 ID**: 数据库（device 表）
- **产品型号**: 数据库（device 表）
- **固件版本**: 数据库（device 表）
- **最后活跃时间**: 数据库（device.last_active_at）

---

## Redis 数据结构

### Key 格式
```
device:shadow:{sn}
```

### Hash 结构
```
HSET device:shadow:SN1234567890 online "true"
HSET device:shadow:SN1234567890 run_state "playing"
HSET device:shadow:SN1234567890 battery "85"
HSET device:shadow:SN1234567890 volume "60"
HSET device:shadow:SN1234567890 ip "192.168.1.100"
HSET device:shadow:SN1234567890 last_update_at "1775692800"
```

### 过期策略
- 设备影子数据自动续期（设备心跳/状态上报时续期）
- 过期时间：30 分钟（可配置）
- 过期后自动删除，下次查询从数据库加载兜底数据

---

## 数据库查询

### 1. 查询设备基础信息
```sql
SELECT id, sn, product_key, mac, firmware_version, hardware_version, 
       ip, status, online_status, secret, last_active_at
FROM device
WHERE sn = $1
LIMIT 1;
```

### 2. 校验设备绑定关系
```sql
SELECT 1 
FROM user_device_bind 
WHERE device_id = $1 AND user_id = $2 AND status = 1 
LIMIT 1;
```

---

## 业务联动

### App 端设备控制面板
```javascript
async getDeviceShadow(deviceSn) {
  const res = await fetch(`/api/v1/device/shadow?device_sn=${deviceSn}`, {
    headers: {
      'Authorization': `Bearer ${accessToken}`
    }
  });
  
  const result = await res.json();
  if (result.code === 0) {
    const shadow = result.data.shadow;
    
    // 更新 UI
    this.deviceOnline = shadow.online;
    this.deviceBattery = shadow.battery;
    this.deviceVolume = shadow.volume;
    this.deviceRunState = shadow.run_state;
    
    // 根据状态显示不同图标
    if (shadow.online && shadow.run_state === 'playing') {
      this.showPlayingIcon();
    } else if (!shadow.online) {
      this.showOfflineIcon();
    }
  }
}
```

### 设备状态监控
```javascript
// 轮询设备状态（每 5 秒）
setInterval(async () => {
  const shadow = await getDeviceShadow('SN1234567890');
  
  // 电量低告警
  if (shadow.battery < 20) {
    showLowBatteryAlert(shadow.battery);
  }
  
  // 离线告警
  if (!shadow.online) {
    showDeviceOfflineAlert();
  }
}, 5000);
```

---

## 调用示例

### cURL
```bash
curl -X GET "http://localhost:8888/api/v1/device/shadow?device_sn=SN1234567890" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

### JavaScript
```javascript
async function getDeviceShadow(deviceSn) {
  const res = await fetch(`/api/v1/device/shadow?device_sn=${deviceSn}`, {
    method: 'GET',
    headers: {
      'Authorization': `Bearer ${accessToken}`,
      'Content-Type': 'application/json'
    }
  });
  
  const result = await res.json();
  if (result.code === 0) {
    console.log('设备影子', result.data.shadow);
    return result.data.shadow;
  }
  return null;
}

// 使用示例
const shadow = await getDeviceShadow('SN1234567890');
if (shadow.online) {
  console.log('设备在线，电量：', shadow.battery);
} else {
  console.log('设备离线');
}
```

### Go 微服务调用
```go
// 在服务间调用
type DeviceShadowService struct {
    httpClient *http.Client
}

func (s *DeviceShadowService) GetShadow(ctx context.Context, deviceSn string) (*types.DeviceShadowItem, error) {
    req := &types.GetDeviceShadowReq{
        DeviceSn: deviceSn,
    }
    
    // 调用设备微服务
    resp, err := s.deviceClient.GetDeviceShadow(ctx, req)
    if err != nil {
        return nil, err
    }
    
    return &resp.Shadow, nil
}
```

---

## 实现文件清单

```
services/device/
├── internal/
│   ├── types/
│   │   └── device_types.go                   # DTO 定义（已更新）
│   ├── logic/
│   │   └── get_device_shadow_logic.go        # 业务逻辑层（新建）
│   ├── handler/
│   │   └── get_device_shadow_handler.go      # API 处理器（新建）
│   └── handler/
│       └── routes.go                         # 路由注册（已更新）
└── docs/
    └── device_shadow_get_api.md              # API 文档（新建）
```

---

## 注意事项

1. **权限隔离**: 用户只能查询自己绑定的设备影子，严禁越权访问
2. **Redis 高并发**: Redis 支撑高频查询，数据库仅作为兜底
3. **数据一致性**: Redis 与数据库数据可能短暂不一致，以 Redis 为准
4. **影子续期**: 设备心跳/状态上报时自动续期影子数据
5. **错误处理**: Redis 查询失败不影响返回，使用数据库兜底数据
6. **日志审计**: 记录所有查询日志，用于溯源和性能分析

---

## 性能优化建议

### 1. Redis 缓存策略
```go
// Redis Hash 存储，支持字段级更新
redis.HSet(ctx, "device:shadow:SN123", "battery", "85")
redis.HSet(ctx, "device:shadow:SN123", "online", "true")

// 批量查询，减少网络往返
redis.HGetAll(ctx, "device:shadow:SN123").Result()
```

### 2. 数据库查询优化
```sql
-- 添加索引加速查询
CREATE INDEX idx_device_sn ON device(sn);
CREATE INDEX idx_bind_user_device ON user_device_bind(user_id, device_id, status);
```

### 3. 连接池配置
```yaml
# Redis 连接池
redis:
  max_idle: 10
  max_active: 100
  idle_timeout: 300s

# 数据库连接池
db:
  max_open_conns: 100
  max_idle_conns: 10
  conn_max_lifetime: 1h
```

---

## 监控与统计

### 1. 关键指标
- 设备影子查询 QPS
- Redis 命中率
- 平均响应时间
- 权限校验失败率

### 2. 日志记录
```go
l.Logger.Infof("设备影子查询：user_id=%d, device_sn=%s, online=%v, battery=%d",
    userId, req.DeviceSn, shadow.Online, shadow.Battery)
```

### 3. 告警规则
- Redis 不可用触发告警
- 数据库查询失败率 > 5% 触发告警
- 响应时间 > 200ms 触发告警

---

## 与其他接口的关系

| 接口 | 功能 | 调用时机 |
|------|------|----------|
| `/device/shadow/report` | **设备上报影子** | 设备定时上报状态 |
| `/device/shadow` | **查询设备影子** | **用户查看设备状态** |
| `/device/status` | 查询设备状态 | 管理端查看设备 |
| `/device/heartbeat` | 设备心跳 | 设备定时心跳 |

---

## 错误码说明

| 错误码 | 说明 | 解决方案 |
|--------|------|----------|
| 0 | 获取成功 | - |
| 1004 | Token 无效或过期 | 重新登录获取新 token |
| 1006 | 设备不存在 | 检查设备 SN 是否正确 |
| 1007 | 无权访问该设备 | 确认设备已绑定给当前用户 |
| 1008 | 参数错误 | 检查 device_sn 参数格式 |

---

## 安全规则

1. **JWT 认证**: 必须携带有效 Token
2. **权限隔离**: 仅能查询自己绑定的设备
3. **参数校验**: 严格校验 device_sn 格式和长度
4. **限流保护**: 单用户高频查询触发限流
5. **HTTPS 传输**: 生产环境强制使用 HTTPS
6. **日志审计**: 所有查询操作全程留痕

---

**版本**: v1.0.0  
**更新时间**: 2026-04-08  
**状态**: ✅ 已完成并编译通过
