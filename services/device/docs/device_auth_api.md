# 设备认证接口文档

设备建档与激活请在 **go-admin 后台「设备管理」** 完成（添加设备、手动激活等）。**设备微服务不再提供** `POST /api/v1/device/register`。

公开设备接入链路仅为：

- `POST /api/v1/device/auth`：设备用 `sn + secret + timestamp + nonce + signature` 向云端认证，并获取短期 token

## 1. 云端认证 / 获取短期 token

- 路径：`POST /api/v1/device/auth`
- 鉴权：无
- 说明：设备使用出厂密钥完成签名认证，云端通过后签发短期 token；后续 HTTP 接口优先走 Bearer token。设备须已在后台录入且处于可联网业务状态（如「正常」）。

请求示例：

```json
{
  "sn": "SN1234567890",
  "secret": "factory_device_secret",
  "timestamp": 1776401010,
  "nonce": "auth-0001",
  "signature": "4d6ea6b7...",
  "firmware_version": "1.0.0",
  "ip": "192.168.1.100"
}
```

成功响应示例：

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "token": "...",
    "expire": 86400,
    "sn": "SN1234567890",
    "device_id": 1001,
    "server_time": 1776401010,
    "mqtt_broker": "tcp://localhost:1883",
    "http_base_url": "http://localhost:8002"
  }
}
```

## 2. 与后台能力的关系

| 能力 | 位置 |
|------|------|
| 预注册设备、查看/修改密钥 | go-admin `POST /api/v1/platform-device` 等 |
| 将「未激活」改为「正常」 | 后台设备列表「激活」或 `PUT /api/v1/platform-device/:sn/status` |

设备端拿到 token 后，再调用影子、上报等需签名的接口。
