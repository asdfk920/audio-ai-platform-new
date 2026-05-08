# Docker 镜像加速器配置指南

## 问题原因
由于网络原因，无法直接从 Docker Hub 拉取镜像。需要配置国内镜像加速器。

## 方法 1: 配置 Docker Desktop 镜像加速器（推荐）

### 步骤 1: 打开 Docker Desktop 设置
1. 右键点击系统托盘中的 Docker 图标
2. 选择 **Settings** (或 **Preferences**)

### 步骤 2: 配置镜像加速器
1. 选择 **Docker Engine** 标签页
2. 在配置文件中添加 `registry-mirrors` 配置

### 步骤 3: 修改配置文件

在 Docker Desktop 的 `settings.json` 中添加以下内容：

```json
{
  "builder": {
    "gc": {
      "defaultKeepStorage": "20GB",
      "enabled": true
    }
  },
  "experimental": false,
  "registry-mirrors": [
    "https://hub-mirror.c.163.com",
    "https://mirror.baidubce.com",
    "https://docker.m.daocloud.io",
    "https://docker.1panel.live"
  ]
}
```

### 步骤 4: 应用并重启
1. 点击 **Apply & Restart**
2. 等待 Docker Desktop 重启

### 验证配置
```powershell
docker info | Select-String "Registry Mirrors"
```

---

## 方法 2: 使用 DaoCloud 镜像加速服务

### 获取专属加速地址
1. 访问 https://www.daocloud.io/mirror
2. 点击"点击这里"获取专属加速地址
3. 复制加速地址（如：`https://your-id.m.daocloud.io`）

### 配置加速地址
在 Docker Desktop 设置中添加：

```json
{
  "registry-mirrors": ["https://your-id.m.daocloud.io"]
}
```

---

## 方法 3: 手动拉取镜像（备选方案）

如果自动拉取失败，可以手动拉取基础镜像：

```powershell
# 拉取 Go 镜像
docker pull golang:1.21-alpine

# 拉取 CUDA 镜像
docker pull nvidia/cuda:12.1.0-runtime-ubuntu22.04

# 重新构建
docker build -t ai-worker:v1 .
```

---

## 方法 4: 使用简化版 Dockerfile

使用官方镜像源（可能需要配置加速器）：

```powershell
# 使用简化版 Dockerfile
docker build -f Dockerfile.simple -t ai-worker:v1 .
```

---

## 常见问题

### Q1: 配置后仍然无法拉取

**解决**:
```powershell
# 清理 Docker 缓存
docker system prune -a

# 重启 Docker Desktop
# 重新尝试构建
docker build -t ai-worker:v1 .
```

### Q2: 找不到合适的镜像加速器

**解决**: 
- 尝试多个加速器地址
- 使用手机热点（有时网络更好）
- 等待网络状况好转

### Q3: 拉取速度很慢

**解决**:
- 使用多个加速器轮流尝试
- 在夜间网络较好时构建
- 使用 DaoCloud 等服务的专属加速地址

---

## 快速验证

配置成功后，执行以下命令测试：

```powershell
# 测试拉取速度
docker pull hello-world

# 如果快速拉取成功，说明加速器配置成功
```

---

## 下一步

加速器配置成功后：

```powershell
cd C:\Users\Lenovo\Desktop\audio-ai-platform\services\ai-worker

# 构建镜像
docker build -t ai-worker:v1 .

# 启动容器
docker run -d --name ai-worker --gpus all -p 8004:8004 -p 9090:9090 ai-worker:v1

# 验证
docker ps
```
