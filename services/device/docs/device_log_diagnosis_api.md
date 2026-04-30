# 设备日志与远程诊断 API（device + admin）

本文档覆盖两条链路：

- 设备侧：日志批量上报、诊断结果回传
- 管理侧：发起远程诊断、查询设备日志、查看诊断历史

## 1. 设备批量上传日志

- 路径：`POST /api/v1/device/logs/upload`
- 鉴权：设备身份，优先 `Authorization: Bearer <device_token>`，兼容 `device_sn + device_secret`
- 说明：设备将本地缓存日志按批次上传；相同 `upload_id` 幂等

请求示例：

```json
{
  "device_sn": "SN12345678",
  "upload_id": "upl-20260417-0001",
  "trigger_type": "error",
  "source": "http",
  "report_time": 1776400000,
  "summary": {
    "buffer_size": 120,
    "reason": "error_flush"
  },
  "logs": [
    {
      "log_type": "system",
      "log_level": "info",
      "module": "boot",
      "content": "device started",
      "report_time": 1776399990,
      "report_source": "device",
      "ip_address": "192.168.1.20"
    },
    {
      "log_type": "error",
      "log_level": "error",
      "module": "mqtt",
      "content": "broker connection lost",
      "error_code": 1001,
      "extra": {
        "retry": 3
      },
      "report_time": 1776400000,
      "report_source": "device"
    }
  ]
}
```

响应示例：

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "device_sn": "SN12345678",
    "upload_id": "upl-20260417-0001",
    "accepted_count": 2,
    "duplicate": false
  }
}
```

## 2. 设备回传诊断结果

- 路径：`POST /api/v1/device/diagnosis/report`
- 鉴权：设备身份，优先 `Authorization: Bearer <device_token>`，兼容 `device_sn + device_secret`
- 说明：设备执行远程诊断后回传结果，同时会回写 `device_diagnosis` 和对应的 `device_instruction`

请求示例：

```json
{
  "device_sn": "SN12345678",
  "diagnosis_id": 9001,
  "instruction_id": 12001,
  "success": true,
  "summary": "network and system checks passed",
  "report_time": 1776400100,
  "items": [
    {
      "item": "network",
      "status": "normal",
      "message": "wifi connected",
      "detail": "rssi=-52"
    },
    {
      "item": "memory",
      "status": "normal",
      "message": "memory usage healthy"
    }
  ],
  "result": {
    "uptime_sec": 3600,
    "wifi_rssi": -52
  },
  "reported": {
    "online": true,
    "run_state": "normal"
  }
}
```

响应示例：

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "diagnosis_id": 9001,
    "instruction_id": 12001,
    "status": "completed"
  }
}
```

## 3. 管理端发起远程诊断

- 路径：`POST /api/v1/platform-device/diagnosis/start`
- 鉴权：Admin JWT + Casbin
- 说明：运维侧创建一条 `device_diagnosis`，并同时写入 `device_instruction`，设备在线时走 MQTT 推送，离线时走 pending 补偿

请求示例：

```json
{
  "device_id": 1001,
  "diag_type": "network"
}
```

响应示例：

```json
{
  "code": 200,
  "msg": "诊断已开始",
  "data": {
    "diagnosis_id": 9001,
    "instruction_id": 12001,
    "device_id": 1001,
    "sn": "SN12345678",
    "diag_type": "network",
    "status": "diagnosing",
    "message": "诊断指令已入队，设备将执行自检"
  }
}
```

## 4. 管理端查询设备日志

- 路径：`GET /api/v1/platform-device/log/list`
- 鉴权：Admin JWT + Casbin
- 说明：支持按设备、时间、类型、级别分页查询

请求示例：

```http
GET /api/v1/platform-device/log/list?sn=SN12345678&log_level=error&start_time=2026-04-17T00:00:00Z&end_time=2026-04-17T23:59:59Z&page=1&page_size=20
Authorization: Bearer <admin_token>
```

## 5. 管理端查询诊断结果 / 历史

- `GET /api/v1/platform-device/diagnosis/result?diagnosis_id=9001`
- `GET /api/v1/platform-device/diagnosis/history?sn=SN12345678&page=1&page_size=20`

## 6. 推荐联调顺序

1. 在 go-admin 完成设备建档与激活后，调用 `/api/v1/device/auth` 获取短期 token。
2. 设备上线后用 Bearer token 通过 `MQTT report` 或 `POST /api/v1/device/report/status` 上报一次当前状态。
3. 后台调用 `/api/v1/platform-device/diagnosis/start` 发起诊断。
4. 设备在线时优先通过 MQTT 收到 `desired/diagnosis` 指令；离线时重连后可通过 `/api/v1/device/commands/pending` 补拉。
5. 设备执行完毕后调用 `/api/v1/device/diagnosis/report`。
6. 如需上传本地运行日志，再调用 `/api/v1/device/logs/upload`。
