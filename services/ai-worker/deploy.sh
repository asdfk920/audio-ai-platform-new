#!/bin/bash

set -e

echo "========================================="
echo "  AI Worker Service - 部署脚本"
echo "  音轨分离模型部署 (Demucs/Spleeter)"
echo "========================================="

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查环境
check_environment() {
    log_info "检查运行环境..."
    
    if command -v docker &> /dev/null; then
        log_info "Docker 版本: $(docker --version)"
    else
        log_error "Docker 未安装，请先安装 Docker"
        exit 1
    fi
    
    if command -v nvidia-smi &> /dev/null; then
        log_info "GPU 信息:"
        nvidia-smi --query-gpu=name,memory.total,driver_version --format=csv,noheader
    else
        log_warn "未检测到 NVIDIA GPU，将使用 CPU 模式"
        export GPU_DEVICE=-1
    fi
    
    if [ -f ".env" ]; then
        source .env
        log_info "已加载 .env 配置文件"
    fi
}

# 步骤1: 构建镜像
build_image() {
    log_info "步骤 1/7: 构建 Docker 镜像..."
    
    docker build \
        --build-arg CUDA_VERSION=11.8 \
        --build-arg CUDNN_VERSION=8.6 \
        -t audio-ai-platform/ai-worker:latest \
        -t audio-ai-platform/ai-worker:v1.0.0 \
        .
    
    log_info "镜像构建完成"
    docker images | grep ai-worker
}

# 步骤2: 推送镜像
push_image() {
    log_info "步骤 2/7: 推送镜像到仓库..."
    
    # 如果配置了镜像仓库地址，则推送
    if [ -n "$DOCKER_REGISTRY" ]; then
        docker tag audio-ai-platform/ai-worker:latest ${DOCKER_REGISTRY}/audio-ai-platform/ai-worker:latest
        docker push ${DOCKER_REGISTRY}/audio-ai-platform/ai-worker:latest
        log_info "镜像推送完成"
    else
        log_warn "未配置 DOCKER_REGISTRY，跳过推送"
    fi
}

# 步骤3: 启动服务 (Docker Compose)
start_service() {
    log_info "步骤 3/7: 启动服务..."
    
    mkdir -p logs output static-media models spleeter demucs
    
    docker-compose up -d
    
    sleep 5
    
    if docker-compose ps | grep -q "running"; then
        log_info "服务启动成功"
        docker-compose ps
    else
        log_error "服务启动失败"
        docker-compose logs --tail=50
        exit 1
    fi
}

# 步骤4: 等待模型加载
wait_for_model() {
    log_info "步骤 4/7: 等待 AI 模型初始化..."
    
    MAX_WAIT=300
    WAITED=0
    
    while [ $WAITED -lt $MAX_WAIT ]; do
        if curl -sf http://localhost:8004/api/v1/health > /dev/null 2>&1; then
            log_info "服务健康检查通过"
            break
        fi
        
        echo -n "."
        sleep 5
        WAITED=$((WAITED + 5))
        
        if [ $((WAITED % 30)) -eq 0 ]; then
            echo ""
            log_info "已等待 ${WAITED}s，继续等待..."
        fi
    done
    
    if [ $WAITED -ge $MAX_WAIT ]; then
        log_error "模型初始化超时 (${MAX_WAIT}s)"
        docker-compose logs --tail=100
        exit 1
    fi
}

# 步骤5: 功能测试
run_tests() {
    log_info "步骤 5/7: 运行功能测试..."
    
    # 健康检查
    log_info "测试 1: 健康检查接口"
    curl -s http://localhost:8004/api/v1/health | python3 -m json.tool || true
    
    # 模型信息
    log_info "测试 2: 获取模型信息"
    curl -s http://localhost:8004/api/v1/model/info | python3 -m json.tool || true
    
    log_info "功能测试完成"
}

# 步骤6: K8s 部署 (可选)
deploy_k8s() {
    if [ "$1" == "--k8s" ] || [ "$DEPLOY_TO_K8S" == "true" ]; then
        log_info "步骤 6/7: 部署到 Kubernetes..."
        
        kubectl create namespace audio-ai-platform --dry-run=client -o yaml | kubectl apply -f -
        kubectl apply -f deploy/k8s/
        
        log_info "K8s 部署完成"
        kubectl get pods -n audio-ai-platform -l app=ai-worker
    else
        log_info "步骤 6/7: 跳过 K8s 部署 (使用 --k8s 参数启用)"
    fi
}

# 步骤7: 监控验证
verify_monitoring() {
    log_info "步骤 7/7: 验证监控..."
    
    if command -v kubectl &> /dev/null && kubectl get svc prometheus -n monitoring &> /dev/null; then
        log_info "Prometheus 已就绪"
        kubectl port-forward svc/prometheus 9090:9090 -n monitoring &
        PROM_PID=$!
        sleep 2
        
        curl -s http://localhost:9090/api/v1/targets | python3 -c "
import sys, json
data = json.load(sys.stdin)
targets = data.get('data', {}).get('activeTargets', [])
print(f'活跃目标数: {len(targets)}')
for t in targets:
    print(f'  - {t[\"labels\"].get(\"job\", \"unknown\")}: {t[\"health\"]}')
" || true
        
        kill $PROM_PID 2>/dev/null || true
    else
        log_warn "Kubernetes/Prometheus 未检测到，跳过监控验证"
    fi
}

# 清理函数
cleanup() {
    log_info "清理资源..."
    docker-compose down -v --remove-orphans 2>/dev/null || true
}

# 主流程
main() {
    case "${1:-all}" in
        build)
            check_environment
            build_image
            ;;
        push)
            push_image
            ;;
        start)
            start_service
            wait_for_model
            run_tests
            ;;
        test)
            run_tests
            ;;
        k8s)
            check_environment
            build_image
            push_image
            deploy_k8s --k8s
            verify_monitoring
            ;;
        stop)
            cleanup
            log_info "服务已停止"
            ;;
        restart)
            cleanup
            main start
            ;;
        all|*)
            check_environment
            build_image
            push_image
            start_service
            wait_for_model
            run_tests
            deploy_k8s
            verify_monitoring
            
            echo ""
            echo "========================================="
            echo -e "${GREEN}🎉 部署完成！${NC}"
            echo "========================================="
            echo ""
            echo "服务地址: http://localhost:8004"
            echo "API 文档: http://localhost:8004/swagger/"
            echo "健康检查: http://localhost:8004/api/v1/health"
            echo "模型信息: http://localhost:8004/api/v1/model/info"
            echo ""
            echo "常用命令:"
            echo "  查看日志: docker-compose logs -f"
            echo "  重启服务: ./deploy.sh restart"
            echo "  停止服务: ./deploy.sh stop"
            echo "  清理所有: ./deploy.sh cleanup"
            echo ""
            ;;
    esac
}

main "$@"