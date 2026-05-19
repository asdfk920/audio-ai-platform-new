# 设备 WebSocket 接入与认证

## 连接

- **URL**：由配置决定，默认 `GET ws://<host>:8002/ws/device`
- **握手**：须在 HTTP 升级为 WebSocket 的请求上携带设备 JWT（任选其一）
  - 请求头：`Authorization: Bearer <token>`
  - 或查询参数：`?token=<token>` / `?access_token=<token>`（部分客户端无法自定义握手头时使用）

`token` 来自 **`POST /api/device/register`** 响应。

## 首包认证消息（连接成功后立即发送，默认 10 秒内）

JSON 字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| `type` | string | 固定 `auth` |
| `sn` | string | 设备序列号，须与 JWT 内 `sn` 一致（忽略大小写，建议大写） |
| `timestamp` | number | **Unix 毫秒**时间戳 |
| `signature` | string | `hex(HMAC-SHA256(device_secret, signData))`，`signData = sn + timestamp`（字符串拼接，无分隔符） |

### 签名与时间戳规则

1. **`device_secret` 在库中为明文**  
   - `timestamp` 使用设备当前 UTC 毫秒时间。  
   - 服务端校验时钟偏差（默认 ±300 秒）。  
   - 服务端用库中 **明文** `device_secret` 重算 HMAC，与 `signature` 比对。

2. **`device_secret` 在库中为 bcrypt 哈希**  
   - 服务端无法用哈希作为 HMAC 密钥重算。  
   - 此时 `timestamp` **必须等于** 注册接口返回的 **`register_timestamp`（毫秒）**。  
   - `signature` **必须与** 注册接口返回的 **`signature`**（或与库存 `register_signature`）一致。

## 云端校验顺序（概要）

1. 读取握手 Token，读取首条 JSON。  
2. 校验 JWT（未过期、`device_id` / `sn` 与库一致）。  
3. 校验消息 `sn` 与 Token 内 `sn` 一致。  
4. 按 `device_id` 加载设备行，校验 **时间戳 + 签名**（规则见上）。  
5. **成功**：返回 `auth_response`（`success: true`），连接保留，设备进入在线列表与后续业务收发。  
6. **失败**：返回 `auth_response`（`success: false`），**立即断开**连接。

## 响应示例

成功：

```json
{
  "type": "auth_response",
  "success": true,
  "message": "认证成功，设备已上线",
  "device_id": 106,
  "expires_in": 86400
}
```

失败：

```json
{
  "type": "auth_response",
  "success": false,
  "message": "…原因…",
  "device_id": 106,
  "expires_in": 0
}
```

## 辅助脚本

- `generate_ws_auth.go` / `generate_ws_auth_auto.ps1`：调用注册接口并生成符合当前协议的 WS 首包 JSON（明文密钥场景默认使用当前毫秒时间戳）。
