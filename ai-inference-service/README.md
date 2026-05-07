# BSRoformer SCNet 音轨分离推理服务

基于 BSRoformer SCNet 模型的云端音轨分离服务，支持 GPU 加速和 Docker 部署。

## 功能特性

- ✅ **BSRoformer SCNet 模型**：先进的音轨分离算法
- ✅ **GPU 加速**：支持 NVIDIA CUDA 11.8
- ✅ **Docker 部署**：一键启动，开箱即用
- ✅ **RESTful API**：简单易用的 HTTP 接口
- ✅ **批量处理**：支持并发请求
- ✅ **健康检查**：实时监控服务状态

## 系统要求

- **GPU**: NVIDIA 显卡（推荐 RTX 3060 或更高）
- **CUDA**: 11.8+
- **Docker**: 20.10+
- **Docker Compose**: 2.0+
- **内存**: 至少 8GB（推荐 16GB）
- **存储**: 至少 10GB 可用空间

## 快速开始

### 1. 克隆项目

```bash
cd ai-inference-service
```

### 2. 下载模型（可选）

服务会自动从 HuggingFace 下载模型，也可以手动下载：

```bash
# 创建模型目录
mkdir -p models

# 下载 BSRoformer SCNet 模型
# 方法 1: 使用 huggingface-cli
pip install huggingface_hub
huggingface-cli download tsurutsu/bsroformer_scnet --local-dir models

# 方法 2: 手动下载后放入 models 目录
```

### 3. 配置环境变量

```bash
cp .env.example .env
# 编辑 .env 文件，根据需要调整配置
```

### 4. 启动服务

```bash
# 构建镜像
docker-compose build

# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f
```

### 5. 验证服务

```bash
# 健康检查
curl http://localhost:8004/health

# 预期响应
{
  "status": "healthy",
  "model_name": "bsroformer_scnet",
  "device": "cuda:0",
  "gpu_available": true
}
```

## API 接口

### 1. 健康检查

```http
GET /health
```

**响应示例:**
```json
{
  "status": "healthy",
  "model_name": "bsroformer_scnet",
  "device": "cuda:0",
  "gpu_available": true
}
```

### 2. 音轨分离

```http
POST /api/v1/separate
Content-Type: multipart/form-data

参数:
- file: 音频文件 (mp3, wav, flac, m4a, ogg)
```

**请求示例:**
```bash
curl -X POST http://localhost:8004/api/v1/separate \
  -F "file=@song.mp3"
```

**响应示例:**
```json
{
  "success": true,
  "message": "音轨分离成功",
  "output_files": {
    "vocals": "/app/data/output/song_vocals.wav",
    "instrumental": "/app/data/output/song_instrumental.wav"
  },
  "processing_time": 12.34
}
```

### 3. 下载分离结果

```http
GET /api/v1/download/{file_path}
```

**请求示例:**
```bash
curl http://localhost:8004/api/v1/download/output/song_vocals.wav \
  -o vocals.wav
```

## 项目结构

```
ai-inference-service/
├── docker-compose.yml      # Docker Compose 配置
├── Dockerfile             # Docker 镜像构建文件
├── requirements.txt       # Python 依赖
├── server.py             # FastAPI 服务器
├── .env.example          # 环境变量示例
├── models/               # 模型文件目录
├── data/                 # 数据目录
│   ├── input/           # 输入音频
│   └── output/          # 输出音频
└── src/                  # 源代码
    ├── config.py        # 配置管理
    ├── model_loader.py  # 模型加载器
    ├── separator.py     # 音频分离器
    └── utils.py         # 工具函数
```

## 配置说明

### 环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| MODEL_NAME | 模型名称 | bsroformer_scnet |
| MODEL_PATH | 模型路径 | /app/models |
| DATA_PATH | 数据路径 | /app/data |
| GPU_DEVICE | GPU 设备号 | 0 |
| MAX_BATCH_SIZE | 最大批处理大小 | 1 |
| MAX_CONCURRENT | 最大并发数 | 3 |
| LOG_LEVEL | 日志级别 | INFO |
| PORT | 服务端口 | 8004 |

## 性能优化

### GPU 优化

1. **使用 TensorRT**（可选）:
```bash
pip install tensorrt
```

2. **混合精度推理**:
在 `model_loader.py` 中启用 AMP:
```python
with torch.cuda.amp.autocast():
    output = model(input)
```

### 批量处理

调整 `MAX_BATCH_SIZE` 参数以优化吞吐量：
- 小批量（1-2）：低延迟
- 大批量（4-8）：高吞吐量

## 故障排查

### 常见问题

**1. GPU 不可用**
```bash
# 检查 NVIDIA Docker
docker run --gpus all nvidia/cuda:11.8-base nvidia-smi
```

**2. 内存不足**
- 减小 `MAX_BATCH_SIZE`
- 使用更小的模型

**3. 模型加载失败**
- 检查模型文件是否完整
- 确认 HuggingFace 连接正常

### 查看日志

```bash
# 实时日志
docker-compose logs -f

# 最近 100 行
docker-compose logs --tail=100
```

## 开发指南

### 本地开发

```bash
# 创建虚拟环境
python -m venv venv
source venv/bin/activate  # Windows: venv\Scripts\activate

# 安装依赖
pip install -r requirements.txt

# 运行服务
python server.py
```

### 添加新模型

1. 在 `src/model_loader.py` 中添加模型加载逻辑
2. 更新 `MODEL_NAME` 配置
3. 测试新模型性能

## 基准测试

### 性能指标

| 音频长度 | GPU (RTX 3060) | CPU (i7-12700K) |
|----------|---------------|-----------------|
| 3 分钟    | ~15 秒         | ~45 秒          |
| 5 分钟    | ~25 秒         | ~75 秒          |

*实际性能取决于音频质量和硬件配置*

## 许可证

MIT License

## 致谢

- [BSRoformer](https://github.com/tsurutsu/bsroformer)
- [HuggingFace](https://huggingface.co/)
- [FastAPI](https://fastapi.tiangolo.com/)

## 联系方式

如有问题，请提交 Issue 或联系开发团队。
