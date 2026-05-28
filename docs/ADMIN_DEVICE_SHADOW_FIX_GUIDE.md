# Admin后台设备影子接口 - 修复与验证指南

## 📌 当前状态

✅ **Admin服务中已完整实现设备影子功能**（不需要在device后端服务中重复实现）

### 已有功能清单

| 功能 | API路径 | 状态 |
|------|---------|------|
| 设备影子列表 | `GET /api/v1/platform-device/shadow/list` | ✅ 已实现 |
| 设备影子详情 | `GET /api/v1/platform-device/shadow?sn=xxx` | ✅ 已实现 |
| 更新期望状态 | `PUT /api/v1/platform-device/shadow/desired` | ✅ 已实现 |
| 规范化影子查询 | `GET /api/v1/platform-device/devices/:sn/shadow` | ✅ 已实现 |
| 前端页面 | `/platform-device-shadow/index` | ✅ 已实现 |

---

## 🔧 修复步骤

### 步骤1：确保数据库表存在

执行以下SQL创建 `device_shadow` 表（如果不存在）：

```sql
-- 在PostgreSQL中执行
CREATE TABLE IF NOT EXISTS public.device_shadow (
    id BIGSERIAL PRIMARY KEY,
    device_id BIGINT NOT NULL REFERENCES public.device(id) ON DELETE CASCADE,
    sn VARCHAR(64) NOT NULL,
    reported JSONB DEFAULT '{}'::jsonb,
    desired JSONB DEFAULT '{}'::jsonb,
    metadata JSONB DEFAULT '{}'::jsonb,
    version BIGINT NOT NULL DEFAULT 0,
    last_report_time TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_device_shadow_device_id UNIQUE (device_id)
);

CREATE INDEX IF NOT EXISTS idx_device_shadow_sn ON public.device_shadow(sn);

-- 补齐可能缺失的列
ALTER TABLE public.device_shadow ADD COLUMN IF NOT EXISTS version BIGINT NOT NULL DEFAULT 0;
ALTER TABLE public.device_shadow ADD COLUMN IF NOT EXISTS metadata JSONB DEFAULT '{}'::jsonb;
ALTER TABLE public.device_shadow ADD COLUMN IF NOT EXISTS desired JSONB DEFAULT '{}'::jsonb;
```

### 步骤2：清理Go模块缓存并重新编译Admin服务

```bash
# 进入admin目录
cd d:\audio-ai-platform\admin

# 清理模块缓存（解决 "package go-admin/app/admin/jobs is not in std" 错误）
go clean -modcache

# 重新下载依赖
go mod download

# 编译（只检查语法，不生成可执行文件）
go build -o nul ./... 2>&1

# 如果成功，启动开发服务器
go run main.go
```

### 步骤3：验证Admin服务是否正常启动

```bash
# 启动后应该看到类似输出：
# [INFO]  server is starting...
# [INFO]  server run success on port=:8000

# 测试API是否可用
curl http://127.0.0.1:8000/api/v1/platform-device/shadow/list?page=1&pageSize=10 \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 步骤4：检查Redis连接配置

Admin服务的设备影子功能依赖Redis，确保：

1. **Redis服务已启动**
   ```bash
   redis-cli ping
   # 应返回: PONG
   ```

2. **Admin配置文件包含Redis配置**
   - 检查 `d:\audio-ai-platform\admin` 目录下的配置文件
   - 确保有Redis连接地址（通常在settings.yaml或环境变量中）

3. **查看Admin日志中的Redis连接状态**
   ```
   [INFO] redis connect success...
   ```

---

## 🧪 测试设备影子接口

### 测试1：获取设备影子列表

```bash
curl -X GET "http://127.0.0.1:8000/api/v1/platform-device/shadow/list?page=1&pageSize=10" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

**预期响应：**
```json
{
  "code": 200,
  "data": {
    "list": [
      {
        "device_id": 1,
        "sn": "AUSP2605000002Y2",
        "firmware_version": "v1.0.0",
        "online_status": 1,
        "display_online": true,
        "battery": 85,
        "run_state": "playing",
        "has_reported": true,
        "has_desired": true,
        "last_report_time": "2024-06-28T15:30:00Z"
      }
    ],
    "count": 5,
    "page": 1,
    "pageSize": 10
  },
  "msg": "查询成功"
}
```

### 测试2：获取单个设备影子详情

```bash
curl -X GET "http://127.0.0.1:8000/api/v1/platform-device/shadow?sn=AUSP2605000002Y2" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

**预期响应：**
```json
{
  "code": 200,
  "data": {
    "device_id": 1,
    "sn": "AUSP2605000002Y2",
    "online": true,
    "firmware_version": "v1.0.0",
    "battery": 85,
    "power_status": "playing",
    "volume": 70,
    "reported": {
      "battery": 85,
      "volume": 70,
      "run_state": "playing",
      "firmware_version": "v1.0.0"
    },
    "desired": {
      "volume": 80
    },
    "delta": {
      "volume": 80
    },
    "redis_present": true,
    "last_report_time": "2024-06-28T15:30:00Z",
    "last_online_time": "2024-06-28T15:35:00Z"
  },
  "msg": "ok"
}
```

### 测试3：更新设备期望状态（desired）

```bash
curl -X PUT "http://127.0.0.1:8000/api/v1/platform-device/shadow/desired" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "sn": "AUSP2605000002Y2",
    "desired": {"volume": 90, "power_status": "pause"},
    "merge": true
  }'
```

**预期响应：**
```json
{
  "code": 200,
  "data": {
    "device_id": 1,
    "sn": "AUSP2605000002Y2",
    "desired": {"volume": 90, "power_status": "pause"},
    "delta": {"volume": 90, "power_status": "pause"},
    "pushed_mqtt": true,
    "online": true
  },
  "msg": "ok"
}
```

---

## 🎯 前端访问测试

### 1. 启动前端开发服务器

```bash
cd d:\audio-ai-platform\admin-ui
npm run dev
```

前端将运行在 `http://localhost:9527`

### 2. 访问设备影子页面

浏览器打开：`http://localhost:9527/platform-device-shadow/index`

### 3. 验证功能

- ✅ 页面正常加载（不再404）
- ✅ 查询按钮点击后显示数据列表
- ✅ 点击行可以查看影子详情（reported/desired/delta）
- ✅ 筛选条件工作正常（SN、在线状态、影子记录）

---

## 🔍 故障排查

### 问题1：仍然404

**检查项：**
```bash
# 1. 确认admin服务正在运行
netstat -an | findstr :8000

# 2. 确认路由已注册
curl http://127.0.0.1:8000/api/v1/platform-device/shadow/list

# 3. 查看admin服务日志，查找错误信息
tail -f logs/admin.log
```

### 问题2：返回空数据

**原因：** 数据库中没有设备或影子记录

**解决：**
```sql
-- 检查是否有设备数据
SELECT COUNT(*) FROM device WHERE deleted_at IS NULL;

-- 检查是否有影子数据
SELECT COUNT(*) FROM device_shadow;

-- 如果没有，先添加测试设备
INSERT INTO device (sn, name, online_status) VALUES ('TEST001', 'Test Device', 1);
```

### 问题3：Redis相关错误

**错误信息：** `redis: connection refused` 或 `nil pointer`

**解决：**
1. 启动Redis服务
2. 检查admin配置中的Redis地址
3. 重启admin服务

---

## 📊 架构说明

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────┐
│                 │     │                  │     │             │
│  admin-ui (:9527)│────▶│  admin (:8000)   │────▶│ PostgreSQL  │
│  (Vue前端)      │     │  (Go后端)        │     │ (数据库)    │
│                 │     │                  │     │             │
└─────────────────┘     └────────┬─────────┘     └─────────────┘
                                 │
                                 ▼
                        ┌─────────────────┐
                        │                 │
                        │  Redis (:6379)  │
                        │ (设备影子缓存)   │
                        │                 │
                        └─────────────────┘
```

**数据流：**
1. 前端调用 `/api/v1/platform-device/shadow/list`
2. Nginx/代理转发到admin服务 `:8000`
3. Admin服务查询PostgreSQL（设备表+影子表）+ Redis（实时状态）
4. 聚合数据返回给前端展示

---

## ✅ 完成清单

- [ ] 执行SQL创建/更新 `device_shadow` 表
- [ ] 清理Go模块缓存并重新编译admin服务
- [ ] 启动Redis服务
- [ ] 启动admin服务（`go run main.go`）
- [ ] 启动前端开发服务器（`npm run dev`）
- [ ] 访问 `http://localhost:9527/platform-device-shadow/index`
- [ ] 测试查询、筛选、详情查看功能
- [ ] 测试更新desired功能

---

## 💡 核心要点

1. **Admin服务已有完整实现** - 无需在device服务中重复开发
2. **聚合查询** - 同时从PostgreSQL和Redis获取数据，保证一致性
3. **自动计算Delta** - 对比desired和reported，显示差异字段
4. **支持分页筛选** - 支持按SN模糊搜索、在线状态过滤
5. **MQTT下发** - 更新desired时，如果设备在线会自动通过MQTT推送

按照以上步骤操作，即可解决404问题！🚀
