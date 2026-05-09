# AI Worker Multi-stage Build
# Build from PROJECT ROOT directory!
# Usage: docker build -f services/ai-worker/Dockerfile -t ai-worker:v1 .

# ============================================
# Stage 1: Build Go binary (Linux AMD64)
# ============================================
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Configure Go China proxy
ENV GOPROXY=https://goproxy.cn,direct
ENV GO111MODULE=on

# Copy ENTIRE project root (for local replace paths in go.mod)
COPY . .

# Change to ai-worker directory
WORKDIR /build/services/ai-worker

# Download dependencies
RUN go mod download

# Build for Linux AMD64
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o ai-worker .

# ============================================
# Stage 2: Runtime environment (CUDA + Python)
# ============================================
FROM nvidia/cuda:12.1.0-runtime-ubuntu22.04

ENV DEBIAN_FRONTEND=noninteractive

# Install system dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    python3.10 \
    python3-pip \
    python3-dev \
    curl \
    ca-certificates \
    ffmpeg \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Install build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    python3-pip python3-dev build-essential \
    && rm -rf /var/lib/apt/lists/*

# Upgrade pip
RUN pip3 install --upgrade pip -i https://pypi.tuna.tsinghua.edu.cn/simple

# Install PyTorch
RUN pip3 install torch torchvision torchaudio \
    --index-url https://pypi.tuna.tsinghua.edu.cn/simple

# Install BSRoFormer
RUN pip3 install BS-RoFormer -i https://pypi.tuna.tsinghua.edu.cn/simple
RUN python3 -c "import bs_roformer; print('BS-RoFormer OK')"

# Install Demucs
RUN pip3 install demucs -i https://pypi.tuna.tsinghua.edu.cn/simple
RUN python3 -c "import demucs; print(f'Demucs OK: {demucs.__version__}')"
RUN demucs --version || echo "Demucs CLI ready"

# Copy compiled binary from builder stage
COPY --from=builder /build/services/ai-worker/ai-worker /app/ai-worker

# Copy config files
COPY services/ai-worker/etc /app/etc

# Create directories
RUN mkdir -p /app/data/output /app/data/cache /app/data/ai-worker-objects /app/models

# Set permissions
RUN chmod +x /app/ai-worker

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8004/api/v1/health || exit 1

EXPOSE 8004

WORKDIR /app
CMD ["./ai-worker", "-f", "etc/ai-worker.yaml"]
