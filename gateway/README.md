# API 网关部署与使用文档

## 概述
API 网关为音频 AI 平台提供统一的入口，实现安全、限流、日志、认证的全链路管控。

## 功能特性

### ✅ 已完成功能
- **统一入口管理**：所有微服务通过网关统一暴露
- **智能路由转发**：根据路径前缀自动转发到对应服务
- **JWT 认证**：统一身份验证，支持白名单路径
- **多维度限流**：全局、IP、用户三级限流保护
- **全链路日志**：请求/响应日志记录，支持结构化输出
- **CORS 跨域**：统一跨域配置，支持移动端和前端
- **健康监控**：服务状态检查和监控接口

## 部署步骤

### 1. 环境要求
- Go 1.22+
- Redis（用于限流，可选）
- 各微服务已启动（用户、设备、内容、后台服务）

### 2. 启动网关服务

```bash
# 进入网关目录
cd gateway

# 下载依赖
go mod tidy

# 启动网关
go run main.go
```

### 3. 验证网关启动

访问健康检查接口：
```bash
curl http://localhost:8080/health
```

预期响应：
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "routes": [
    {
      "path_prefix": "/api/v1/platform-",
      "target": "http://127.0.0.1:8000",
      "timeout": "30s",
      "name": "admin-service"
    },
    ...
  ]
}
```

## 路由配置说明

网关根据路径前缀自动转发请求到对应微服务：

| 路径前缀 | 目标服务 | 端口 | 说明 |
|---------|---------|------|------|
| `/api/v1/platform-` | 后台管理服务 | 8000 | 后台管理接口 |
| `/api/v1/user` | 用户服务 | 8001 | 用户认证、注册、信息管理 |
| `/api/v1/device` | 设备服务 | 8002 | 设备管理、状态上报、指令下发 |
| `/api/v1/content` | 内容服务 | 8003 | 内容管理、播放、下载 |

## 认证机制

### JWT 认证流程
1. 用户通过 `/api/v1/user/login` 获取 JWT token
2. 后续请求在 Header 中添加：`Authorization: Bearer <token>`
3. 网关统一验证 token 有效性

### 免认证路径（白名单）
- `/api/v1/user/login` - 用户登录
- `/api/v1/user/register` - 用户注册
- `/api/v1/user/refresh-token` - token 刷新
- `/api/v1/device/auth` - 设备认证
- `/api/v1/device/heartbeat` - 设备心跳
- `/health` - 健康检查

## 限流策略

### 三级限流保护
1. **全局限流**：1000 请求/秒
2. **IP 限流**：100 请求/秒/IP
3. **用户限流**：50 请求/秒/用户

### 限流响应
当触发限流时，返回：
```json
{
  "error": "请求过于频繁",
  "limit_type": "ip/user/global",
  "retry_after": 1
}
```

## 日志系统

### 日志格式
网关记录完整的请求/响应信息：
- 请求方法、路径、查询参数
- 客户端 IP、User-Agent
- 响应状态码、处理时长
- 用户信息（如已认证）
- 请求/响应体（小文件）

### 日志级别
- `debug`：详细调试信息
- `info`：正常请求日志
- `warn`：客户端错误（4xx）
- `error`：服务端错误（5xx）

## 监控接口

### 健康检查
```
GET /health
```

### 监控指标
```
GET /metrics
```

响应示例：
```json
{
  "status": "running",
  "timestamp": 1672531200,
  "rate_limits": {
    "global_rps": 1000,
    "ip_rps": 100,
    "user_rps": 50,
    "window_seconds": 1
  },
  "routes": [...]
}
```

## 配置说明

### 配置文件 (config.yaml)
```yaml
# 基础配置
Name: api-gateway
Host: 0.0.0.0
Port: 8080

# 日志配置
Log:
  Level: info
  Format: json

# JWT 认证
JWT:
  Secret: audio-ai-platform-gateway-secret
  ExpireHours: 24
  SkipPaths: [...]

# 限流配置
RateLimit:
  GlobalRPS: 1000
  IPRPS: 100
  UserRPS: 50
  WindowSeconds: 1

# 路由配置
Routes:
  - PathPrefix: "/api/v1/platform-"
    Target: "http://127.0.0.1:8000"
    Timeout: 30s
    Name: admin-service
  ...

# CORS 配置
CORS:
  AllowOrigins: ["http://localhost:9527", "*"]
  AllowMethods: ["GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"]
  AllowHeaders: ["Origin", "Content-Type", "Accept", "Authorization"]
  MaxAge: 86400
  AllowCredentials: true

# Redis 配置（限流使用）
Redis:
  Addr: localhost:6379
  Password: ""
  DB: 0
```

## 测试用例

### 1. 健康检查测试
```bash
curl -X GET http://localhost:8080/health
```

### 2. 用户登录测试（免认证）
```bash
curl -X POST http://localhost:8080/api/v1/user/login \
  -H "Content-Type: application/json" \
  -d '{"username":"test","password":"123456"}'
```

### 3. 内容查询测试（需认证）
```bash
curl -X GET http://localhost:8080/api/v1/content/list \
  -H "Authorization: Bearer <your-jwt-token>"
```

### 4. 限流测试
```bash
# 快速连续请求测试限流
for i in {1..60}; do
  curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/health
  sleep 0.1
done
```

## 性能压测

### 使用 wrk 进行压测
```bash
# 安装 wrk
sudo apt install wrk

# 压测健康检查接口（无认证）
wrk -t12 -c400 -d30s http://localhost:8080/health

# 压测认证接口
wrk -t12 -c400 -d30s -H "Authorization: Bearer <token>" http://localhost:8080/api/v1/content/list
```

### 预期性能指标
- 单机 QPS：5000+
- 并发连接：1000+
- 平均延迟：< 50ms
- 99% 延迟：< 200ms

## 故障排查

### 常见问题
1. **网关无法启动**
   - 检查端口 8080 是否被占用
   - 检查配置文件路径和格式

2. **路由转发失败**
   - 确认目标服务已启动
   - 检查目标服务端口和路径

3. **认证失败**
   - 检查 JWT token 格式和有效期
   - 确认认证服务正常运行

4. **限流触发频繁**
   - 调整限流参数配置
   - 检查是否有异常请求

### 日志分析
查看网关日志定位问题：
```bash
# 查看实时日志
tail -f gateway.log

# 搜索错误日志
grep "ERROR" gateway.log

# 搜索特定路径的请求
grep "/api/v1/user/login" gateway.log
```

## 安全建议

1. **生产环境配置**
   - 修改 JWT Secret
   - 启用 HTTPS
   - 配置防火墙规则

2. **监控告警**
   - 设置网关健康检查告警
   - 监控限流触发频率
   - 日志异常检测

3. **备份恢复**
   - 定期备份配置文件
   - 建立服务恢复流程

## 后续优化方向

1. **功能增强**
   - 熔断器机制
   - 服务发现集成
   - API 文档自动生成

2. **性能优化**
   - 连接池优化
   - 缓存策略
   - 负载均衡

3. **监控增强**
   - Prometheus 指标集成
   - 分布式追踪
   - 性能分析

---

**文档版本**: v1.0  
**最后更新**: 2026-04-13  
**维护团队**: 音频 AI 平台开发组