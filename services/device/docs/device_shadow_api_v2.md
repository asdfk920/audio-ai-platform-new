# 设备影子 API（device 微服务）

本文档对应当前 `device` 微服务内实现的影子能力，覆盖 App 查询/写入影子、设备拉取待执行命令、设备回执执行结果。

## 1. 查询设备影子

- 路径：`GET /api/v1/device/shadow`
- 鉴权：JWT
- 说明：仅允许查询当前用户已绑定设备

请求参数：

```http
GET /api/v1/device/shadow?device_sn=SN12345678
Authorization: Bearer <access_token>
```

响应示例：

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "device_id": 1001,
    "device_sn": "SN12345678",
    "online": true,
    "reported": {
      "power": "on",
      "temperature": 22,
      "mode": "cool",
      "battery": 88
    },
    "desired": {
      "power": "on",
      "temperature": 25
    },
    "delta": {
      "temperature": 25
    },
    "metadata": {
      "reported": {
        "temperature": { "timestamp": 1710000000 }
      },
      "desired": {
        "temperature": { "timestamp": 1710000010 }
      }
    },
    "version": 12,
    "last_report_time": 1710000000
  }
}
```

## 2. App 写入 reported

- 路径：`PUT /api/v1/device/shadow/reported`
- 鉴权：JWT
- 说明：用于 App 通过蓝牙读取设备真实状态后回写云端

请求示例：

```json
{
  "device_sn": "SN12345678",
  "reported": {
    "power": "on",
    "temperature": 22,
    "mode": "cool",
    "online": true
  }
}
```

## 3. App 写入 desired

- 路径：`PUT /api/v1/device/shadow/desired`
- 鉴权：JWT
- 说明：用户在 App 侧操作设备时写入目标状态，同时生成待执行命令

请求示例：

```json
{
  "device_sn": "SN12345678",
  "merge": true,
  "desired": {
    "power": "on",
    "temperature": 25,
    "mode": "cool"
  }
}
```

说明：

- `merge=true`：与现有 `desired` 做 JSON merge
- `merge=false`：以本次请求完全覆盖 `desired`
- 响应里新增：
  - `instruction_id`：本次生成的真实指令 ID
  - `command_status`：`dispatched` / `cached` / `noop`
  - `queued_count`：当前设备未完成指令数
  - `expires_at`：本次指令过期时间
  - `instruction_type`：`manual` / `scheduled`
  - `command_code`：当前统一命令码，当前主链路为 `shadow_sync`

## 4. 设备拉取待执行命令

- 路径：`GET /api/v1/device/commands/pending`
- 鉴权：设备身份，优先 `Authorization: Bearer <device_token>`，兼容 `device_sn + device_secret`
- 说明：设备上线后或 App 补偿同步时拉取未完成命令

请求示例：

```http
GET /api/v1/device/commands/pending?device_sn=SN12345678&limit=20
Authorization: Bearer <device_token>
```

响应示例：

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "device_sn": "SN12345678",
    "list": [
      {
        "instruction_id": 501,
        "cmd": "shadow_sync",
        "command_code": "shadow_sync",
        "instruction_type": "manual",
        "params": {
          "temperature": 25
        },
        "status": 2,
        "priority": 100,
        "retry_count": 0,
        "expires_at": 1710000700,
        "created_at": 1710000100
      }
    ]
  }
}
```

## 5. 设备上报命令执行结果

- 路径：`POST /api/v1/device/commands/result`
- 鉴权：设备身份，优先 `Authorization: Bearer <device_token>`，兼容 `device_sn + device_secret`
- 说明：设备执行命令后回执状态；若同时携带 `reported`，云端会继续更新影子并重新计算 `delta`

请求示例：

```json
{
  "device_sn": "SN12345678",
  "instruction_id": 501,
  "status": 3,
  "result": {
    "ok": true
  },
  "reported": {
    "power": "on",
    "temperature": 25,
    "mode": "cool",
    "online": true
  }
}
```

状态说明：

- `1`: pending
- `2`: executing
- `3`: success
- `4`: failed
- `5`: timeout
- `6`: cancelled

## 6. 当前链路建议

1. 设备在 go-admin 后台完成建档/激活后，调用 `/auth` 获取短期 `device_token`（设备微服务已移除 `POST /register`）
2. App 绑定设备后，先查一次 `/shadow`
3. App 下发控制时写 `/shadow/desired`
4. 设备在线时优先通过 MQTT 接收命令
5. 当设备断网后重新连上并重新上报状态时，云端会再次尝试把 pending 的影子/诊断命令通过 MQTT 推送给设备
6. 如果设备侧没有及时收到 MQTT，设备重连后仍可通过 `/commands/pending` 补拉
7. 设备执行完成后调用 `/commands/result`，并附带最新 `reported`

## 8. 用户侧指令历史与取消

- `GET /api/v1/device/commands/history`
  - JWT
  - 支持 `device_sn`、`status`、`page`、`page_size`
- `GET /api/v1/device/commands/detail?instruction_id=xxx`
  - JWT
  - 返回单条指令详情和状态流水
- `POST /api/v1/device/commands/cancel`
  - JWT
  - 仅允许取消当前用户发起且尚未完成的指令

取消请求示例：

```json
{
  "instruction_id": 501,
  "reason": "user_cancelled"
}
```

## 9. 定时指令接口

- `POST /api/v1/device/commands/schedule`
- `PUT /api/v1/device/commands/schedule`
- `POST /api/v1/device/commands/schedule/cancel`
- `GET /api/v1/device/commands/schedule/list`

创建一次性任务示例：

```json
{
  "device_sn": "SN12345678",
  "schedule_type": "once",
  "desired_payload": {
    "power": "off"
  },
  "merge_desired": true,
  "execute_at": 1710003600,
  "expires_at": 1710007200
}
```

创建 cron 任务示例：

```json
{
  "device_sn": "SN12345678",
  "schedule_type": "cron",
  "desired_payload": {
    "volume": 15
  },
  "merge_desired": true,
  "cron_expr": "0 8 * * *",
  "timezone": "Asia/Shanghai"
}
```

说明：

- 调度 worker 会把到期 schedule 转成真实 `device_instruction`
- 生成后的执行链路与即时指令完全一致：在线 MQTT 推送，离线 pending 缓存，设备侧仍走 `/commands/pending` 和 `/commands/result`

## 7. MQTT 主题与双写（ShadowMQTT）

配置见 `services/device/etc/device.yaml` 中 `ShadowMQTT`：

| 用途 | 主题模板（占位 `{sn}` 大写 SN，`{id}` 为 device_id） | 载荷 |
|------|------------------------------------------------------|------|
| Legacy 下发 | `device/{sn}/desired` | `type: shadow_delta` 指令包（与历史一致） |
| 新层级（SN） | `device/shadow/{sn}/desired` | 同上 |
| 新层级（ID） | `device/shadow/{id}/desired` | 同上 |
| 可选纯 delta | `device/shadow/{sn}/desired/delta` / `device/shadow/{id}/desired/delta` | 仅 JSON delta（需 `PublishJSONDelta: true`） |

- `EnableLegacyTopics` / `PublishShadowBySN` / `PublishShadowByID` 控制是否向对应主题各发一份（默认 `true`）。
- `DesiredPublishQOS` / `DeltaPublishQOS`：0–2，默认 1。
- 设备端应对同一 `instruction_id` 幂等，避免双主题重复投递导致重复执行。

## 8. 设备影子一键同步（HTTP，断网重连）

- 路径：`GET /api/v1/device/shadow/sync`
- 鉴权：`device_sn` + `device_secret`（与 `/report/status`、`/commands/pending` 一致，支持 Bearer 设备 token）
- 查询参数：`client_version`（可选，设备本地缓存的 shadow version）、`limit`（待执行命令条数，默认 20）

响应在单设备影子字段基础上增加：

- `pending_commands`：与 `/commands/pending` 同源
- `server_time`：Unix 秒
- `version_stale`：当 `client_version > 0` 且小于服务端 `version` 时为 `true`，提示需以本响应为准重同步

## 9. MQTT 重连说明（补充）

- 云端下发除 legacy `device/{sn}/desired` 外，可按配置同时向 `device/shadow/{sn}/desired` 与 `device/shadow/{id}/desired` 双写。
- 后端 MQTT 订阅在 broker 重连后会自动恢复。
- 设备错过实时推送时：可发状态上报触发 `PushPending`，或调用 `GET /shadow/sync` / `GET /commands/pending` 兜底。
