# Docker 部署 - 简单步骤

## 步骤 1: 配置 Docker 镜像加速器

### 方法 A: 手动修改配置文件（推荐）

1. 打开文件资源管理器，导航到：
   ```
   C:\Users\Lenovo\.docker\desktop\settings.json
   ```

2. 用记事本打开 `settings.json`

3. 添加或修改 `registry-mirrors` 部分：

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
    "https://docker.m.daocloud.io"
  ]
}
```

4. 保存文件

5. 重启 Docker Desktop：
   - 右键系统托盘 Docker 图标
   - 选择 **Restart Docker Desktop**

### 方法 B: 使用 Docker Desktop 设置

1. 右键 Docker 图标 → **Settings**
2. 选择 **Docker Engine**
3. 在 JSON 配置中添加：

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

---

## 步骤 2: 验证配置

打开 PowerShell，执行：

```powershell
docker info | Select-String "Registry Mirrors"
```

如果看到配置的镜像地址，说明配置成功。

---

## 步骤 3: 构建 Docker 镜像

```powershell
cd C:\Users\Lenovo\Desktop\audio-ai-platform\services\ai-worker
docker build -t ai-worker:v1 .
```

**预计时间**: 10-20 分钟

---

## 步骤 4: 启动容器

```powershell
docker run -d --name ai-worker --gpus all -p 8004:8004 -p 9090:9099 -v ${PWD}\models:/app/models -v ${PWD}\data\output:/app/data/output -v ${PWD}\data\cache:/app/data/cache -v ${PWD}\etc\ai-worker.yaml:/app/etc/ai-worker.yaml:ro --restart unless-stopped ai-worker:v1
```

---

## 步骤 5: 验证部署

```powershell
# 等待 30 秒
Start-Sleep -Seconds 30

# 查看容器状态
docker ps | Select-String ai-worker

# 测试健康检查
Invoke-RestMethod http://localhost:8004/api/v1/health

# 查看 GPU
docker exec ai-worker nvidia-smi
```

---

## 快速参考

### 如果构建失败

```powershell
# 清理 Docker 缓存
docker system prune -a

# 重启 Docker Desktop
# 重新构建
docker build -t ai-worker:v1 .
```

### 常用命令

```powershell
# 查看日志
docker logs -f ai-worker

# 停止容器
docker stop ai-worker

# 重启容器
docker restart ai-worker

# 删除容器
docker rm -f ai-worker
```

---

## 成功标志

✅ `docker ps` 显示容器状态为 `Up`  
✅ 健康检查返回 `200 OK`  
✅ `nvidia-smi` 显示 GPU 信息  

---

## 下一步

部署成功后：

1. 访问：http://localhost:8004/api/v1/health
2. 测试推理 API
3. 查看监控：http://localhost:9090
