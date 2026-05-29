# audio-ai-platform

基于 Go 的 IoT 音频 AI 云平台 Monorepo，涵盖用户/设备/内容/媒体处理等微服务、运营后台与设备影子（Device Shadow）能力。

---

## 代码架构总览

```
┌──────────────┐     ┌──────────────┐     ┌─────────────────────────────┐
│  App / 设备端 │     │  admin-ui    │     │  API Gateway / Nginx (:8080) │
│  (REST/WS)   │     │  Vue2 (:9527)│     └──────────────┬──────────────┘
└──────┬───────┘     └──────┬───────┘                    │
       │                    │                            │
       └────────────────────┼────────────────────────────┘
                            │
       ┌────────────────────┼────────────────────┐
       │                    │                    │
┌──────▼──────┐  ┌──────────▼─────────┐  ┌──────▼──────┐  ┌──────────────────┐
│ user :8001  │  │ device :8002       │  │ content     │  │ media-processing │
│             │  │ (WS/MQTT/Shadow)   │  │ :8003       │  │ / ai-worker :8004│
└──────┬──────┘  └──────────┬─────────┘  └──────┬──────┘  └────────┬─────────┘
       │                    │                    │                 │
       └────────────────────┼────────────────────┴─────────────────┘
                            │
              ┌─────────────▼─────────────┐
              │ PostgreSQL (audio_platform)│
              │ Redis / MQTT / RabbitMQ     │
              │ S3 / OSS / Kafka(SQS)       │
              └─────────────────────────────┘
                            │
              ┌─────────────▼─────────────┐
              │ admin :8000 (go-admin)    │
              │ 平台设备/用户/内容运营 API   │
              └─────────────────────────────┘
```

---

## 顶层目录

| 目录 | 说明 |
|------|------|
| `services/` | 业务微服务（go-zero REST） |
| `admin/` | 运营后台 API（go-admin + Gin） |
| `admin-ui/` | 管理前端（Vue 2 + Element UI） |
| `common/` | 跨服务公共库（独立 `go.mod`：错误码、CORS、校验等） |
| `pkg/` | 根模块基础设施封装（AWS、Redis、MQTT、Kafka、JWT 等） |
| `gateway/` | 可选 API 网关（Gin，JWT/限流/转发） |
| `doc/` | 架构、部署、Swagger 文档 |
| `scripts/` | DB 迁移、Swagger 生成、本地/CI 脚本 |
| `deploy/` | Docker、K8s、Nginx、EMQX 等部署配置 |

---

## Monorepo 与模块关系

本仓库采用**多 `go.mod` + 本地 `replace`** 的 Monorepo 策略：

```
audio-ai-platform/              ← 根 go.mod（pkg、平台级依赖）
├── common/                     ← 独立 module
├── services/{user,device,...}/ ← 各自 go.mod
│     replace:
│       github.com/jacklau/audio-ai-platform => ../..
│       github.com/jacklau/audio-ai-platform/common => ../../common
└── admin/                      ← go-admin 独立 module
```

- 微服务通过 `replace` 引用本地 `common` 与根模块，避免 CI 向远端拉取私有包。
- 代码生成：各服务 `*.api`（goctl）→ `handler` / `types`；部分服务额外使用 swaggo。

---

## 微服务（go-zero）

统一分层：**handler → logic → svc → repository/repo**，配置在 `etc/*.yaml`。

| 服务 | 端口 | 入口 | 核心能力 |
|------|------|------|----------|
| **user** | 8001 | `services/user/user.go` | 注册登录、JWT、验证码、OAuth、实名、设备绑定/分享、会员权益 |
| **device** | 8002 | `services/device/device.go` | 设备注册/认证、WebSocket、播放控制、状态上报、指令下发、**设备影子**、歌单 |
| **content** | 8003 | `services/content/content.go` | 内容列表/详情、点赞收藏、歌单、艺术家订阅、消息通知 |
| **media-processing** | 8004 | `services/media-processing/main.go` | 推流地址、鉴权回调、DRM 透传、网易云 IoT 二维码登录等 |
| **ai-worker** | 8004* | `services/ai-worker/ai-worker.go` | 音频上传 OSS、音轨分离、AI 推理任务（与 media-processing 部署时需错开端口） |
| **ota** | 8005 | `services/ota/ota.go` | 固件版本批量检测、灰度升级 |

\* `ai-worker` 与 `media-processing` 默认同为 8004，生产环境需分开部署。

### device 服务内部结构（示例）

```
services/device/internal/
├── handler/          # HTTP/WS 路由入口
├── logic/            # 业务逻辑
├── svc/              # 依赖注入（DB、Redis、MQTT 等）
├── repository/       # PostgreSQL 持久化
├── shadowsvc/        # 影子查询聚合
├── device/shadowv2/  # Redis 热数据 + Lua 原子更新
├── commandsvc/       # 设备指令
├── rabbitmq/         # 指令异步削峰
└── heartbeat/        # 在线心跳
```

---

## 设备影子（Device Shadow）

采用经典 IoT Shadow 模型：`reported` / `desired` / `delta` / `version`。

```
App/Admin ──HTTP──► device 服务 ──► Redis (shadowv2, key: device:shadow:{SN})
                         │
                         ├──► PostgreSQL device_shadow 表
                         └──► 指令 ──► WebSocket / MQTT / RabbitMQ
设备端 ◄── WS / MQTT / HTTP report ──► 回写 reported
```

- **热路径**：`internal/device/shadowv2/`（Redis Hash + Lua）
- **持久化**：`repository/device_shadow_repo.go`
- **Admin 读模型**：`admin/app/admin/device/` 与微服务 Redis 键约定对齐，提供平台侧影子列表/详情 API

---

## 运营后台

### admin（go-admin）

- **框架**：Gin + go-admin-core + GORM + Casbin + JWT
- **启动**：`go run main.go server -c config/settings.yml`
- **业务扩展**：`app/admin/{device,user,content}/`（apis / service / models / router）
- **路由前缀**：`/api/v1/platform-*`
- **能力**：平台设备管理、影子查看、固件/OTA、用户/会员、内容运营、RBAC 权限

### admin-ui

- **栈**：Vue 2 + Vue Router + Vuex + Element UI + Axios
- **页面**：`src/views/admin/`（平台设备、影子、会员等）
- **API**：`src/api/admin/` + OpenAPI 生成的 `src/api/openapi-generated/`

---

## 公共库

### `common/`（独立 module）

| 包 | 用途 |
|----|------|
| `errorx` | 业务错误码、HTTP 状态映射、统一 JSON 响应 `{code,msg,data}` |
| `cors` | 跨域中间件 |
| `validate` | 请求参数校验 |
| `httpresp` | 标准响应辅助 |
| `middleware` | 通用鉴权/日志 |

### `pkg/`（根 module）

| 包 | 用途 |
|----|------|
| `awsx` | S3 封装 |
| `redisx` | Redis 初始化 |
| `mqttx` | MQTT 客户端 |
| `kafkax` / `events` | Kafka 事件 |
| `jwtx` | JWT 签发/解析 |
| `devicebind` | 跨服务设备绑定逻辑 |
| `logger` | Zap 日志 |

---

## 数据与中间件

| 组件 | 用途 |
|------|------|
| **PostgreSQL** | 主库 `audio_platform`；迁移脚本 `scripts/db/migrations/` |
| **Redis** | 设备影子热数据、在线状态、验证码/Session |
| **MQTT**（EMQX/Mosquitto） | 指令下发、影子 desired 推送、OTA |
| **RabbitMQ** | device 服务指令异步队列 |
| **Kafka / SQS** | 事件总线、媒体/AI 异步任务 |
| **S3 / OSS** | 音频与内容对象存储 |

---

## 文档与工具链

| 类别 | 位置 |
|------|------|
| 架构详设 | `doc/架构设计.md` |
| 快速开始 | `doc/快速开始.md`、`PROJECT.md` |
| Swagger UI | `doc/swagger/`（`scripts/serve-swagger-ui.ps1`） |
| OpenAPI 客户端 | `scripts/gen-openapi-client.ps1` → `admin-ui/src/api/openapi-generated/` |
| CI | `.github/workflows/ci.yml`（lint / test / build / 集成测试） |
| 本地 CI | `scripts/ci-local.sh` / `scripts/ci-local.ps1` |

---

## 服务端口速查

| 组件 | 端口 |
|------|------|
| Admin API | 8000 |
| User | 8001 |
| Device | 8002 |
| Content | 8003 |
| Media-processing / AI-worker | 8004 |
| OTA | 8005 |
| Gateway / Nginx | 8080 |
| Admin-UI（开发） | 9527 |
| PostgreSQL | 5432（Docker 常映射 5433） |
| Redis | 6379 |

---

## 快速开始

```powershell
# 仓库根目录
.\scripts\serve-swagger-ui.ps1    # 本地 API 文档（默认 http://127.0.0.1:8090）
bash scripts/ci-local.sh          # 本地 lint + test + build（Git Bash）
```

更多命令与部署说明见 [`PROJECT.md`](PROJECT.md)、[`doc/快速开始.md`](doc/快速开始.md)。
