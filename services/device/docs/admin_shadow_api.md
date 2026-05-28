# Admin 后台设备影子管理 API 文档

## 概述

本文档描述 Admin 后台专用的设备影子管理接口，用于查看和管理设备的完整状态信息。

## 基础信息

- **基础路径**: `/api/admin/device/shadow`
- **认证方式**: 需要管理员权限（建议添加 JWT 或 API Key 认证）
- **数据格式**: JSON

---

## 1. 获取设备影子完整详情

查询指定设备的完整影子数据，包括 reported、desired、version、status 等所有字段。

### 请求

```
GET /api/admin/device/shadow/detail?device_sn={device_sn}
```

### 参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| device_sn | string | 是 | 设备序列号 |

### 响应示例

**成功 (200)**:
```json
{
  "code": 200,
  "success": true,
  "message": "查询成功",
  "data": {
    "device_sn": "AUSP2605000001",
    "found": true,
    "reported": {
      "power": "on",
      "volume": 80,
      "play_state": "playing",
      "current_song": "song_001"
    },
    "desired": {
      "volume": 90
    },
    "version": 42,
    "status": "online",
    "status_text": "在线",
    "update_time": 1704067200,
    "update_time_formatted": "1704067200",
    "metadata": {
      "firmware": "v1.2.3",
      "ip_address": "192.168.1.100"
    },
    "is_online": true,
    "reported_fields_count": 4,
    "desired_fields_count": 1
  }
}
```

**设备不存在 (404)**:
```json
{
  "code": 404,
  "success": false,
  "message": "设备影子不存在",
  "data": {
    "device_sn": "AUSP2605000001",
    "hint": "设备可能尚未上线或影子未被初始化"
  }
}
```

**参数错误 (400)**:
```json
{
  "code": 400,
  "success": false,
  "message": "device_sn 参数不能为空",
  "data": null
}
```

### 使用场景

- Admin 后台设备详情页面
- 设备故障排查
- 设备状态监控大屏
- 运维人员手动检查设备状态

---

## 2. 批量查询设备影子列表

一次查询多个设备的影子状态，适用于设备列表展示。

### 请求

```
POST /api/admin/device/shadow/list
Content-Type: application/json
```

### 请求体

```json
{
  "device_sns": [
    "AUSP2605000001",
    "AUSP2605000002",
    "AUSP2605000003"
  ]
}
```

### 参数限制

- `device_sns`: 必填，数组类型
- 单次最多查询 **50** 个设备
- 数组不能为空

### 响应示例

**成功 (200)**:
```json
{
  "code": 200,
  "success": true,
  "message": "查询成功",
  "data": {
    "total": 3,
    "found_count": 2,
    "not_found": ["AUSP2605000003"],
    "items": [
      {
        "device_sn": "AUSP2605000001",
        "found": true,
        "reported": {"power": "on"},
        "desired": {},
        "version": 42,
        "status": "online",
        "status_text": "在线",
        "update_time": 1704067200,
        "is_online": true,
        "reported_fields_count": 1,
        "desired_fields_count": 0
      },
      {
        "device_sn": "AUSP2605000002",
        "found": true,
        "reported": {"power": "off"},
        "desired": {},
        "version": 38,
        "status": "offline",
        "status_text": "离线",
        "update_time": 1704067100,
        "is_online": false,
        "reported_fields_count": 1,
        "desired_fields_count": 0
      },
      {
        "device_sn": "AUSP2605000003",
        "found": false
      }
    ]
  }
}
```

### 使用场景

- Admin 后台设备列表页面的状态列
- 批量设备健康检查
- 设备分组状态概览

---

## 3. 获取设备影子统计信息

获取所有设备影子的统计数据，用于监控大屏或仪表盘。

### 请求

```
GET /api/admin/device/shadow/stats
```

### 响应示例

**成功 (200)**:
```json
{
  "code": 200,
  "success": true,
  "message": "查询成功",
  "data": {
    "total_devices": 1000,
    "online_devices": 850,
    "offline_devices": 130,
    "abnormal_devices": 20,
    "online_rate": 85.0,
    "timestamp": 1704067200
  }
}
```

### 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| total_devices | int64 | 设备总数 |
| online_devices | int64 | 在线设备数 |
| offline_devices | int64 | 离线设备数 |
| abnormal_devices | int64 | 异常设备数 |
| online_rate | float64 | 在线率（百分比） |
| timestamp | int64 | 统计时间戳 |

### 使用场景

- 监控大屏首页
- 运营仪表盘
- 设备健康度报告
- 定时任务告警阈值判断

---

## 4. 删除设备影子

⚠️ **危险操作**：删除设备的影子数据，此操作不可逆！

### 请求

```
DELETE /api/admin/device/shadow?device_sn={device_sn}
```

### 参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| device_sn | string | 是 | 设备序列号 |

### 响应示例

**成功 (200)**:
```json
{
  "code": 200,
  "success": true,
  "message": "删除成功",
  "data": {
    "device_sn": "AUSP2605000001",
    "deleted": true
  }
}
```

**参数错误 (400)**:
```json
{
  "code": 400,
  "success": false,
  "message": "device_sn 参数不能为空",
  "data": null
}
```

### 使用场景

- 设备退役清理
- 测试环境重置
- 数据修复（谨慎使用）
- ⚠️ **不建议在生产环境频繁使用**

---

## 错误码说明

| HTTP 状态码 | code | 说明 |
|-------------|------|------|
| 200 | 200 | 成功 |
| 400 | 400 | 参数错误 |
| 404 | 404 | 设备影子不存在 |
| 500 | 500 | 服务器内部错误 |

---

## 最佳实践

### 1. 性能优化

- **批量查询**: 使用 `/list` 接口一次性查询多个设备，避免循环调用
- **缓存策略**: 统计接口 `/stats` 可考虑缓存 30-60 秒
- **分页支持**: 如需查询大量设备，建议实现分页功能

### 2. 安全建议

- **认证鉴权**: 所有 Admin 接口必须添加管理员身份验证
- **操作审计**: 删除操作应记录操作日志（谁、何时、删除了什么）
- **权限控制**: 删除接口仅限超级管理员使用
- **频率限制**: 防止恶意调用导致 Redis 压力过大

### 3. 监控告警

- **异常检测**: 当 `abnormal_devices` 数量超过阈值时触发告警
- **在线率监控**: 当 `online_rate` 低于预期时通知运维
- **响应时间**: 接口响应超过 1 秒时应优化或扩容

---

## 示例代码

### cURL 示例

```bash
# 1. 查询单个设备影子详情
curl -X GET "http://localhost:8000/api/admin/device/shadow/detail?device_sn=AUSP2605000001" \
  -H "Authorization: Bearer {admin_token}"

# 2. 批量查询设备列表
curl -X POST "http://localhost:8000/api/admin/device/shadow/list" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer {admin_token}" \
  -d '{"device_sns": ["AUSP2605000001", "AUSP2605000002"]}'

# 3. 获取统计信息
curl -X GET "http://localhost:8000/api/admin/device/shadow/stats" \
  -H "Authorization: Bearer {admin_token}"

# 4. 删除设备影子（谨慎！）
curl -X DELETE "http://localhost:8000/api/admin/device/shadow?device_sn=AUSP2605000001" \
  -H "Authorization: Bearer {admin_token}"
```

### JavaScript/TypeScript 示例

```typescript
// 查询设备详情
async function getDeviceShadowDetail(deviceSN: string) {
  const response = await fetch(
    `/api/admin/device/shadow/detail?device_sn=${deviceSN}`,
    {
      headers: {
        'Authorization': `Bearer ${getAdminToken()}`
      }
    }
  );
  
  const result = await response.json();
  
  if (!result.success) {
    throw new Error(result.message);
  }
  
  return result.data;
}

// 使用示例
const shadow = await getDeviceShadowDetail('AUSP2605000001');
console.log(`设备状态: ${shadow.status_text}`);
console.log(`版本号: ${shadow.version}`);
console.log(`上报属性:`, shadow.reported);
```

---

## 相关文档

- [设备影子 V2 API](./device_shadow_api_v2.md)
- [Redis Hash 数据结构设计](../docs/device_shadow_design.md)
- [WebSocket 实时推送机制](../docs/websocket_push.md)

---

## 更新日志

### v1.0.0 (2024-01-01)
- ✅ 初始版本发布
- ✅ 支持查询单个设备影子详情
- ✅ 支持批量查询设备列表
- ✅ 支持获取统计信息
- ✅ 支持删除设备影子（需谨慎使用）

---

## 联系方式

如有问题或建议，请联系开发团队：
- **技术负责人**: Dev Team
- **邮箱**: dev@example.com
- **文档最后更新**: 2024-01-01
