# 🚀 AI Worker Docker 部署 - 快速开始

## 当前状态
✅ Docker Desktop 已运行  
✅ Docker 版本：29.2.1  
⚠️ 需要配置镜像加速器才能构建镜像

---

## 📋 部署步骤

### 步骤 1: 配置 Docker 镜像加速器（必须）

由于网络原因，需要配置国内镜像加速器。

#### 方法 A: 使用自动配置脚本（推荐）

```powershell
cd C:\Users\Lenovo\Desktop\audio-ai-platform\services\ai-worker
.\configure_docker_mirror.ps1
```

脚本会：
- ✅ 检测 Docker 状态
- ✅ 显示可用加速器
- ✅ 自动配置多个加速器
- ✅ 提示重启 Docker
- ✅ 可选：直接构建镜像

#### 方法 B: 手动配置 Docker Desktop

1. 右键 Docker 图标 → **Settings**
2. 选择 **Docker Engine**
3. 添加配置：

```json
{
  "registry-mirrors": [
    "https://hub-mirror.c.163.com",
    "https://mirror.baidubce.com",
    "https://docker.m.daocloud.io"
  ]
}
```

4. 点击 **Apply & Restart**

#### 方法 C: 获取专属加速地址

访问 [DaoCloud](https://www.daocloud.io/mirror) 获取专属加速地址。

---

### 步骤 2: 构建 Docker 镜像

配置完成后，构建镜像：

```powershell
cd C:\Users\Lenovo\Desktop\audio-ai-platform\services\ai-worker
docker build -t ai-worker:v1 .
```

**预计时间**: 10-20 分钟（首次）  
**镜像大小**: 约 8-10GB

---

### 步骤 3: 启动容器

```powershell
# 一键启动
.\deploy.ps1

# 或手动启动
docker run -d `
    --name ai-worker `
    --gpus all `
    -p 8004:8004 `
    -p 9090:9090 `
    -v ${PWD}\models:/app/models `
    -v ${PWD}\data\output:/app/data/output `
    -v ${PWD}\data\cache:/app/data/cache `
    -v ${PWD}\etc\ai-worker.yaml:/app/etc/ai-worker.yaml:ro `
    --restart unless-stopped `
    ai-worker:v1
```

---

### 步骤 4: 验证部署

```powershell
# 等待 30 秒
Start-Sleep -Seconds 30

# 检查容器
docker ps | Select-String ai-worker

# 测试健康检查
Invoke-RestMethod http://localhost:8004/api/v1/health

# 查看 GPU
docker exec ai-worker nvidia-smi
```

---

## 🔧 故障排查

### 问题 1: 构建失败 - 无法拉取镜像

**错误**: `failed to resolve source metadata`

**解决**:
```powershell
# 运行配置脚本
.\configure_docker_mirror.ps1

# 重启 Docker Desktop
# 重新构建
docker build -t ai-worker:v1 .
```

### 问题 2: GPU 不可用

**错误**: `could not select device driver`

**解决**:
1. 确保 NVIDIA 驱动已安装
2. Docker Desktop 设置中启用 WSL 2 集成
3. 重启 Docker Desktop

### 问题 3: 端口被占用

**错误**: `port 8004 is already allocated`

**解决**:
```powershell
# 查找占用进程
netstat -ano | findstr :8004

# 或使用其他端口
docker run -p 8005:8004 ... ai-worker:v1
```

---

## 📊 部署后验证

### 健康检查
```powershell
curl http://localhost:8004/api/v1/health
```

### 测试推理
```powershell
$body = @{
    audio_id = "test_001"
    model = "bsroformer"
} | ConvertTo-Json

Invoke-RestMethod `
    -Uri "http://localhost:8004/api/inference/start" `
    -Method POST `
    -ContentType "application/json" `
    -Body $body
```

### 查看日志
```powershell
docker logs -f ai-worker
```

### 查看 GPU 使用
```powershell
docker exec ai-worker nvidia-smi
```

---

## 🎯 常用命令

```powershell
# 构建镜像
docker build -t ai-worker:v1 .

# 启动容器
docker run -d --name ai-worker --gpus all -p 8004:8004 ai-worker:v1

# 查看状态
docker ps

# 查看日志
docker logs -f ai-worker

# 进入容器
docker exec -it ai-worker bash

# 停止容器
docker stop ai-worker

# 重启容器
docker restart ai-worker

# 删除容器
docker rm -f ai-worker

# 查看镜像
docker images ai-worker
```

---

## 📁 相关文件

| 文件 | 说明 |
|------|------|
| [`Dockerfile`](Dockerfile) | 生产级镜像配置 |
| [`Dockerfile.simple`](Dockerfile.simple) | 简化版镜像配置 |
| [`deploy.ps1`](deploy.ps1) | 一键部署脚本 |
| [`configure_docker_mirror.ps1`](configure_docker_mirror.ps1) | 镜像加速器配置 |
| [`DOCKER_MIRROR_CONFIG.md`](DOCKER_MIRROR_CONFIG.md) | 镜像配置详细指南 |
| [`DEPLOYMENT.md`](DEPLOYMENT.md) | 完整部署文档 |

---

## 🎉 成功标志

部署成功后，您应该看到：

✅ 容器状态：`Up`  
✅ 健康检查：`200 OK`  
✅ GPU 可用：显示 GPU 信息  
✅ 推理 API: 返回任务 ID  

---

## 📞 获取帮助

如果遇到问题：

1. 查看 [`DEPLOYMENT.md`](DEPLOYMENT.md) 故障排查章节
2. 查看 [`DOCKER_MIRROR_CONFIG.md`](DOCKER_MIRROR_CONFIG.md) 镜像配置指南
3. 查看容器日志：`docker logs ai-worker`

---

## ⏭️ 下一步

部署成功后：

1. **测试推理功能**
   - 调用 `/api/inference/start` 接口
   
2. **查看监控**
   - Prometheus: http://localhost:9090
   
3. **配置持久化**
   - 挂载模型和数据卷

4. **性能优化**
   - 参考 [`DEPLOYMENT.md`](DEPLOYMENT.md) 性能优化章节

---

**立即开始**: 运行 `.\configure_docker_mirror.ps1` 配置镜像加速器！🚀
