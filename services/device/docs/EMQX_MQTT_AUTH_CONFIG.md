# EMQX MQTT 设备认证接口配置指南

## 📋 概述

本文档说明如何在 **EMQX MQTT Broker** 中配置 HTTP 认证，对接设备管理微服务的认证接口，实现设备连接时的身份验证。

---

## 🔧 一、EMQX Dashboard 配置

### 1.1 开启 HTTP 认证插件

登录 EMQX Dashboard（默认地址：http://localhost:18083）：

```
用户名: admin
密码: public
```

**操作步骤**：

1. 进入 **Extensions（扩展）** → **Authentication（认证）**
2. 点击 **Create（创建）** 选择 **HTTP Server**
3. 填写以下配置信息：

#### 认证接口配置

| 配置项 | 值 | 说明 |
|--------|-----|------|
| **Method** | `post` | HTTP 请求方法 |
| **URL** | `http://localhost:8002/mqtt/auth` | 设备管理微服务认证接口 |
| **Headers** | `{"Content-Type": "application/json"}` | 请求头 |
| **Body** | `{"clientid": "${clientid}", "username": "${username}", "password": "${password}"}` | 请求体模板 |

#### 连接请求参数映射

EMQX 会自动替换以下变量：

- `${clientid}` - MQTT ClientID（设备SN）
- `${username}` - 用户名（设备SN）
- `${password}` - 密码（设备密钥）

### 1.2 配置断开连接 Webhook（可选）

如果需要自动更新设备离线状态：

1. 进入 **Extensions（扩展）** → **Webhooks**
2. 点击 **Create（创建）**
3. 选择事件类型：**Client Disconnected**
4. 配置目标 URL：`http://localhost:8002/mqtt/disconnect`
5. 消息模板：
```json
{
  "clientid": "${clientid}",
  "reason": "${reason}"
}
```

---

## 📝 二、EMQX 配置文件方式（emqx.conf）

如果使用配置文件方式，在 `emqx.conf` 中添加：

```hocon
## ============================================================
## HTTP Authentication
## ============================================================

authentication {
  # 启用 HTTP 认证
  enable = true

  # 认证后端类型
  backend = "http"

  # HTTP 配置
  http {
    # 请求方法
    method = post

    # 认证接口地址（设备管理微服务）
    url = "http://localhost:8002/mqtt/auth"

    # 请求头
    headers {
      "Content-Type" = "application/json"
    }

    # 请求体模板（EMQX 变量替换）
    body = "{\"clientid\":\"${clientid}\",\"username\":\"${username}\",\"password\":\"${password}\"}"

    # 超时时间（毫秒）
    connect_timeout = "5s"

    # 连接池大小
    pool_size = 32
  }
}

## ============================================================
## Webhook - Client Disconnected Event（可选）
## ============================================================

webhook {
  url = "http://localhost:8002/mqtt/disconnect"
  headers {
    "Content-Type" = "application/json"
  }
  body = "{\"clientid\":\"${clientid}\",\"reason\":\"${reason}\"}
  pool_size = 32
}

rule {
  event {
    client_disconnected = true
  }
  action = ["webhook:1"]
}
```

---

## 🔌 三、设备端连接参数

### 3.1 MQTT 连接参数要求

设备固件必须固化以下参数：

```c
// MQTT 连接参数（示例代码）
const char* mqtt_broker   = "tcp://your-emqx-server:1883";
const char* mqtt_client_id = "ABCDEF1234567890";  // ⭐ 设备SN（16位字母数字）
const char* mqtt_username  = "ABCDEF1234567890";  // ⭐ 设备SN（与clientid一致）
const char* mqtt_password  = "device_secret_xxx"; // ⭐ 设备专属密钥（注册时生成）
int         mqtt_port      = 1883;
int         mqtt_qos        = 1;
```

### 3.2 参数规范

| 参数 | 值 | 来源 | 是否可修改 |
|------|-----|------|-----------|
| ClientID | 设备SN（16位） | 设备注册时分配 | ❌ 固化不可改 |
| Username | 设备SN（16位） | 与ClientID一致 | ❌ 固化不可改 |
| Password | 设备密钥 | 注册时由后台生成 | ❌ 固化不可改 |

### 3.3 连接流程伪代码

```python
import paho.mqtt.client as mqtt

# 设备固件参数（预置）
DEVICE_SN = "ABCDEF1234567890"       # 设备序列号
DEVICE_SECRET = "hashed_secret_xxx"  # 设备密钥
MQTT_BROKER = "tcp://emqx-server:1883"

def on_connect(client, userdata, flags, rc):
    if rc == 0:
        print("✅ 设备认证成功，已连接到MQTT Broker")
        # 订阅专属主题
        client.subscribe(f"device/{DEVICE_SN}/command")
        client.subscribe(f"device/{DEVICE_SN}/ota")
    else:
        print(f"❌ 认证失败，错误码: {rc}")

# 创建MQTT客户端
client = mqtt.Client(client_id=DEVICE_SN)

# 设置连接参数
client.username_pw_set(DEVICE_SN, DEVICE_SECRET)

# 绑定回调函数
client.on_connect = on_connect

# 发起连接（Broker会调用HTTP认证接口验证）
try:
    client.connect(MQTT_BROKER, 1883, 60)
    client.loop_start()
except Exception as e:
    print(f"❌ 连接失败: {e}")
    # 重连逻辑...
```

---

## ✅ 四、认证响应格式说明

### 4.1 成功响应

**HTTP 状态码**: `200 OK`

```json
{
  "result": "allow"
}
```

**EMQX 行为**：放行设备连接，允许订阅/发布主题

### 4.2 失败响应

**HTTP 状态码**: `200 OK`（或其他状态码）

```json
{
  "result": "deny",
  "error_msg": "设备未注册"
}
```

**EMQX 行为**：拒绝连接，断开TCP连接

### 4.3 可能的错误原因

| error_msg | 触发条件 |
|-----------|---------|
| 客户端标识不能为空 | ClientID为空字符串 |
| 客户端标识与用户名不匹配 | ClientID ≠ Username |
| 设备未注册 | SN不存在于数据库 |
| 设备密钥未配置，请联系管理员 | Secret字段为空 |
| 设备密钥错误 | Password与数据库不匹配 |
| 设备已被管理员禁用 | Status=0（禁用） |
| 设备已报废注销 | Status=2（报废） |
| 设备状态异常 | Status值非法 |

---

## 📊 五、认证日志查询

### 5.1 查询所有认证记录

```sql
-- 查询最近100条认证日志
SELECT 
    id,
    device_sn,
    auth_result,
    error_msg,
    client_ip,
    created_at
FROM device_auth_log 
ORDER BY created_at DESC 
LIMIT 100;
```

### 5.2 查询指定设备的认证历史

```sql
-- 查询设备 ABCDEF1234567890 的最近20次认证记录
SELECT 
    id,
    auth_result,
    error_msg,
    created_at
FROM device_auth_log 
WHERE device_sn = 'ABCDEF1234567890'
ORDER BY created_at DESC 
LIMIT 20;
```

### 5.3 统计认证成功率

```sql
-- 统计最近7天的认证成功率
SELECT 
    auth_result,
    COUNT(*) as count,
    ROUND(COUNT(*) * 100.0 / SUM(COUNT(*)) OVER(), 2) as percentage
FROM device_auth_log 
WHERE created_at >= NOW() - INTERVAL '7 days'
GROUP BY auth_result;
```

### 5.4 查询频繁失败的设备

```sql
-- 查询最近24小时内失败次数超过10次的设备
SELECT 
    device_sn,
    COUNT(*) as failure_count,
    MAX(error_msg) as last_error
FROM device_auth_log 
WHERE auth_result = 'failed'
  AND created_at >= NOW() - INTERVAL '24 hours'
GROUP BY device_sn
HAVING COUNT(*) > 10
ORDER BY failure_count DESC;
```

---

## 🛡️ 六、安全约束规则

### 6.1 设备准入控制

✅ **已实现的安全机制**：

1. **唯一性校验**
   - 每台设备仅能使用自身 SN 作为标识
   - ClientID 必须等于 Username（防冒用）

2. **密钥验证**
   - 使用 bcrypt 加密存储密钥
   - 密码比对采用安全哈希算法

3. **状态管控**
   - 禁用设备直接拦截（Status=0）
   - 报废设备永远无法接入（Status=2）
   - 仅允许正常状态的设备接入（Status=1）

4. **匿名访问禁止**
   - 所有设备必须通过后端认证
   - 不支持匿名接入模式
   - 无任何白名单或免认证通道

### 6.2 防攻击措施

- ❌ **无暴力破解防护**（建议后续添加）：
  - 可增加连续失败锁定机制
  - IP限流或设备级限流

---

## 🧪 七、测试方法

### 7.1 测试工具（curl）

#### 测试认证成功场景

```bash
# 假设设备 SN=TEST000000000001, 密钥=test_secret_123
curl -X POST http://localhost:8002/mqtt/auth \
  -H "Content-Type: application/json" \
  -d '{
    "clientid": "TEST000000000001",
    "username": "TEST000000000001",
    "password": "test_secret_123"
  }'
```

**预期响应**：
```json
{"result":"allow"}
```

#### 测试认证失败场景（设备不存在）

```bash
curl -X POST http://localhost:8002/mqtt/auth \
  -H "Content-Type: application/json" \
  -d '{
    "clientid": "NONEXISTENT",
    "username": "NONEXISTENT",
    "password": "wrong_password"
  }'
```

**预期响应**：
```json
{
  "result": "deny",
  "error_msg": "设备未注册"
}
```

#### 测试断开连接接口

```bash
curl -X POST http://localhost:8002/mqtt/disconnect \
  -H "Content-Type: application/json" \
  -d '{
    "clientid": "TEST000000000001",
    "reason": "normal"
  }'
```

**预期响应**：
```json
{"result":"ok"}
```

### 7.2 使用 MQTT 客户端工具测试

#### Mosquitto 客户端

```bash
# 订阅主题测试（需要先在数据库中录入设备）
mosquitto_sub -h localhost -p 1883 \
  -i "TEST000000000001" \
  -u "TEST000000000001" \
  -P "test_secret_123" \
  -t "device/TEST000000000001/command" \
  -v
```

#### MQTT.fx / MQTT Explorer

1. 配置连接参数：
   - Broker: `localhost:1883`
   - Client ID: `TEST000000000001`
   - Username: `TEST000000000001`
   - Password: `test_secret_123`

2. 点击 Connect
3. 观察是否连接成功

---

## 📚 八、相关文档

- [设备注册流程](./docs/device_auth_api.md)
- [设备影子API](./docs/device_shadow_api_v2.md)
- [EMQX官方文档](https://www.emqx.io/docs/en/latest/)
- [EMQX HTTP认证文档](https://www.emqx.io/docs/en/latest/authentication/http-auth.html)

---

## 🔗 九、接口清单

| 接口路径 | 方法 | 用途 | 调用方 |
|----------|------|------|--------|
| `/mqtt/auth` | POST | 设备MQTT连接认证 | EMQX Broker |
| `/mqtt/disconnect` | POST | 设备断开连接处理 | EMQX Webhook |

---

## 🐛 十、常见问题排查

### Q1: 设备无法连接，提示"设备未注册"
**原因**：设备SN未录入数据库  
**解决**：调用 `/api/device/register` 接口注册设备

### Q2: 提示"设备密钥错误"
**原因**：Password参数与数据库不匹配  
**解决**：检查设备固件中硬编码的密钥是否正确

### Q3: 提示"设备已被禁用"
**原因**：设备状态被设置为禁用  
**解决**：联系管理员启用设备，或检查业务逻辑

### Q4: EMQX 无法调用认证接口
**原因**：网络不通或防火墙拦截  
**解决**：
1. 检查设备管理微服务是否启动（端口8002）
2. 检查 EMQX 到 device-api 的网络连通性
3. 查看 EMQX 日志确认HTTP请求是否发出

### Q5: 认证接口响应慢
**原因**：数据库查询慢或网络延迟  
**解决**：
1. 检查数据库索引是否存在（SN字段应有索引）
2. 检查数据库连接池配置
3. 增加 EMQX HTTP超时时间（默认5秒）

---

**📅 文档版本**: v1.0  
**📅 更新日期**: 2026-05-12  
**👤 维护者**: Device Service Team
