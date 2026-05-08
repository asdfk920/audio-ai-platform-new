#!/bin/bash
# AI Worker Docker 部署脚本

set -e

echo "========================================="
echo "AI Worker Docker 部署脚本"
echo "========================================="
echo ""

# 检查 Docker
if ! command -v docker &> /dev/null; then
    echo "✗ Docker 未安装，请先安装 Docker"
    exit 1
fi
echo "✓ Docker 已安装：$(docker --version)"

# 检查 NVIDIA Container Toolkit
if ! docker run --rm --gpus all nvidia/cuda:12.1.0-base-ubuntu22.04 nvidia-smi &> /dev/null; then
    echo "⚠ NVIDIA Container Toolkit 可能未正确配置"
    echo "  请确保已安装 nvidia-container-toolkit"
fi

echo ""

# 构建镜像
echo "[步骤 1/3] 构建 Docker 镜像..."
docker build -t ai-worker:v1 .

if [ $? -eq 0 ]; then
    echo "✓ 镜像构建成功"
else
    echo "✗ 镜像构建失败"
    exit 1
fi

echo ""

# 查看镜像
echo "[步骤 2/3] 查看镜像..."
docker images | grep ai-worker

echo ""

# 启动容器
echo "[步骤 3/3] 启动容器..."

# 停止旧容器
docker stop ai-worker 2>/dev/null || true
docker rm ai-worker 2>/dev/null || true

# 启动新容器
docker run -d \
    --name ai-worker \
    --gpus all \
    -p 8004:8004 \
    -p 9090:9090 \
    -v $(pwd)/models:/app/models \
    -v $(pwd)/data/output:/app/data/output \
    -v $(pwd)/data/cache:/app/data/cache \
    -v $(pwd)/etc/ai-worker.yaml:/app/etc/ai-worker.yaml:ro \
    --restart unless-stopped \
    ai-worker:v1

echo ""

# 等待服务启动
echo "等待服务启动..."
sleep 10

# 检查容器状态
echo ""
echo "容器状态:"
docker ps | grep ai-worker

# 查看日志
echo ""
echo "最近日志:"
docker logs --tail 20 ai-worker

# 测试健康检查
echo ""
echo "测试健康检查:"
if curl -f http://localhost:8004/api/v1/health &> /dev/null; then
    echo "✓ 服务健康检查通过"
else
    echo "⚠ 服务健康检查失败，请查看日志"
fi

echo ""
echo "========================================="
echo "部署完成！"
echo "========================================="
echo ""
echo "服务信息:"
echo "  容器名称：ai-worker"
echo "  服务端口：http://localhost:8004"
echo "  监控端口：http://localhost:8090"
echo ""
echo "常用命令:"
echo "  查看日志：docker logs -f ai-worker"
echo "  停止服务：docker stop ai-worker"
echo "  启动服务：docker start ai-worker"
echo "  删除容器：docker rm -f ai-worker"
echo ""
