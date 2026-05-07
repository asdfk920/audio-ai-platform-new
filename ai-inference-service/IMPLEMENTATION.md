# BSRoformer SCNet 音轨分离服务 - 完整实现

## 项目概述

这是一个完整的云端音轨分离服务，支持设备端上传音频链接进行 AI 处理。基于 BSRoformer SCNet 模型，可以将音频分离成人声和伴奏两个音轨。

## 核心功能

✅ **设备端上传链接** - 设备只需提供音频 URL，无需上传文件  
✅ **云端自动处理** - 云端自动下载、分离、上传结果  
✅ **异步任务处理** - 支持任务队列和状态跟踪  
✅ **实时进度查询** - 可随时查询任务处理进度  
✅ **回调通知** - 支持处理完成后回调通知  
✅ **GPU 加速** - 支持 NVIDIA GPU 加速推理  
✅ **任务管理** - 完整的任务生命周期管理  

## 项目结构

```
ai-inference-service/
├── server.py                    # 主服务器（完整版）
├── server_test.py              # 测试服务器（简化版）
├── server_simple.py            # 简单测试版
├── client_example.py           # 客户端调用示例
├── test_full.py                # 完整流程测试
├── requirements.txt            # 完整依赖
├── requirements_minimal.txt    # 最小化依赖
├── docker-compose.yml          # Docker 配置
├── Dockerfile                  # Docker 镜像
├── README.md                   # 项目说明
├── API_DOCUMENTATION.md        # API 文档
├── EXAMPLES.md                 # 使用示例
├── .env.example               # 环境变量示例
├── .gitignore                 # Git 忽略
├── deploy.bat                 # Windows 部署脚本
├── run_local.bat              # 本地运行脚本
├── data/                      # 数据目录
│   ├── input/                # 输入音频
│   └── output/               # 输出音频
├── models/                    # 模型文件
└── src/                       # 源代码
    ├── config.py             # 配置管理
    ├── models.py             # 数据模型
    ├── task_manager.py       # 任务管理器
    ├── service.py            # 业务服务
    ├── separator.py          # 音频分离器
    ├── utils.py              # 工具函数
    └── model_loader.py       # 模型加载器
```

## 快速开始

### 方法 1: 使用简化版测试（推荐快速验证）

```bash
# 无需安装依赖，直接运行
python server_test.py
```

### 方法 2: 使用完整版（需要安装依赖）

```bash
# 1. 创建虚拟环境
python -m venv venv
venv\Scripts\activate  # Windows
source venv/bin/activate  # Linux/Mac

# 2. 安装依赖
pip install -r requirements_minimal.txt

# 3. 启动服务
python server.py
```

### 方法 3: Docker 部署

```bash
# 构建并启动
docker-compose build
docker-compose up -d

# 查看日志
docker-compose logs -f
```

## API 使用流程

### 1. 设备端发起分离请求

```python
import requests

# 用户点击"分离音频"按钮
audio_url = "https://cdn.example.com/user_audio/song.mp3"

# 调用云端 API
response = requests.post(
    "http://your-server.com:8004/api/v1/separate/start",
    json={
        "audio_url": audio_url,
        "user_id": "user_123",
        "device_id": "device_456"
    }
)

result = response.json()
task_id = result["task_id"]
print(f"任务 ID: {task_id}")
```

### 2. 轮询查询进度

```python
import time

while True:
    status_response = requests.get(
        f"http://your-server.com:8004/api/v1/separate/status/{task_id}"
    )
    
    status_data = status_response.json()
    status = status_data["status"]
    progress = status_data["progress"]
    
    print(f"进度：{progress['percentage']}% - {progress['message']}")
    
    if status == "completed":
        print("分离完成！")
        break
    elif status == "failed":
        print(f"失败：{status_data['error_message']}")
        break
    
    time.sleep(2)  # 每 2 秒查询一次
```

### 3. 下载分离结果

```python
# 下载人声
vocals_response = requests.get(
    f"http://your-server.com:8004/api/v1/separate/download/{task_id}/vocals"
)
with open("vocals.wav", "wb") as f:
    f.write(volals_response.content)

# 下载伴奏
instrumental_response = requests.get(
    f"http://your-server.com:8004/api/v1/separate/download/{task_id}/instrumental"
)
with open("instrumental.wav", "wb") as f:
    f.write(instrumental_response.content)

print("下载完成！")
```

## 完整架构图

```
┌─────────────┐
│   设备端     │
│  (App/Web)  │
└──────┬──────┘
       │
       │ 1. 用户上传音频链接
       │
       ▼
┌─────────────────────────────────┐
│      AI 推理服务 (云端)          │
│                                 │
│  ┌─────────────────────────┐   │
│  │  API 接口层              │   │
│  │  - 开始分离              │   │
│  │  - 查询状态              │   │
│  │  - 下载结果              │   │
│  └──────────┬──────────────┘   │
│             │                  │
│  ┌──────────▼──────────────┐   │
│  │  任务管理器              │   │
│  │  - 创建任务              │   │
│  │  - 状态跟踪              │   │
│  │  - 异步处理              │   │
│  └──────────┬──────────────┘   │
│             │                  │
│  ┌──────────▼──────────────┐   │
│  │  处理流程                │   │
│  │  1. 下载音频文件         │   │
│  │  2. BSRoformer 分离      │   │
│  │  3. 上传结果             │   │
│  └──────────┬──────────────┘   │
│             │                  │
│  ┌──────────▼──────────────┐   │
│  │  BSRoformer SCNet 模型   │   │
│  │  - 人声分离              │   │
│  │  - 伴奏分离              │   │
│  └─────────────────────────┘   │
└─────────────────────────────────┘
       │
       │ 2. 返回分离后的音频
       │
       ▼
┌─────────────┐
│   设备端     │
│  播放结果    │
└─────────────┘
```

## 任务状态流程

```
pending (等待处理)
   ↓
downloading (下载音频 - 10%)
   ↓
processing (分离处理 - 30%-80%)
   ↓
uploading (上传结果 - 80%)
   ↓
completed (完成 - 100%)
```

## API 接口列表

| 接口 | 方法 | 描述 |
|------|------|------|
| `/health` | GET | 健康检查 |
| `/api/v1/separate/start` | POST | 开始分离 |
| `/api/v1/separate/status/{task_id}` | GET | 查询状态 |
| `/api/v1/separate/download/{task_id}/{stem}` | GET | 下载结果 |
| `/api/v1/separate/{task_id}/cancel` | POST | 取消任务 |
| `/api/v1/tasks` | GET | 列出任务 |

## 性能指标

### 处理时间（参考）

| 音频长度 | CPU (i7-12700K) | GPU (RTX 3060) |
|----------|-----------------|----------------|
| 3 分钟    | ~45 秒           | ~15 秒          |
| 5 分钟    | ~75 秒           | ~25 秒          |

### 并发支持

- 默认最大任务数：10
- 可配置：通过 `MAX_TASKS` 环境变量调整

## 配置说明

### 环境变量

```bash
# 服务配置
HOST=0.0.0.0
PORT=8004

# 模型配置
MODEL_NAME=bsroformer_scnet
MODEL_PATH=models

# 数据路径
DATA_PATH=data
INPUT_PATH=data/input
OUTPUT_PATH=data/output

# GPU 配置
GPU_DEVICE=0
USE_GPU=false

# 性能配置
MAX_BATCH_SIZE=1
MAX_CONCURRENT=3
MAX_TASKS=10

# 任务配置
TASK_TIMEOUT=300

# 日志配置
LOG_LEVEL=INFO
```

## 错误处理

### 常见错误

1. **下载失败**: 音频 URL 不可访问
2. **任务超时**: 处理时间超过 300 秒
3. **模型加载失败**: 模型文件不存在
4. **内存不足**: 音频文件过大

### 解决方案

- 确保音频 URL 公开可访问
- 增加 `TASK_TIMEOUT` 配置
- 下载模型文件到 `models/` 目录
- 限制音频文件大小（建议<10MB）

## 生产环境部署建议

1. **身份验证**: 添加 JWT 或其他认证机制
2. **限流**: 使用 Redis 实现请求限流
3. **监控**: 集成 Prometheus + Grafana
4. **日志**: 使用 ELK 栈集中管理日志
5. **存储**: 使用对象存储（如 S3）存储音频文件
6. **负载均衡**: 使用 Nginx 进行负载均衡
7. **自动扩缩容**: 基于 CPU/GPU 使用率自动扩缩容

## 测试

### 运行完整测试

```bash
python test_full.py
```

### 使用客户端示例

```bash
python client_example.py
```

## 相关文档

- [API 文档](API_DOCUMENTATION.md) - 详细 API 说明
- [使用示例](EXAMPLES.md) - 更多使用场景
- [项目说明](README.md) - 项目概述

## 技术栈

- **框架**: FastAPI
- **深度学习**: PyTorch
- **音频处理**: torchaudio, librosa
- **异步**: asyncio, aiohttp
- **数据验证**: Pydantic
- **日志**: loguru
- **容器化**: Docker, Docker Compose

## 许可证

MIT License

## 联系方式

如有问题，请提交 Issue 或联系开发团队。
