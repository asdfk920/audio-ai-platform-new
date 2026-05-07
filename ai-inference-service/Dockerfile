FROM nvidia/cuda:11.8-cudnn8-runtime-ubuntu22.04

# 设置环境变量
ENV PYTHONUNBUFFERED=1
ENV DEBIAN_FRONTEND=noninteractive
ENV PYTHONDONTWRITEBYTECODE=1

# 安装 Python 和依赖
RUN apt-get update && apt-get install -y \
    python3.10 \
    python3.10-dev \
    python3-pip \
    ffmpeg \
    libsndfile1 \
    curl \
    git \
    && rm -rf /var/lib/apt/lists/*

# 设置工作目录
WORKDIR /app

# 复制 requirements
COPY requirements.txt .

# 安装 Python 依赖（使用国内镜像加速）
RUN pip3 install --no-cache-dir -r requirements.txt -i https://pypi.tuna.tsinghua.edu.cn/simple

# 复制模型文件
COPY models/ /app/models/

# 复制代码
COPY src/ /app/src/
COPY server.py .

# 暴露端口
EXPOSE 8004

# 启动命令
CMD ["python3", "server.py"]
