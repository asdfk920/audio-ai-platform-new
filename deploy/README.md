# 部署与平台组件

## 统一网关（Nginx）

- 配置：`deploy/gateway/nginx.conf`
- 合并启动后监听 **`http://localhost:8080`**
- 路由约定：
  - ` /api/v1/user/*` → 本机 **8001**
  - ` /api/v1/device/*` → 本机 **8002**（去掉前缀后转发）
  - ` /api/v1/content/*` → 本机 **8003**（去掉前缀后转发）
- 生产请替换为 **Kong / 云 API 网关**，并增加 TLS、限流、WAF、集中鉴权与审计。

## Kafka 协议消息总线（Redpanda 开发节点）

- `docker-compose.platform.yml` 中 **`redpanda`**
- 本机客户端：`localhost:29092`
- Go 封装：`pkg/kafkax`，主题常量：`pkg/events`

## MQTT（Mosquitto）

- 端口 **1883**，配置见 `deploy/mosquitto/mosquitto.conf`（**仅开发匿名**）
- Go 封装：`pkg/mqttx`
- 设备影子、OTA、状态上报等建议在 **MQTT + Redis/DB** 组合上迭代，避免仅靠轮询数据库。

## 启动示例

```bash
# 基础依赖（PG、Redis…）
docker compose -f docker-compose.yml up -d

# 叠加网关 + Redpanda + MQTT
docker compose -f docker-compose.yml -f docker-compose.platform.yml up -d
```

确保本机已启动 `user-api` / `device-api` / `content-api` 对应端口，网关容器通过 `host.docker.internal` 访问宿主机。
