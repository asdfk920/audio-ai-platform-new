# Docker 部署步骤指南

## 步骤 1: 启动 Docker Desktop

### Windows
1. 打开 **Docker Desktop** 应用程序
2. 等待 Docker 完全启动（右下角图标变绿）
3. 确保启用了 WSL 2 后端

### 验证 Docker 运行
```powershell
# 打开 PowerShell，执行
docker version
```

如果看到客户端和服务端版本信息，说明 Docker 已正常运行。

---

## 步骤 2: 准备目录结构

打开 PowerShell，进入 ai-worker 目录：

```powershell
cd C:\Users\Lenovo\Desktop\audio-ai-platform\services\ai-worker

# 创建必要的目录
New-Item -ItemType Directory -Path models,data\output,data\cache,data\objects -Force
```

---

## 步骤 3: 构建 Docker 镜像

```powershell
# 在 ai-worker 目录下执行
docker build -t ai-worker:v1 .
```

**预计时间**: 10-20 分钟（首次构建需要下载基础镜像和依赖）

**镜像大小**: 约 8-10GB

---

## 步骤 4: 启动 Docker 容器

### 方式 1: 使用部署脚本（推荐）

```powershell
.\deploy.ps1
```

### 方式 2: 手动启动

```powershell
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

## 步骤 5: 验证部署

### 检查容器状态
```powershell
docker ps | Select-String ai-worker
```

### 查看日志
```powershell
docker logs -f ai-worker
```

### 测试健康检查
```powershell
# 等待 30 秒让服务启动
Start-Sleep -Seconds 30

# 测试 API
Invoke-RestMethod http://localhost:8004/api/v1/health
```

### 测试 GPU
```powershell
docker exec ai-worker nvidia-smi
```

---

## 步骤 6: 测试推理功能

```powershell
# 创建测试请求
$body = @{
    audio_id = "test_001"
    model = "bsroformer"
} | ConvertTo-Json

# 发送请求
Invoke-RestMethod `
    -Uri "http://localhost:8004/api/inference/start" `
    -Method POST `
    -ContentType "application/json" `
    -Body $body
```

---

## 常见问题

### 问题 1: Docker Desktop 无法启动

**症状**: `docker version` 显示连接错误

**解决**:
1. 重启 Docker Desktop
2. 检查 WSL 2 是否安装：`wsl --list --verbose`
3. 更新 WSL：`wsl --update`

### 问题 2: GPU 不可用

**症状**: `docker exec ai-worker nvidia-smi` 失败

**解决**:
1. 确保安装了 NVIDIA 驱动
2. 在 Docker Desktop 设置中启用 WSL 2 集成
3. 重启 Docker Desktop

### 问题 3: 端口被占用

**症状**: 容器启动失败，提示端口 8004 已被占用

**解决**:
```powershell
# 查找占用端口的进程
netstat -ano | findstr :8004

# 停止占用进程，或使用不同端口
docker run -p 8005:8004 ... ai-worker:v1
```

### 问题 4: 镜像构建失败

**症状**: `docker build` 失败

**解决**:
```powershell
# 清理 Docker 缓存
docker system prune -a

# 重新构建
docker build -t ai-worker:v1 .
```

---

## 常用命令

```powershell
# 查看容器状态
docker ps

# 查看日志
docker logs -f ai-worker

# 进入容器
docker exec -it ai-worker bash

# 停止容器
docker stop ai-worker

# 启动容器
docker start ai-worker

# 重启容器
docker restart ai-worker

# 删除容器（会丢失数据）
docker rm -f ai-worker

# 查看镜像
docker images ai-worker
```

---

## 下一步

部署成功后：
1. 访问健康检查：http://localhost:8004/api/v1/health
2. 查看监控：http://localhost:9090
3. 测试推理 API

---

## 快速参考

| 项目 | 值 |
|------|-----|
| 服务端口 | 8004 |
| 监控端口 | 9090 |
| 容器名称 | ai-worker |
| 镜像名称 | ai-worker:v1 |
| 健康检查 | http://localhost:8004/api/v1/health |
