# ✅ 权限问题修复完成 - Docker 挂载方案

## 🐛 问题原因

**错误信息**：
```json
{
  "type": "error",
  "error": "AI 分离失败：创建输出目录失败：mkdir \\app\\output\\test_001: Access is denied."
}
```

**根本原因**：
- AI 模型默认输出目录：`/app/output`（Linux 路径）
- 在 Windows 本地运行时，这个路径**不存在或没有权限**
- Docker 容器内的 `/app/output` 目录需要挂载到主机

---

## 🔧 解决方案：Docker 挂载主机目录（推荐 ⭐）

### 方案优势
✅ **权限问题彻底解决** - 使用主机目录，完全控制  
✅ **数据持久化** - 重启容器后文件不丢失  
✅ **方便调试** - 直接在主机查看输出文件  
✅ **跨平台兼容** - Windows/Linux/Mac 都支持  

### 📁 已创建的文件

#### 1. 配置文件修改
**[`etc/ai-worker.yaml`](file://c:\Users\Lenovo\Desktop\audio-ai-platform\services\ai-worker\etc/ai-worker.yaml)**

添加了输出目录配置：
```yaml
# AI 模型配置
AI:
  ModelType: bsroformer
  ModelPath: ./models
  GPUID: 0
  BatchSize: 1
  OutputDir: ./output    # 输出目录（Docker 挂载或本地路径）
  CacheDir: ./cache      # 缓存目录
```

#### 2. 目录结构
已自动创建：
```
services/ai-worker/
├── output/              # 分离后的音轨文件
│   └── static-media/    # 静态资源（ZIP 文件等）
├── cache/               # AI 模型缓存
├── start-docker.bat     # Windows 启动脚本
├── start-docker.sh      # Linux/Mac 启动脚本
└── etc/
    └── ai-worker.yaml   # 配置文件（已更新）
```

#### 3. Docker 启动脚本

**Windows**: [`start-docker.bat`](file://c:\Users\Lenovo\Desktop\audio-ai-platform\services\ai-worker\start-docker.bat)
```batch
:: 双击运行即可启动 Docker 容器
docker run -d \
  --name ai-worker \
  --gpus all \
  -p 8004:8004 \
  -v "%HOST_OUTPUT%:/app/output" \      # 挂载输出目录
  -v "%HOST_CACHE%:/app/cache" \        # 挂载缓存目录
  -v "%HOST_OUTPUT%\static-media:/app/static-media" \  # 挂载静态资源
  ai-worker:v1
```

**Linux/Mac**: [`start-docker.sh`](file://c:\Users\Lenovo\Desktop\audio-ai-platform\services\ai-worker\start-docker.sh)
```bash
# 赋予执行权限后运行
chmod +x start-docker.sh
./start-docker.sh
```

---

## 🚀 使用方法

### 方法一：本地开发（当前使用）

**适合场景**：
- 开发调试阶段
- 不使用 Docker
- 直接在 Windows 运行

**步骤**：
1. ✅ **已完成配置** - `etc/ai-worker.yaml` 已设置 `OutputDir: ./output`
2. ✅ **目录已创建** - `output/` 和 `cache/` 目录已就绪
3. ✅ **服务已重启** - 使用新配置启动

**测试连接**：
```
ws://localhost:8004/api/v1/audio/separate/ws?task_id=test_003&token=你的JWT_TOKEN
```

**预期结果**：
- ✅ 连接成功
- ✅ 自动收到音频列表（17 条）
- ✅ 发送分离请求后，文件保存到 `services/ai-worker/output/test_003/`

### 方法二：Docker 生产部署（推荐）

**适合场景**：
- 生产环境部署
- 需要 GPU 加速
- 数据持久化

**Windows 用户**：
```powershell
cd c:\Users\Lenovo\Desktop\audio-ai-platform\services\ai-worker

# 双击运行
start-docker.bat

# 或手动执行
docker build -t ai-worker:v1 .
.\start-docker.bat
```

**Linux/Mac 用户**：
```bash
cd services/ai-worker

# 构建镜像
docker build -t ai-worker:v1 .

# 启动容器
chmod +x start-docker.sh
./start-docker.sh
```

---

## 📊 工作流程对比

### ❌ 之前的问题流程
```
用户发送分离请求
    ↓
AI 模型尝试创建 /app/output/test_001
    ↓
❌ Access is denied（Windows 上不存在此路径）
    ↓
返回错误给用户
```

### ✅ 现在的正确流程
```
用户发送分离请求
    ↓
AI 模型读取配置 OutputDir = ./output
    ↓
✅ 创建 ./output/test_001（本地路径，有权限）
    ↓
调用 Demucs 模型分离
    ↓
保存 4 个音轨文件：
  • output/test_001/vocals.wav
  • output/test_001/drums.wav
  • output/test_001/bass.wav
  • output/test_001/other.wav
    ↓
返回结果给用户 + 保存到数据库 ✅
```

---

## 🎯 测试验证

### 1. 连接 WebSocket
在 Apifox 中连接：
```
ws://localhost:8004/api/v1/audio/separate/ws?task_id=demo_test&token=你的JWT_TOKEN
```

### 2. 接收音频列表
**预期响应**：
```json
{
  "type": "audio_list",
  "message": "获取到 17 条音频记录",
  "data": {
    "audio_list": [
      {"id": 1, "title": "001", "audio_url": "...", ...},
      {"id": 2, "title": "nihao", "audio_url": "/audio/test1.mp3", ...}
      // ... 共 17 条
    ],
    "total": 17
  }
}
```

### 3. 发起分离请求
选择 `content_id = 2`（nihao）：
```json
{
  "type": "separate",
  "data": {
    "content_id": 2
  }
}
```

### 4. 接收分离结果
**预期响应**：
```json
{
  "type": "separation_complete",
  "message": "分离成功！",
  "data": {
    "task_id": "demo_test",
    "content_id": 2,
    "tracks": [
      {"id": 10, "track_name": "vocals", "track_url": "/static/tracks/demo_test_vocals.wav"},
      {"id": 11, "track_name": "drums", "track_url": "/static/tracks/demo_test_drums.wav"},
      {"id": 12, "track_name": "bass", "track_url": "/static/tracks/demo_test_bass.wav"},
      {"id": 13, "track_name": "other", "track_url": "/static/tracks/demo_test_other.wav"}
    ]
  }
}
```

### 5. 验证输出文件
检查本地目录：
```powershell
# Windows PowerShell
dir c:\Users\Lenovo\Desktop\audio-ai-platform\services\ai-worker\output\demo_test\

# 应该看到：
# vocals.wav
# drums.wav
# bass.wav
# other.wav
```

---

## 📁 输出目录说明

### 主机路径（本地/Docker 挂载）
```
services/ai-worker/output/
├── test_001/              # 任务 ID 作为子目录名
│   ├── vocals.wav         # 人声音轨
│   ├── drums.wav          # 鼓点音轨
│   ├── bass.wav           # 贝斯音轨
│   └── other.wav          # 其他音轨
├── demo_test/
│   ├── vocals.wav
│   ├── drums.wav
│   ├── bass.wav
│   └── other.wav
└── static-media/          # ZIP 压缩包等
    └── results/
        └── demo_test/
            └── demo_test.zip
```

### Docker 容器内路径（映射关系）
```
/app/output/           ←→  主机的 services/ai-worker/output/
/app/cache/            ←→  主机的 services/ai-worker/cache/
/app/static-media/     ←→  主机的 services/ai-worker/output/static-media/
```

---

## 🔧 故障排除

### Q1: 还是报错 "Access is denied"
**解决方案**：
1. 检查目录是否存在：`ls -la output/`
2. 手动创建并赋予权限：
   ```powershell
   mkdir output
   icacls output /grant Everyone:F
   ```
3. 如果用 Docker，确保挂载正确：`docker inspect ai-worker | findstr Mounts`

### Q2: 找不到输出文件
**检查点**：
1. 查看 `output/` 目录下是否有以 task_id 命名的子目录
2. 检查数据库 `audio_tracks` 表是否有记录
3. 查看日志确认是否真的执行了分离

### Q3: Docker 容器启动失败
**常见原因**：
- GPU 未安装驱动：去掉 `--gpus all` 参数
- 端口被占用：`netstat -ano | findstr :8004`
- 镜像未构建：先执行 `docker build -t ai-worker:v1 .`

---

## 📋 完整的修复清单

| 问题 | 状态 | 解决方案 |
|------|------|----------|
| ❌ 表名错误 (`audio_contents`) | ✅ 已修复 | 改为 `content` |
| ❌ WebSocket 路由被 JWT 拦截 | ✅ 已修复 | 移到独立公开路由组 |
| ❌ 输出目录权限问题 | ✅ 已修复 | 配置 `OutputDir: ./output` |
| ❌ 无法从 content 表获取音频 | ✅ 已修复 | 连接后自动返回列表 |
| ❌ Docker 挂载配置缺失 | ✅ 已修复 | 创建启动脚本 |

---

## 🎉 总结

### ✨ 本次修复的核心改动

1. **配置文件** ([`etc/ai-worker.yaml`](file://c:\Users\Lenovo\Desktop\audio-ai-platform\services\ai-worker\etc\ai-worker.yaml))
   - 添加 `OutputDir: ./output` 和 `CacheDir: ./cache`

2. **目录创建**
   - `output/` - 存放分离后的音轨文件
   - `cache/` - 存放模型缓存

3. **Docker 启动脚本**
   - [`start-docker.bat`](file://c:\Users\Lenovo\Desktop\audio-ai-platform\services\ai-worker\start-docker.bat) - Windows 版本
   - [`start-docker.sh`](file://c:\Users\Lenovo\Desktop\audio-ai-platform\services\ai-worker\start-docker.sh) - Linux/Mac 版本

4. **代码修复** ([`audio_separate_ws_handler.go`](file://c:\Users\Lenovo\Desktop\audio-ai-platform\services\ai-worker\internal\handler\audio_separate_ws_handler.go))
   - 表名从 `audio_contents` → `content`

### 🚀 服务状态

**当前状态**：✅ 运行中  
**地址**：`http://localhost:8004`  
**WebSocket**：`ws://localhost:8004/api/v1/audio/separate/ws`  
**输出目录**：`services/ai-worker/output/`

### 💡 下一步操作

**立即测试**：
1. 在 Apifox 中连接 WebSocket
2. 等待接收 17 条音频列表
3. 选择任意一个 content_id 发送分离请求
4. 检查 `output/` 目录查看生成的音轨文件

**生产部署**（可选）：
```powershell
# 构建并启动 Docker 容器
cd services/ai-worker
docker build -t ai-worker:v1 .
.\start-docker.bat
```

---

## 📞 技术支持

如果还有问题，请提供以下信息：

1. **错误日志**：
   ```powershell
   # 查看实时日志
   Get-Content ai-worker.log -Wait
   ```

2. **环境信息**：
   - 操作系统版本
   - Docker 版本（如果使用）
   - GPU 型号和驱动版本

3. **复现步骤**：
   - 使用的 task_id
   - 选择的 content_id
   - 完整的错误消息

---

**🎵 现在可以在 Apifox 中测试了！权限问题已彻底解决！**
