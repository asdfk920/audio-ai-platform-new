# AI 音轨分离服务 API 文档

## 概述

这是一个基于 BSRoformer SCNet 模型的云端音轨分离服务。设备端可以通过上传音频链接到云端，由云端调用大模型进行音轨分离处理。

## 服务地址

- **本地测试**: `http://localhost:8004`
- **云端部署**: `http://your-server.com:8004`

## 快速开始

### 1. 启动服务

```bash
# 使用简化版测试（无需依赖）
python server_test.py

# 使用完整版（需要安装依赖）
python server.py
```

### 2. 验证服务

```bash
curl http://localhost:8004/health
```

## API 接口

### 1. 健康检查

**接口**: `GET /health`

**响应示例**:
```json
{
  "status": "healthy",
  "model_name": "bsroformer_scnet",
  "device": "cpu",
  "gpu_available": false,
  "tasks": {
    "total": 5,
    "pending": 0,
    "processing": 2,
    "completed": 3,
    "failed": 0
  }
}
```

### 2. 开始音轨分离 ⭐

**接口**: `POST /api/v1/separate/start`

**描述**: 设备端调用此接口，上传音频链接到云端处理

**请求参数**:
```json
{
  "audio_url": "https://example.com/audio/song.mp3",
  "user_id": "user_123",
  "device_id": "device_456",
  "callback_url": "https://your-server.com/callback"
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| audio_url | string | 是 | 音频文件 URL |
| user_id | string | 否 | 用户 ID |
| device_id | string | 否 | 设备 ID |
| callback_url | string | 否 | 回调 URL（用于上传结果） |

**响应示例**:
```json
{
  "success": true,
  "task_id": "task_20260507180530_a1b2c3d4",
  "message": "任务创建成功，正在处理",
  "data": {
    "task_id": "task_20260507180530_a1b2c3d4",
    "user_id": "user_123",
    "device_id": "device_456",
    "audio_url": "https://example.com/audio/song.mp3",
    "status": "pending",
    "progress": {
      "percentage": 0,
      "stage": "pending",
      "message": "等待处理"
    },
    "created_at": "2026-05-07T18:05:30.123456"
  }
}
```

### 3. 查询任务状态

**接口**: `GET /api/v1/separate/status/{task_id}`

**描述**: 查询音轨分离任务的进度

**路径参数**:
- `task_id`: 任务 ID

**响应示例**:

**处理中**:
```json
{
  "success": true,
  "task_id": "task_20260507180530_a1b2c3d4",
  "status": "processing",
  "progress": {
    "percentage": 50,
    "stage": "processing",
    "message": "正在分离音轨"
  },
  "output_files": null,
  "error_message": null,
  "processing_time": null
}
```

**已完成**:
```json
{
  "success": true,
  "task_id": "task_20260507180530_a1b2c3d4",
  "status": "completed",
  "progress": {
    "percentage": 100,
    "stage": "completed",
    "message": "分离完成"
  },
  "output_files": {
    "vocals": "data/output/task_20260507180530_a1b2c3d4/task_20260507180530_a1b2c3d4_vocals.wav",
    "instrumental": "data/output/task_20260507180530_a1b2c3d4/task_20260507180530_a1b2c3d4_instrumental.wav"
  },
  "error_message": null,
  "processing_time": 12.34
}
```

**失败**:
```json
{
  "success": true,
  "task_id": "task_20260507180530_a1b2c3d4",
  "status": "failed",
  "progress": {
    "percentage": 0,
    "stage": "failed",
    "message": "下载失败"
  },
  "output_files": null,
  "error_message": "下载音频文件失败：https://example.com/audio/song.mp3",
  "processing_time": 5.67
}
```

### 4. 下载分离结果

**接口**: `GET /api/v1/separate/download/{task_id}/{stem}`

**描述**: 下载分离后的音频文件

**路径参数**:
- `task_id`: 任务 ID
- `stem`: 音轨类型
  - `vocals`: 人声
  - `instrumental`: 伴奏

**响应**: 音频文件（WAV 格式）

**示例**:
```bash
# 下载人声
curl http://localhost:8004/api/v1/separate/download/task_123/vocals \
  -o vocals.wav

# 下载伴奏
curl http://localhost:8004/api/v1/separate/download/task_123/instrumental \
  -o instrumental.wav
```

### 5. 取消任务

**接口**: `POST /api/v1/separate/{task_id}/cancel`

**描述**: 取消正在进行的任务

**响应示例**:
```json
{
  "success": true,
  "message": "任务已取消"
}
```

### 6. 列出所有任务

**接口**: `GET /api/v1/tasks`

**查询参数**:
- `status`: 按状态筛选（可选）
  - `pending`: 等待处理
  - `downloading`: 下载中
  - `processing`: 处理中
  - `uploading`: 上传中
  - `completed`: 已完成
  - `failed`: 失败

**响应示例**:
```json
{
  "success": true,
  "count": 3,
  "tasks": [
    {
      "task_id": "task_001",
      "status": "completed",
      "created_at": "2026-05-07T18:00:00"
    },
    ...
  ]
}
```

## 完整流程示例

### 设备端调用流程

```python
import requests
import time

# 1. 用户点击分离按钮，设备上传音频链接
audio_url = "https://cdn.example.com/user_audio/song.mp3"

# 2. 调用开始分离接口
response = requests.post(
    "http://localhost:8004/api/v1/separate/start",
    json={
        "audio_url": audio_url,
        "user_id": "user_123",
        "device_id": "device_456"
    }
)

result = response.json()
task_id = result["task_id"]
print(f"任务 ID: {task_id}")

# 3. 轮询查询状态
while True:
    status_response = requests.get(
        f"http://localhost:8004/api/v1/separate/status/{task_id}"
    )
    
    status_data = status_response.json()
    status = status_data["status"]
    progress = status_data["progress"]
    
    print(f"状态：{status}, 进度：{progress['percentage']}%")
    
    if status == "completed":
        print("分离完成！")
        
        # 4. 下载结果
        output_files = status_data["output_files"]
        
        # 下载人声
        vocals_response = requests.get(
            f"http://localhost:8004/api/v1/separate/download/{task_id}/vocals"
        )
        with open("vocals.wav", "wb") as f:
            f.write(volals_response.content)
        
        # 下载伴奏
        instrumental_response = requests.get(
            f"http://localhost:8004/api/v1/separate/download/{task_id}/instrumental"
        )
        with open("instrumental.wav", "wb") as f:
            f.write(instrumental_response.content)
        
        break
    
    elif status == "failed":
        print(f"分离失败：{status_data['error_message']}")
        break
    
    time.sleep(2)  # 每 2 秒查询一次
```

### 使用客户端库

```python
from client_example import AudioSeparationClient

# 创建客户端
client = AudioSeparationClient("http://localhost:8004")

# 开始分离
response = client.start_separation(
    audio_url="https://example.com/audio/song.mp3",
    user_id="user_123",
    device_id="device_456"
)

task_id = response["task_id"]
print(f"任务 ID: {task_id}")

# 等待完成（自动轮询）
final_status = client.wait_for_completion(task_id, timeout=300)

if final_status["status"] == "completed":
    # 下载结果
    client.download_result(task_id, "vocals", "vocals.wav")
    client.download_result(task_id, "instrumental", "instrumental.wav")
    print("下载完成！")
```

## 任务状态说明

| 状态 | 说明 | 进度 |
|------|------|------|
| pending | 等待处理 | 0% |
| downloading | 正在下载音频文件 | 10% |
| processing | 正在处理音频 | 30%-80% |
| uploading | 正在上传结果 | 80% |
| completed | 分离完成 | 100% |
| failed | 处理失败 | 0% |
| timeout | 任务超时 | 0% |
| cancelled | 任务已取消 | 0% |

## 错误处理

### 常见错误码

| 状态码 | 说明 |
|--------|------|
| 200 | 成功 |
| 400 | 请求参数错误 |
| 404 | 任务不存在或资源未找到 |
| 500 | 服务器内部错误 |

### 错误响应格式

```json
{
  "success": false,
  "error": "错误描述信息"
}
```

## 性能指标

### 处理时间参考

| 音频长度 | CPU (i7) | GPU (RTX 3060) |
|----------|----------|----------------|
| 3 分钟    | ~45 秒    | ~15 秒          |
| 5 分钟    | ~75 秒    | ~25 秒          |

*实际时间取决于音频质量和硬件配置*

## 部署说明

### 本地开发

```bash
# 1. 安装依赖
pip install torch torchaudio librosa soundfile fastapi uvicorn aiohttp aiofiles loguru

# 2. 启动服务
python server.py

# 3. 访问文档
# http://localhost:8004/docs
```

### Docker 部署

```bash
# 1. 构建镜像
docker-compose build

# 2. 启动服务
docker-compose up -d

# 3. 查看日志
docker-compose logs -f
```

## 支持的音频格式

- MP3 (.mp3)
- WAV (.wav)
- FLAC (.flac)
- M4A (.m4a)
- OGG (.ogg)

## 输出说明

分离后生成两个音轨文件：
- `{task_id}_vocals.wav` - 人声音轨
- `{task_id}_instrumental.wav` - 伴奏音轨

所有输出文件均为 WAV 格式，44.1kHz 采样率，16 位深度。

## 安全建议

1. **生产环境**：添加身份验证
2. **限流**：限制每个用户的请求频率
3. **文件大小**：限制上传音频文件的最大大小
4. **超时**：设置合理的任务超时时间
5. **日志**：记录所有操作日志

## 常见问题

### Q: 任务一直处于 pending 状态？
A: 检查服务器是否正常运行，查看日志了解是否有错误。

### Q: 下载音频失败？
A: 确保 audio_url 是公开可访问的 URL。

### Q: 如何优化处理速度？
A: 使用 GPU 加速，调整 MAX_BATCH_SIZE 参数。

## 联系方式

如有问题，请提交 Issue 或联系开发团队。
