# App 用户 WebSocket · 歌曲下载链路

## 路径与鉴权

- URL：`GET ws://<device-host>:<port>/ws/user`
- 与用户 HTTP 接口相同 **用户 JWT**（`Authorization: Bearer <token>`）；也可在握手 URL 使用 `?token=` / `?access_token=`。

设备侧仍为 `GET /ws/device`（设备 JWT），二者不可混用。

## App → 服务端

发送文本帧 JSON。

### 订阅设备状态（影子推送）

```json
{
  "type": "subscribe",
  "device_sn": "AUSP2605000002Y2"
}
```

下行成功示例：`{"cmd":"subscribe_success","device_sn":"..."}`。**不要**把 `subscribe_success` 当作上行命令发送。

### 歌曲下载

```json
{
  "type": "download_song",
  "device_id": 123,
  "content_id": 456,
  "song_name": "可选",
  "audio_url": "可选：已解析 CDN 地址；不传则由服务端按 content_id 请求内容服务",
  "sn": "可选：未传 device_id 时用 16 位 SN",
  "task_id": "可选：与第三方任务对齐"
}
```

服务端下行（即时）示例：

```json
{
  "type": "download_accepted",
  "task_id": "...",
  "content_id": 456,
  "dispatch_status": "delivered",
  "message": "..."
}
```

## 服务端 → App（异步）

同一连接上推送：

| `type`                | 说明 |
|-----------------------|------|
| `download_device_ack` | 设备已应答收到指令（需在设备 JSON 中带 `task_id`，见下） |
| `download_progress`    | 设备上报进度 |
| `download_result`      | 成功/失败终态；含 `message`、`error_msg` |

## 设备上行（与现有 `/ws/device` 共用）

1. **可选应答**（转发给当前发起下载的用户连接）：

```json
{
  "type": "cmd_response",
  "cmd": "download_song",
  "task_id": "<与下行指令一致>",
  "message": "已收到，准备开始下载"
}
```

2. **进度**：

```json
{
  "event": "download_progress",
  "task_id": "...",
  "song_id": 456,
  "percent": 30.0
}
```

3. **结果**：

```json
{
  "event": "download_finish",
  "task_id": "...",
  "song_id": 456,
  "status": "success",
  "local_path": "...",
  "file_size": 12345,
  "duration_ms": 8900
}
```

失败时：`"status": "fail"`，且必须带 `error_msg`。

## 服务端行为摘要

- 校验用户与设备绑定，预写 `user_downloads` 为 `downloading`，经指令系统 **WebSocket** 下发 `download_song`。
- 指令创建成功后登记 `task_id`，用于将设备上行与 **当前用户** WS 关联。
- 收到 `download_finish` 后更新 `content_download_stats`、`user_downloads`，并向 App 推送 `download_result`。

说明：用户 WS 连接池为**进程内内存**；多实例部署时需由网关粘滞到同一节点，或后续再接 Redis 分发（与设备侧 WS relay 类似）。
