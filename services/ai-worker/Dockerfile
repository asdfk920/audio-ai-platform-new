# BSRoformer AI Worker - 简化版 Docker 镜像
# 支持 CUDA、FP16/INT8 推理，延迟<100ms

# ============================================
# 运行时环境（CUDA + Python）
# ============================================
FROM nvidia/cuda:12.1.0-runtime-ubuntu22.04

# 避免交互式提示
ENV DEBIAN_FRONTEND=noninteractive

# 安装系统依赖
RUN apt-get update && apt-get install -y --no-install-recommends \
    python3.10 \
    python3-pip \
    python3-dev \
    curl \
    ca-certificates \
    ffmpeg \
    && rm -rf /var/lib/apt/lists/*

# 设置工作目录
WORKDIR /app

# 安装基础依赖
RUN apt-get update && apt-get install -y --no-install-recommends \
    python3-pip python3-dev build-essential \
    && rm -rf /var/lib/apt/lists/*

# 升级 pip
RUN pip3 install --upgrade pip -i https://pypi.tuna.tsinghua.edu.cn/simple

# 安装 PyTorch（使用清华镜像）
RUN pip3 install torch torchvision torchaudio \
    --index-url https://pypi.tuna.tsinghua.edu.cn/simple

# 安装 BSRoFormer
RUN pip3 install BS-RoFormer -i https://pypi.tuna.tsinghua.edu.cn/simple

# 验证安装（检查 bs_roformer 包是否存在）
RUN python3 -c "import bs_roformer; print('BS-RoFormer installed successfully')"

# 安装 Demucs 音频分离模型 ⭐
RUN pip3 install demucs -i https://pypi.tuna.tsinghua.edu.cn/simple

# 验证 Demucs 安装
RUN python3 -c "import demucs; print(f'Demucs installed successfully: {demucs.__version__}')"
RUN demucs --version || echo "Demucs CLI ready"

# 复制编译好的服务（需要在本地先编译好）
COPY ai-worker /app/ai-worker

# 复制配置文件
COPY etc /app/etc

# 创建目录
RUN mkdir -p /app/data/output /app/data/cache /app/data/ai-worker-objects /app/models

# 设置权限
RUN chmod +x /app/ai-worker

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8004/api/v1/health || exit 1

# 暴露端口
EXPOSE 8004

# 启动命令
WORKDIR /app
CMD ["./ai-worker", "-f", "etc/ai-worker.yaml"]
