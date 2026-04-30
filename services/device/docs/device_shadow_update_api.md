# 设备影子更新接口文档

## 接口说明

- **接口地址**: `/api/v1/device/shadow/update`
- **请求方式**: `POST`
- **功能**: 设备端上报最新状态数据，更新 Redis 设备影子并持久化到数据库
- **权限要求**: 需要设备认证 Token（设备身份认证后颁发）
- **使用场景**: 设备定时上报状态、状态变更通知、心跳保活

---

## 请求参数

### Header
```
Authorization: Bearer <device_access_token>
Content-Type: application/json
```

### Body
```json
{
  "sn": "SN1234567890",
  "online": true,
  "run_state": "playing",
  "firmware_version": "FW_1.2.3",
  "battery": 85,
  "volume": 60,
  "mode": "normal",
  "error_code": "",
  "ip": "192.168.1.100"
}
```

### 参数说明
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| sn | string | 是 | 设备序列号（8-64 字符） |
| online | bool | 否 | 在线状态（true=在线，false=离线） |
| run_state | string | 否 | 运行状态（playing/paused/stopped/normal/fault/sleep/upgrading/charging） |
| firmware_version | string | 否 | 固件版本（最大 32 字符） |
| battery | int32 | 否 | 电量（0-100） |
| volume | int32 | 否 | 音量（0-100） |
| mode | string | 否 | 工作模式（normal/eco/performance） |
| error_code | string | 否 | 故障码（最大 64 字符） |
| ip | string | 否 | 设备 IP 地址（最大 45 字符） |

### 增量更新说明
- **支持增量上报**：只需上报变化的字段，不需要全量上报
- **字段覆盖策略**：上报的字段会覆盖 Redis 中对应字段，未上报的字段保持不变
- **示例**：仅上报电量变化
```json
{
  "sn": "SN1234567890",
  "battery": 75
}
```

---

## 后端处理全流程

### 1. 设备端上报状态
- 设备通过 MQTT 或 HTTP 向服务端上报最新状态数据
- 携带 device_sn、设备凭证、状态参数
- 设备已认证，携带合法 Token

### 2. 后端接收数据，校验设备身份
- 从 Token 解析设备 ID
- 校验 device_sn 是否存在
- 校验设备是否合法
- 认证是否有效

### 3. 校验设备状态
- 查询设备表，校验设备状态
- 设备已禁用则拒绝更新

### 4. 解析设备上报的状态字段
- 在线状态（online）
- 电量（battery）
- 固件版本（firmware_version）
- 运行模式（run_state）
- 音量（volume）
- 故障码（error_code）
- 最后上报时间（自动添加）

### 5. 更新 Redis 设备影子
- 将最新状态合并写入 Redis 对应的设备影子结构
- 覆盖旧数据，保证存储最新快照
- 设置过期时间（30 分钟）
- 更新在线设备集合

### 6. 异步持久化到数据库
- 根据策略异步将设备影子数据持久化到数据库
- 用于历史查询与故障恢复
- 不阻塞设备上报流程

### 7. 更新设备在线状态与最后活跃时间
- 更新 device 表的 online_status
- 更新 last_active_at
- 异步执行，不阻塞主流程

### 8. 记录设备状态上报日志
- 插入 device_status_log 表
- 记录运行状态、电量、音量、故障码
- 用于历史追溯和数据分析

### 9. 返回更新成功应答
- 返回更新后的影子摘要
- 包含更新时间、在线状态、电量等

---

## 返回结果

### 成功响应（200）
```json
{
  "code": 0,
  "msg": "更新成功",
  "data": {
    "sn": "SN1234567890",
    "device_id": 123,
    "updated_at": 1775692800,
    "online": true,
    "run_state": "playing",
    "battery": 85,
    "volume": 60,
    "last_update_at": 1775692800
  }
}
```

### 设备不存在（404）
```json
{
  "code": 1006,
  "msg": "设备不存在"
}
```

### 设备已禁用（403）
```json
{
  "code": 1010,
  "msg": "设备已禁用，无法上报状态"
}
```

### 参数错误（400）
```json
{
  "code": 1008,
  "msg": "设备 SN 不能为空"
}
```

### Token 无效（401）
```json
{
  "code": 1004,
  "msg": "设备认证失败"
}
```

### 运行状态非法（400）
```json
{
  "code": 1008,
  "msg": "run_state 须为 playing/paused/stopped/normal/fault/sleep/upgrading/charging"
}
```

---

## Redis 数据结构

### Key 格式
```
device:shadow:{sn}
```

### Hash 结构（更新后）
```
HSET device:shadow:SN1234567890 online "true"
HSET device:shadow:SN1234567890 run_state "playing"
HSET device:shadow:SN1234567890 battery "85"
HSET device:shadow:SN1234567890 volume "60"
HSET device:shadow:SN1234567890 mode "normal"
HSET device:shadow:SN1234567890 firmware_version "FW_1.2.3"
HSET device:shadow:SN1234567890 ip "192.168.1.100"
HSET device:shadow:SN1234567890 last_update_at "1775692800"
```

### 在线设备集合
```
SADD device:online "SN1234567890"
```

### 过期策略
- 设备影子数据：30 分钟
- 在线设备集合：30 分钟
- 设备上报时自动续期

---

## 数据库变更

### 1. 更新 device 表
```sql
UPDATE device 
SET last_active_at = NOW(),
    firmware_version = COALESCE(?, firmware_version),
    ip = COALESCE(?, ip),
    online_status = ?
WHERE id = ?;
```

### 2. 插入 device_status_log 表
```sql
INSERT INTO device_status_log (device_id, run_state, battery, volume, error_code, created_at)
VALUES (?, ?, ?, ?, ?, NOW());
```

**注意**: `device_status_log` 表需要预先创建，表结构如下：
```sql
CREATE TABLE IF NOT EXISTS device_status_log (
    id BIGSERIAL PRIMARY KEY,
    device_id BIGINT NOT NULL,
    run_state VARCHAR(32) NOT NULL,
    battery INT NOT NULL DEFAULT 0,
    volume INT NOT NULL DEFAULT 0,
    error_code VARCHAR(64),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    INDEX idx_device_id (device_id),
    INDEX idx_created_at (created_at)
);
```

---

## 运行状态枚举

| 状态值 | 说明 | 使用场景 |
|--------|------|----------|
| playing | 播放中 | 设备正在播放音频 |
| paused | 暂停 | 设备暂停播放 |
| stopped | 停止 | 设备停止播放 |
| normal | 正常 | 设备空闲待机 |
| fault | 故障 | 设备发生故障 |
| sleep | 睡眠 | 设备进入睡眠模式 |
| upgrading | 升级中 | 设备固件升级 |
| charging | 充电中 | 设备正在充电 |

---

## 业务联动

### 设备定时上报
```go
// 设备端每 30 秒上报一次状态
ticker := time.NewTicker(30 * time.Second)
for range ticker.C {
    status := getDeviceStatus()
    reportToCloud(status)
}
```

### 状态变更通知
```go
// 设备状态变更时立即上报
func onStateChanged(newStatus DeviceStatus) {
    reportToCloud(newStatus)
    
    // 本地更新 UI
    updateDisplay(newStatus)
}
```

### 离线检测
```javascript
// 服务端检测离线设备
setInterval(async () => {
    const onlineDevices = await redis.smembers('device:online');
    
    // 30 分钟未上报的设备视为离线
    for (const sn of onlineDevices) {
        const shadow = await redis.hgetall(`device:shadow:${sn}`);
        const lastUpdate = parseInt(shadow.last_update_at);
        
        if (Date.now() - lastUpdate > 30 * 60 * 1000) {
            await markDeviceOffline(sn);
        }
    }
}, 60000); // 每分钟检查一次
```

---

## 调用示例

### cURL
```bash
curl -X POST http://localhost:8888/api/v1/device/shadow/update \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "Content-Type: application/json" \
  -d '{
    "sn": "SN1234567890",
    "online": true,
    "battery": 85,
    "run_state": "playing"
  }'
```

### 设备端（C++）
```cpp
#include <curl/curl.h>

void reportDeviceStatus(const std::string& sn, int battery, const std::string& runState) {
    CURL* curl = curl_easy_init();
    if (curl) {
        std::string json = "{\"sn\":\"" + sn + "\",\"battery\":" + 
                          std::to_string(battery) + ",\"run_state\":\"" + runState + "\"}";
        
        struct curl_slist* headers = NULL;
        headers = curl_slist_append(headers, "Authorization: Bearer DEVICE_TOKEN");
        headers = curl_slist_append(headers, "Content-Type: application/json");
        
        curl_easy_setopt(curl, CURLOPT_URL, "http://cloud.api/device/shadow/update");
        curl_easy_setopt(curl, CURLOPT_POSTFIELDS, json.c_str());
        curl_easy_setopt(curl, CURLOPT_HTTPHEADER, headers);
        
        curl_easy_perform(curl);
        curl_easy_cleanup(curl);
    }
}
```

### 设备端（Python）
```python
import requests
import time

def report_device_status(sn, battery, run_state):
    url = "http://cloud.api/api/v1/device/shadow/update"
    headers = {
        "Authorization": "Bearer DEVICE_TOKEN",
        "Content-Type": "application/json"
    }
    
    data = {
        "sn": sn,
        "battery": battery,
        "run_state": run_state
    }
    
    response = requests.post(url, json=data, headers=headers)
    if response.status_code == 200:
        result = response.json()
        print(f"上报成功：{result}")
    else:
        print(f"上报失败：{response.text}")

# 定时上报
while True:
    status = get_device_status()
    report_device_status(
        sn="SN1234567890",
        battery=status.battery,
        run_state=status.run_state
    )
    time.sleep(30)  # 30 秒上报一次
```

---

## 实现文件清单

```
services/device/
├── internal/
│   ├── types/
│   │   └── device_types.go                   # DTO 定义（已更新）
│   ├── logic/
│   │   └── device_shadow_update_logic.go     # 业务逻辑层（新建）
│   ├── handler/
│   │   └── device_shadow_update_handler.go   # API 处理器（新建）
│   └── handler/
│       └── routes.go                         # 路由注册（已更新）
└── docs/
    └── device_shadow_update_api.md           # API 文档（新建）
```

---

## 注意事项

1. **设备认证**: 必须先调用 `/device/auth` 获取设备 Token
2. **增量更新**: 支持只上报变化的字段，减少网络流量
3. **异步持久化**: 数据库写入异步执行，不阻塞设备上报
4. **Redis 降级**: Redis 异常时降级写入数据库，不阻塞设备上报
5. **状态枚举**: run_state 必须是允许的状态值之一
6. **日志审计**: 所有上报记录都会写入日志表，用于追溯

---

## 性能优化建议

### 1. 批量上报
```json
{
  "sn": "SN1234567890",
  "battery": 85,
  "volume": 60,
  "run_state": "playing",
  "mode": "normal"
}
```
一次上报多个字段，减少网络请求次数

### 2. 变更上报
```python
# 仅当状态变更时上报
if current_status != last_reported_status:
    report_device_status(current_status)
    last_reported_status = current_status
```

### 3. 自适应上报频率
```python
# 根据网络状态调整上报频率
if network_quality == "good":
    report_interval = 30  # 30 秒
elif network_quality == "poor":
    report_interval = 60  # 60 秒
else:
    report_interval = 300  # 5 分钟
```

---

## 监控与统计

### 1. 关键指标
- 设备上报 QPS
- Redis 写入成功率
- 数据库持久化成功率
- 平均响应时间

### 2. 日志记录
```go
l.Logger.Infof("设备影子更新：device_id=%d, sn=%s, online=%v, battery=%d, run_state=%s",
    device.ID, req.Sn, req.Online, req.Battery, req.RunState)
```

### 3. 告警规则
- Redis 不可用触发告警
- 数据库写入失败率 > 5% 触发告警
- 响应时间 > 500ms 触发告警
- 单设备高频上报（> 10 次/分钟）触发限流

---

## 与其他接口的关系

| 接口 | 功能 | 调用时机 |
|------|------|----------|
| `/device/auth` | **设备认证** | **设备接入前获取 Token** |
| `/device/shadow/update` | **设备上报影子** | **设备定时上报状态** |
| `/device/shadow` | 查询设备影子 | 用户查看设备状态 |
| `/device/shadow/report` | 用户端上报影子 | 用户端更新影子 |
| `/device/heartbeat` | 设备心跳 | 设备定时心跳 |

---

## 错误码说明

| 错误码 | 说明 | 解决方案 |
|--------|------|----------|
| 0 | 更新成功 | - |
| 1004 | 设备认证失败 | 重新调用 /device/auth 获取 Token |
| 1006 | 设备不存在 | 检查设备 SN 是否正确 |
| 1008 | 参数错误 | 检查请求参数格式和枚举值 |
| 1010 | 设备已禁用 | 联系管理员启用设备 |
| 1011 | Redis 错误 | 检查 Redis 服务状态 |

---

## 安全规则

1. **设备认证**: 必须携带设备 Token，防止非法设备接入
2. **SN 校验**: 严格校验设备 SN 格式和长度
3. **限流保护**: 单设备高频上报触发限流（如 10 次/分钟）
4. **HTTPS 传输**: 生产环境强制使用 HTTPS
5. **数据加密**: 敏感数据加密传输
6. **日志审计**: 所有上报操作全程留痕

---

## 容灾策略

### Redis 不可用
- 降级直接写入数据库
- 记录错误日志
- 返回成功（不阻塞设备）

### 数据库不可用
- 仅写入 Redis
- 记录错误日志
- 异步重试写入数据库

### 网络异常
- 设备端缓存状态数据
- 网络恢复后批量上报
- 支持断点续传

---

**版本**: v1.0.0  
**更新时间**: 2026-04-08  
**状态**: ✅ 已完成并编译通过
