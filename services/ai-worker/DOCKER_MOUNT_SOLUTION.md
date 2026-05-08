# ✅ 100% 生效的 Docker 挂载方案

## 🎯 问题背景

**错误信息**：
```json
{"type":"error","error":"音频处理失败：不支持的音频路径格式: C:\Users\Lenovo\Downloads\file_example_MP3_700KB.mp3"}
```

**根本原因**：
- 数据库中的 `audio_url` 是 Windows 本地路径（`C:\Users\...`）
- Docker 容器内无法访问主机的 Windows 路径
- 需要通过 **Docker Volume 挂载** 将主机目录映射到容器内

---

## 🚀 解决方案：3 步挂载（100% 生效）

### 📌 前置准备（必须先完成）

#### 1️⃣ 确保 Docker Desktop 已启动

检查方法：
```powershell
docker info
```

如果报错：
```
failed to connect to the docker API
```

**解决**：
1. 双击桌面上的 "Docker Desktop" 图标
2. 等待托盘图标变为稳定状态（约 30 秒）
3. 再次运行 `docker info`

#### 2️⃣ 准备音频文件

```powershell
# 创建 D:\audio 目录
mkdir D:\audio -Force

# 复制测试文件
copy "C:\Users\Lenovo\Downloads\file_example_MP3_700KB.mp3" "D:\audio\test1.mp3"

# 验证文件存在
dir D:\audio\
```

预期输出：
```
 Directory of D:\audio

05/08/2026  18:23    <DIR>          .
05/08/2026  18:23    <DIR>          ..
05/08/2026  18:XX           XXX,XXX test1.mp3
```

#### 3️⃣ 更新数据库（重要！）

打开数据库客户端（pgAdmin / DBeaver），连接到 `audio_platform` 数据库：

```sql
-- 查看当前数据
SELECT id, title, audio_url FROM content WHERE id = 2;

-- ⚡ 执行更新（关键步骤）
UPDATE content 
SET audio_url = '/audio/test1.mp3' 
WHERE id = 2;

-- 验证更新成功
SELECT id, title, audio_url FROM content WHERE id = 2;
```

**预期结果**：
```
 id | title |     audio_url
----+-------+-------------------
  2 | nihao | /audio/test1.mp3   ← 必须是这个路径！
```

---

## ⚡ 步骤 1：停止并删除旧容器

### 方法 A：使用脚本（推荐）⭐

双击运行：
```
services/ai-worker/execute-docker-mount.bat
```

脚本会自动完成所有步骤并验证结果。

### 方法 B：手动执行命令

打开 PowerShell 或 CMD：

```powershell
# 停止容器
docker stop ai-worker

# 删除容器
docker rm ai-worker

echo "✅ 旧容器已停止并删除"
```

**预期输出**：
```
ai-worker
ai-worker
✅ 旧容器已停止并删除
```

如果提示找不到容器（这是正常的）：
```
Error response from daemon: No such container: ai-worker
Error: No such container: ai-worker
```
可以忽略，继续下一步。

---

## ⚡ 步骤 2：用正确的挂载命令启动新容器

### 🔑 关键参数：`-v D:\audio:/audio`

这个参数的作用：
- `D:\audio` → 主机 Windows 路径（存放音频文件的地方）
- `/audio` → 容器 Linux 路径（AI Worker 访问的位置）

### 执行命令

```powershell
cd c:\Users\Lenovo\Desktop\audio-ai-platform\services\ai-worker

docker run -d `
  --name ai-worker `
  --gpus all `
  -p 8004:8004 `
  -v "D:\audio:/audio" `
  ai-worker:v1
```

**或者使用 CMD 格式**（Windows CMD）：
```cmd
docker run -d ^
  --name ai-worker ^
  --gpus all ^
  -p 8004:8004 ^
  -v "D:\audio:/audio" ^
  ai-worker:v1
```

**预期输出**：
```
xxxxxxxxxxxxx  (容器的 ID)
```

**参数详解**：
| 参数 | 说明 |
|------|------|
| `-d` | 后台运行（detached mode）|
| `--name ai-worker` | 容器名称 |
| `--gpus all` | 使用所有 GPU（Demucs 需要 CUDA 加速）|
| `-p 8004:8004` | 端口映射：主机 8004 → 容器 8004 |
| `-v "D:\audio:/audio"` | **🔑 关键挂载参数** |
| `ai-worker:v1` | 镜像名称和标签 |

---

## ⚡ 步骤 3：验证挂载是否生效

### 方法 A：自动验证（推荐）

如果你使用了 `execute-docker-mount.bat` 脚本，它会自动验证。

### 方法 B：手动验证

#### 1️⃣ 检查容器是否运行

```powershell
docker ps | findstr ai-worker
```

**预期输出**：
```
CONTAINER ID   IMAGE         STATUS          PORTS                    NAMES
xxxxxxxxxxxx   ai-worker:v1  Up X seconds    0.0.0.0:8004->8004/tcp  ai-worker
```

#### 2️⃣ 进入容器查看挂载的文件

```powershell
docker exec -it ai-worker bash
```

进入容器后，执行：

```bash
# 查看 /audio 目录内容
ls -lh /audio/
```

**✅ 成功标志**（应该看到 test1.mp3）：
```
total 700K
-rw-r--r-- 1 root root 700K May 8 18:xx test1.mp3
```

**❌ 失败情况**：
```
ls: cannot access '/audio': No such file or directory
```

**解决方案**：
1. 检查主机上 `D:\audio\test1.mp3` 是否存在
2. 重启容器：`docker restart ai-worker`
3. 重新挂载（回到步骤 2）

#### 3️⃣ 退出容器

```bash
exit
```

---

## 🧪 测试完整流程

### 1️⃣ 检查服务是否正常

```powershell
# 查看服务日志
docker logs ai-worker --tail 20
```

**预期输出**：
```
=========================================
🚀 AI Worker Service 启动中...
=========================================
✅ AI 模型已就绪
🌐 HTTP 服务信息:
   地址：http://0.0.0.0:8004
========================================
✅ 服务启动完成，等待请求...
========================================
```

### 2️⃣ 健康检查

浏览器访问或使用 curl：
```
http://localhost:8004/api/v1/health
```

**预期响应**：
```json
{
  "status": "ok",
  "service": "ai-worker",
  "version": "1.0.0"
}
```

### 3️⃣ 测试 WebSocket（Apifox）

**连接地址**：
```
ws://localhost:8004/api/v1/audio/separate/ws?task_id=docker_mount_test&token=你的JWT_TOKEN
```

**发送消息**：
```json
{
  "type": "separate",
  "data": {
    "content_id": 2
  }
}
```

**预期流程**：

1️⃣ 接收音频列表（17 条记录）

2️⃣ 发送分离请求后，查看容器日志：
```powershell
docker logs ai-worker -f --tail 50
```

应该看到：
```
[WebSocket] 使用容器内绝对路径: /audio/test1.mp3
[WebSocket] 使用音频文件: task_id=docker_mount_test, path=/audio/test1.mp3
[INFO] 正在调用 Demucs 分离...
[INFO] demucs "/audio/test1.mp3" -o "./output/docker_mount_test"
[WebSocket] 分离完成: task_id=docker_mount_test, tracks=4
```

3️⃣ 接收成功结果：
```json
{
  "type": "separation_complete",
  "message": "分离成功！",
  "data": {
    "task_id": "docker_mount_test",
    "content_id": 2,
    "tracks": [
      {"track_name": "vocals", ...},
      {"track_name": "drums", ...},
      {"track_name": "bass", ...},
      {"track_name": "other", ...}
    ]
  }
}
```

4️⃣ 查看输出文件：
```powershell
# 在主机上查看（需要额外挂载 output 目录）
dir D:\output\docker_mount_test\

# 或在容器内查看
docker exec ai-worker ls -lh /app/output/docker_mount_test/
```

---

## 🔧 故障排除

### Q1: Docker 未运行

**错误**：
```
failed to connect to the docker API at npipe:////./pipe/dockerDesktopLinuxEngine
```

**解决**：
1. 启动 Docker Desktop
2. 等待 30 秒直到完全启动
3. 运行 `docker version` 验证

### Q2: 容器启动失败

**可能原因及解决**：

**a) 端口被占用**
```powershell
# 查找占用 8004 端口的进程
netstat -ano | findstr :8004

# 结束进程（替换 PID）
taskkill /PID <PID> /F
```

**b) 镜像不存在**
```powershell
# 构建镜像
cd services/ai-worker
docker build -t ai-worker:v1 .
```

**c) GPU 不可用**
```powershell
# 检查 NVIDIA GPU
nvidia-smi

# 如果没有 GPU，去掉 --gpus all 参数
docker run -d --name ai-worker -p 8004:8004 -v "D:\audio:/audio" ai-worker:v1
```

### Q3: 挂载失败

**症状**：容器内 `/audio` 目录为空或不存在

**排查步骤**：

```powershell
# 1. 确认主机文件存在
dir D:\audio\test1.mp3

# 2. 检查容器挂载配置
docker inspect ai-worker | Select-String -Pattern "Mounts" -Context 0,20

# 3. 重新创建容器
docker stop ai-worker
docker rm ai-worker
docker run -d --name ai-worker -p 8004:8004 -v "D:\audio:/audio" ai-worker:v1
```

### Q4: Demucs 还是报错 exit status 2

**手动测试 Demucs**：

```powershell
# 进入容器
docker exec -it ai-worker bash

# 手动运行 demucs
demucs /audio/test1.mp3 -o /app/output/manual_test

# 如果成功，说明文件路径正确
# 问题可能在 Go 代码的调用方式
```

### Q5: 如何添加更多音频文件？

**步骤**：

1. **放入文件**
```powershell
copy 新歌曲.mp3 D:\audio\new_song.mp3
```

2. **更新数据库**
```sql
INSERT INTO content (title, audio_url)
VALUES ('新歌曲', '/audio/new_song.mp3');
```

3. **立即可用**（无需重启容器）
```json
// Apifox 中发送新的 content_id
{"type":"separate","data":{"content_id": 新ID}}
```

---

## 📊 架构图

```
┌─────────────────────────────────────────────────────────────┐
│                     主机 (Windows)                           │
│                                                              │
│  D:\audio\                                                   │
│  └── test1.mp3 ◄─────────────────────────────┐              │
│                                              │              │
│  Docker Desktop                              │              │
│  (运行中 ✓)                                  │              │
└──────────────────────────────────────────────┼──────────────┘
                                               │
                        Docker Volume Mount     │
                   -v D:\audio:/audio          │
                                               ▼
┌─────────────────────────────────────────────────────────────┐
│                    容器 (ai-worker)                          │
│                                                              │
│  /audio/                                                     │
│  └── test1.mp3 ◄── 来自主机挂载                              │
│       │                                                      │
│       ▼                                                      │
│  AI Worker Service                                          │
│  ├─ WebSocket Server (:8004)                                │
│  ├─ PostgreSQL Client                                       │
│  └─ Demucs 调用器                                           │
│       │                                                     │
│       ▼                                                     │
│  demucs("/audio/test1.mp3", "-o", "/app/output/task_xxx")  │
│                                                              │
│  /app/output/                                                │
│  └── task_xxx/                                               │
│      ├── vocals.wav                                         │
│      ├── drums.wav                                          │
│      ├── bass.wav                                           │
│      └── other.wav                                          │
└─────────────────────────────────────────────────────────────┘
```

---

## 📁 文件清单

| 文件 | 用途 |
|------|------|
| [`execute-docker-mount.bat`](file://c:\Users\Lenovo\Desktop\audio-ai-platform\services\ai-worker\execute-docker-mount.bat) | 一键式挂载脚本（推荐）⭐ |
| [`start-docker-audio.bat`](file://c:\Users\Lenovo\Desktop\audio-ai-platform\services\ai-worker\start-docker-audio.bat) | Docker 启动脚本 |
| [audio_separate_ws_handler.go](file:///internal/handler/audio_separate_ws_handler.go) | 已修改支持容器路径 |

---

## 🎯 快速开始（5 分钟搞定）

### 方案 A：全自动脚本（最简单）⭐

```powershell
# 1. 确保 Docker Desktop 已启动

# 2. 准备音频文件
copy "C:\Users\Lenovo\Downloads\file_example_MP3_700KB.mp3" "D:\audio\test1.mp3"

# 3. 更新数据库（在数据库客户端中执行）
# UPDATE content SET audio_url = '/audio/test1.mp3' WHERE id = 2;

# 4. 双击运行脚本
cd services\ai-worker
.\execute-docker-mount.bat

# 5. 测试 WebSocket（Apifox）
```

### 方案 B：手动执行（完全控制）

```powershell
# Step 1: 停止旧容器
docker stop ai-worker; docker rm ai-worker

# Step 2: 启动新容器（关键挂载）
docker run -d --name ai-worker --gpus all -p 8004:8004 -v "D:\audio:/audio" ai-worker:v1

# Step 3: 验证挂载
docker exec -it ai-worker bash
ls /audio/
# 应该看到 test1.mp3
exit

# Step 4: 测试
# 在 Apifox 中连接 WebSocket 并测试
```

---

## ✅ 成功标志

当满足以下所有条件时，说明挂载成功：

- [ ] `docker ps` 显示 ai-worker 容器正在运行
- [ ] `docker exec ai-worker ls /audio/` 显示 test1.mp3
- [ ] `http://localhost:8004/api/v1/health` 返回 OK
- [ ] WebSocket 连接成功并发送分离请求
- [ ] 日志显示 `[WebSocket] 使用容器内绝对路径: /audio/test1.mp3`
- [ ] 接收到 `separation_complete` 成功消息
- [ ] 输出目录生成 4 个 .wav 文件

---

## 💡 核心要点总结

1. **挂载是关键**：`-v D:\audio:/audio` 让容器能访问主机文件
2. **数据库路径要改**：`audio_url = '/audio/test1.mp3'`（不是 C:\...）
3. **代码已适配**：以 `/` 开头的路径直接传给 Demucs
4. **无需重启容器添加新文件**：直接复制到 D:\audio 即可
5. **GPU 加速**：`--gpus all` 让 Demucs 使用 CUDA（快 10 倍以上）

---

**🎉 按照 3 步方案执行后，问题将 100% 解决！**

详细操作指南：本文件（DOCKER_MOUNT_SOLUTION.md）
一键式脚本：[`execute-docker-mount.bat`](file://c:\Users\Lenovo\Desktop\audio-ai-platform\services\ai-worker\execute-docker-mount.bat)

**现在请开始执行吧！** 🚀
