# Docker 外网访问配置指南

## 📋 配置说明

已为 AI Worker 服务配置 **两种网络模式**，确保 Docker 容器可以访问外网下载 OSS 音频文件。

---

## 🔧 方案对比

| 特性 | Host 模式（默认） | Bridge 模式 |
|------|------------------|-------------|
| **外网访问** | ✅ 直接访问 | ✅ 需配置 DNS |
| **性能** | ⭐⭐⭐ 最佳 | ⭐⭐ 良好 |
| **隔离性** | ❌ 无隔离 | ✅ 完全隔离 |
| **端口冲突** | ⚠️ 可能冲突 | ✅ 不会冲突 |
| **适用场景** | 开发/测试 | 生产环境 |

---

## 🚀 快速启动（推荐）

### 方式一：使用启动脚本（最简单）

```bash
# 双击运行或命令行执行
cd services/ai-worker
start-docker-network.bat

# 选择 [1] Host 网络模式（推荐）
```

### 方式二：手动启动

#### ✅ Host 网络模式（推荐用于开发）

```bash
cd services/ai-worker

# 启动服务
docker-compose up -d --build ai-worker

# 查看日志
docker logs -f ai-worker --tail 100

# 测试外网访问
docker exec ai-worker curl -I https://220aaa.oss-cn-beijing.aliyuncs.com/test1.mp3
```

#### ✅ Bridge 网络模式（推荐用于生产）

```bash
cd services/ai-worker

# 使用优化版配置启动
docker-compose -f docker-compose.bridge.yml up -d --build ai-worker

# 查看容器 IP
docker inspect ai-worker --format='{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}'

# 测试外网访问
docker exec ai-worker ping -c 3 8.8.8.8
```

---

## 🔍 验证外网连通性

### 1️⃣ 检查容器网络模式

```bash
docker inspect ai-worker --format='{{.HostConfig.NetworkMode}}'

# 输出: host (表示使用宿主机网络)
```

### 2️⃣ 测试 DNS 解析

```bash
# Host 模式（直接在容器内测试）
docker exec ai-worker nslookup oss-cn-beijing.aliyuncs.com

# Bridge 模式
docker exec ai-worker nslookup oss-cn-beijing.aliyuncs.com
```

### 3️⃣ 测试 HTTP 访问（OSS）

```bash
# 测试下载 OSS 文件
docker exec ai-worker curl -v https://220aaa.oss-cn-beijing.aliyuncs.com/test1.mp3 \
  --connect-timeout 10 \
  -o /dev/null \
  -w "HTTP Status: %{http_code}\nSize: %{size_download} bytes\nTime: %{time_total}s\n"
```

### 4️⃣ 测试 WebSocket 连接

```bash
# 健康检查
curl http://localhost:8004/api/v1/health

# 应返回：
# {"status":"ok","message":"服务运行正常"}
```

---

## 📊 完整工作流程验证

```bash
# 1. 启动容器
docker-compose up -d --build ai-worker

# 2. 等待服务就绪（约 30-60 秒）
docker logs -f ai-worker --tail 50

# 看到：✅ AI Worker Service 启动中...

# 3. 测试健康检查
curl http://localhost:8004/api/v1/health

# 4. 在 Apifox 中测试 WebSocket
# URL: ws://localhost:8004/api/v1/audio/separate/ws?task_id=test_001&token=YOUR_TOKEN
# 发送:
{
  "type": "separate",
  "data": {
    "content_id": 2
  }
}

# 5. 观察日志输出
# 应该看到：
# [WebSocket] 下载远程文件: https://220aaa.oss-cn-beijing.aliyuncs.com/test1.mp3
# [WebSocket] HTTP 响应状态: 200
# [Demucs] 开始推理...
# [Demucs] 推理完成...
```

---

## ⚠️ 常见问题排查

### 问题 1：容器无法访问外网

**症状**：
```
[WebSocket] 下载请求失败: dial tcp: lookup xxx: no such host
```

**解决方案**：

#### Host 模式：
```bash
# 检查宿主机网络
ping 8.8.8.8

# 重启 Docker Desktop
# Windows 任务栏 → 右键 Docker → Restart
```

#### Bridge 模式：
```bash
# 方法 1: 添加 DNS
docker-compose -f docker-compose.bridge.yml down
# 编辑 docker-compose.bridge.yml，修改 dns 为：
dns:
  - 223.5.5.5  # 阿里 DNS
  - 114.114.114.114

# 方法 2: 使用 host-gateway
extra_hosts:
  - "host.docker.internal:host-gateway"

# 重新启动
docker-compose -f docker-compose.bridge.yml up -d
```

### 问题 2：OSS AccessDenied

**症状**：
```
AccessDenied: You have no right to access this object because of bucket acl
```

**解决方案**：

1. **设置 Bucket 公开读权限**（临时测试）：
   - 登录 [阿里云 OSS 控制台](https://oss.console.aliyun.com/)
   - 选择 Bucket `220aaa`
   - 权限控制 → 读写权限 → **公共读**
   - 或对单个文件设置公开读

2. **使用签名 URL**（生产推荐）：
   ```python
   # 在后端代码中使用 OSS SDK 生成签名 URL
   import oss2
   auth = oss2.Auth('LTAI5t8vrb1GWkvu1Bw5T4k4', '7GMwzmyWLbMgW7HpPnjkMZTEYEfNLs')
   bucket = oss2.Bucket(auth, 'https://oss-cn-beijing.aliyuncs.com', '220aaa')
   signed_url = bucket.sign_url('GET', 'test1.mp3', 3600)
   ```

### 问题 3：Demucs 执行失败 (exit status 2)

**症状**：
```
AI 分离失败：Demucs 推理失败: exit status 2
```

**解决方案**：

```bash
# 进入容器调试
docker exec -it ai-worker bash

# 手动测试 demucs
export KMP_DUPLICATE_LIB_OK=TRUE
demucs --help

# 如果提示找不到模型，首次运行会自动下载
demucs -n htdemucs /app/data/output/test_audio.mp3

# 退出容器
exit
```

### 问题 4：端口被占用

**症状**（仅 Bridge 模式）：
```
Error: Port already in use: 0.0.0.0:8004
```

**解决方案**：

```bash
# 查找占用端口的进程
netstat -ano | findstr :8004

# 终止进程（替换 PID）
taskkill /PID <PID> /F

# 或修改端口映射（编辑 docker-compose.bridge.yml）
ports:
  - "18004:8004"  # 改用其他端口
```

---

## 📁 文件说明

| 文件名 | 用途 |
|--------|------|
| `docker-compose.yml` | **主配置** - Host 网络模式（默认） |
| `docker-compose.bridge.yml` | **备选配置** - Bridge 网络模式（生产） |
| `start-docker-network.bat` | **启动脚本** - 交互式选择网络模式 |
| `Dockerfile` | 镜像构建文件（无需修改） |

---

## 🎯 推荐配置（根据场景）

### 开发环境
```bash
# ✅ 使用 Host 模式 + 直接运行 Go 二进制
cd services/ai-worker
go run ai-worker.go -f etc/ai-worker.yaml
```

### Docker 本地开发
```bash
# ✅ 使用 Host 模式（简单快速）
start-docker-network.bat
# 选择 [1]
```

### 生产部署
```bash
# ✅ 使用 Bridge 模式（安全隔离）
docker-compose -f docker-compose.bridge.yml up -d --build
# 配置 Nginx 反向代理 + SSL
```

---

## ✅ 验证清单

- [ ] Docker Desktop 已安装并运行
- [ ] NVIDIA GPU 驱动已安装（如需 GPU）
- [ ] 已执行 `docker-compose build` 构建镜像
- [ ] 容器成功启动：`docker ps | grep ai-worker`
- [ ] 健康检查通过：`curl http://localhost:8004/api/v1/health`
- [ ] 外网可访问：`docker exec ai-worker curl -I https://www.baidu.com`
- [ ] OSS 可访问：`docker exec ai-worker curl -I https://220aaa.oss-cn-beijing.aliyuncs.com/test1.mp3`
- [ ] Demucs 可执行：`docker exec ai-worker demucs --version`
- [ ] WebSocket 可连接：Apifox 测试连接成功

---

## 📞 技术支持

如有问题，请查看日志：
```bash
# 实时日志
docker logs -f ai-worker --tail 200

# 错误日志
docker logs ai-worker 2>&1 | grep -i error

# 网络相关日志
docker logs ai-worker 2>&1 | grep -iE "(network|download|http|oss)"
```

---

**最后更新**: 2026-05-09
**适用版本**: AI Worker v1.0+
