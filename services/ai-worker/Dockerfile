# ============================================
# AI Worker Service - 音轨分离模型部署
# 支持模型: Demucs, Spleeter, BSRFormer
# GPU: CUDA 11.8 / cuDNN 8.6
# ============================================

# 阶段1: 基础环境
FROM nvidia/cuda:11.8.0-cudnn8-runtime-ubuntu22.04 AS base

ENV DEBIAN_FRONTEND=noninteractive
ENV TZ=Asia/Shanghai

RUN apt-get update && apt-get install -y --no-install-recommends \
    python3.10 \
    python3-pip \
    python3-venv \
    ffmpeg \
    libsndfile1 \
    sox \
    libsox-dev \
    libsox-fmt-all \
    git \
    curl \
    wget \
    && rm -rf /var/lib/apt/lists/*

# 阶段2: Python 环境 + PyTorch
FROM base AS python-env

RUN python3 -m venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"

# 安装 PyTorch (CUDA 11.8)
RUN pip install --no-cache-dir \
    torch==2.1.0+cu118 \
    torchaudio==2.1.0+cu118 \
    -f https://download.pytorch.org/whl/cu118/torch_stable.html

# 安装音轨分离库
RUN pip install --no-cache-dir \
    demucs==4.0.5 \
    spleeter==2.3.0 \
    librosa==0.10.1 \
    soundfile==0.12.1 \
    numpy==1.24.3 \
    tqdm==65.0.0

# 阶段3: 模型下载 (可选，也可运行时下载)
FROM python-env AS model-download

WORKDIR /app/models

# Demucs 模型 (htdemucs - 高质量)
RUN demucs --download htdemucs || echo "Model download skipped, will download on first use"

# Spleeter 模型
RUN spleeter download 'spleeter:4stems' || echo "Spleeter download skipped"

# 阶段4: 应用构建
FROM python-env AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ai-worker .

# 阶段5: 最终镜像
FROM python-env AS runtime

LABEL maintainer="audio-ai-platform"
LABEL description="AI Worker Service - Audio Source Separation with GPU Support"
LABEL version="1.0.0"

# 创建非 root 用户
RUN useradd -m -u 1000 aiworker && mkdir -p /app && chown -R aiworker:aiworker /app
USER aiworker

WORKDIR /app

# 复制应用二进制文件
COPY --from=builder /build/ai-worker .

# 创建必要目录
RUN mkdir -p /app/models /app/output /app/cache /app/static-media /app/logs

# 环境变量配置
ENV MODEL_BASE_DIR=/app/models
ENV OUTPUT_DIR=/app/output
ENV CACHE_DIR=/app/cache
ENV GPU_DEVICE=0
ENV MAX_CONCURRENT_JOBS=3
ENV DEFAULT_MODEL=htdemucs
ENV SEGMENT_SIZE=10
ENV OVERLAP=0.25
ENV USE_FP16=true

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:8004/api/v1/health || exit 1

# 暴露端口
EXPOSE 8004

# 启动命令
ENTRYPOINT ["./ai-worker"]
CMD ["-f", "etc/ai-worker.yaml"]