# 设备状态上报（`POST /api/v1/device/report/status`）

## 请求

- **鉴权**：`device_sn` + `device_secret`（与 MQTT 上报一致）。
- **主字段**：`timestamp`、`reported`（JSON 对象字符串）、可选 `battery`、`run_state`、`firmware_version`、`online`、`ip`、`children`、`history`（离线 FIFO 补传批次）。

### `reported` 推荐结构（均为可选）

未传的字段表示云端未知；字段名使用 **snake_case**。

| 分组 | 字段 | 说明 |
|------|------|------|
| `power` | `percent`, `charging_state`, `is_charging`, `temperature_c`, `health_percent`, `estimated_minutes_remaining` | 电量 0–100；`charging_state` 如 `charging`/`full`/`idle`；充电中或已充满时云端建议缩短上报间隔为正常 60s |
| `storage` | `total_bytes`, `used_bytes`, `low_space`, `fs_health` | 校验：`used_bytes` ≤ `total_bytes`（若二者均提供） |
| `speakers` | `[{ name, mac, rssi, codec, channel, volume }]` | 已连接扬声器 |
| `uwb` | `x`, `y`, `z`, `accuracy_m`, `mode`, `refresh_hz`, `anchor_count`, `ts_ms` | `accuracy_m` ≥ 0 |
| `acoustic` | `calibration_state`, `calibrated_at_ms`, `freq_response`, `mic_sensitivity`, `speaker_delay_ms`, `av_sync_ms` | 校准状态 |
| `runtime` | `uptime_seconds`, `firmware_version` | 与顶层 `firmware_version` 并存时以合并后影子为准 |
| 顶层 | `report_mode` | `normal` \| `energy_saving` \| `emergency`；`energy_saving` 且未充电时建议间隔 300s |
| 顶层 | `emergency` | `true` 时视为紧急模式 |
| 顶层 | `alerts` | `[{ severity|level: info|warning|critical|emergency, ... }]`，`critical`/`emergency` 触发紧急间隔 |

顶层仍可放 **`battery`**（与 `power.percent` 二选一或并存，服务端会合并进影子）。

## 响应

| 字段 | 说明 |
|------|------|
| `next_interval` | 建议下次上报间隔（**秒**），由服务端根据配置 [StatusReport](services/device/etc/device.yaml) 与本次 `reported` 计算 |
| `version` | 设备影子版本 |
| `commands` | 待下发指令 |
| `accepted_reports` | 本请求实际落库的主机上报条数（含 history 中成功去重插入的） |

### `next_interval` 计算规则（摘要）

1. **充电**：`is_charging=true` 或 `power.charging_state` ∈ {`charging`,`full`,`charged`} → **60s**（可配置 `IntervalNormalSec`）。
2. **紧急**：`report_mode=emergency` 或 `emergency=true` 或 `alerts[].severity|level` 为 `critical`/`emergency` → **10s**（`IntervalEmergencySec`，夹在 min/max 内）。
3. **显式节能**：`report_mode=energy_saving` 且未充电 → **300s**（`IntervalEnergySec`）。
4. **按电量**（未充电，且未命中上条）：电量 **&lt; 20%** → 300s；**20%–50%** → 120s；否则 60s。电量取自顶层 `battery` 或 `power.percent`。

上下限：`IntervalMinSec`（默认 10）、`IntervalMaxSec`（默认 300）。

## 服务端校验

若 `battery`、`power.percent`、`storage` 字节字段、`uwb.accuracy_m` 等超出合理范围，接口返回参数错误，**整批拒绝**（避免脏数据进影子）。

## 告警（MVP）

阈值命中时写结构化日志 `[device_status_alert]`；`warning`/`critical` 异步写入表 `device_status_alert`（需执行迁移 `065_device_status_alert.sql`）。`info` 级仅日志。

## MQTT

主题与负载与 HTTP 对齐：`device/{sn}/report`，JSON 字段与 HTTP body 一致，`source` 为 `mqtt`，间隔策略与 HTTP 相同。
