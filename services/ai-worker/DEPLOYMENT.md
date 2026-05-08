# AI Worker 容器化部署完整指南

## 目录

1. [快速开始](#快速开始)
2. [Docker 部署](#docker-部署)
3. [Kubernetes 部署](#kubernetes-部署)
4. [监控和日志](#监控和日志)
5. [故障排查](#故障排查)

---

## 快速开始

### 前置要求

- **Docker**: 20.10+
- **NVIDIA Driver**: 535+
- **NVIDIA Container Toolkit**: 1.13+
- **GPU**: 支持 CUDA 的 NVIDIA GPU（推荐 RTX 3060+）

### 一键部署（推荐）

#### Windows

```powershell
cd services/ai-worker
.\deploy.ps1
```

#### Linux/Mac

```bash
cd services/ai-worker
chmod +x deploy.sh
./deploy.sh
```

---

## Docker 部署

### 步骤 1: 安装 NVIDIA Container Toolkit

#### Ubuntu/Debian

```bash
# 添加仓库
distribution=$(. /etc/os-release;echo $ID$VERSION_ID)
curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey | sudo gpg --dearmor -o /usr/share/keyrings/nvidia-container-toolkit.gpg
curl -s -L https://nvidia.github.io/libnvidia-container/$distribution/libnvidia-container.list | \
    sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit.gpg] https://#g' | \
    sudo tee /etc/apt/sources.list.d/nvidia-container-toolkit.list

# 安装
sudo apt-get update
sudo apt-get install -y nvidia-container-toolkit

# 配置 Docker
sudo nvidia-ctk runtime configure --runtime=docker
sudo systemctl restart docker

# 验证
docker run --rm --gpus all nvidia/cuda:12.1.0-base-ubuntu22.04 nvidia-smi
```

### 步骤 2: 构建镜像

```bash
cd services/ai-worker

# 构建镜像
docker build -t ai-worker:v1 .

# 查看镜像
docker images | grep ai-worker
```

### 步骤 3: 启动容器

#### 方式 1: 使用部署脚本

```bash
./deploy.sh
```

#### 方式 2: 手动启动

```bash
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
```

#### 方式 3: Docker Compose

```bash
# 启动所有服务（包括监控）
docker-compose up -d

# 查看状态
docker-compose ps

# 查看日志
docker-compose logs -f ai-worker
```

### 步骤 4: 验证部署

```bash
# 检查容器状态
docker ps | grep ai-worker

# 测试健康检查
curl http://localhost:8004/api/v1/health

# 查看 GPU 使用情况
docker exec ai-worker nvidia-smi

# 测试推理
curl -X POST http://localhost:8004/api/inference/start \
    -H "Content-Type: application/json" \
    -d '{"audio_id": "test_001", "model": "bsroformer"}'
```

---

## Kubernetes 部署

### 前置要求

- **K8s 集群**: 1.20+
- **NVIDIA Device Plugin**: 已安装
- **StorageClass**: 已配置（用于 PVC）
- **kubectl**: 已配置

### 步骤 1: 准备命名空间

```bash
kubectl create namespace audio-ai-platform
```

### 步骤 2: 创建 ConfigMap

```bash
kubectl apply -f k8s/configmap.yaml
```

### 步骤 3: 创建持久化存储

```bash
kubectl apply -f k8s/pvc.yaml
```

### 步骤 4: 部署应用

```bash
# 部署 Deployment
kubectl apply -f k8s/deployment.yaml

# 部署 Service
kubectl apply -f k8s/service.yaml

# 部署 HPA（弹性伸缩）
kubectl apply -f k8s/hpa.yaml
```

### 步骤 5: 验证部署

```bash
# 查看 Pod 状态
kubectl get pods -n audio-ai-platform

# 查看 Service
kubectl get svc -n audio-ai-platform

# 查看 HPA
kubectl get hpa -n audio-ai-platform

# 查看日志
kubectl logs -f deployment/ai-worker -n audio-ai-platform
```

### 步骤 6: 访问服务

#### ClusterIP（集群内部访问）

```bash
# 端口转发
kubectl port-forward svc/ai-worker 8004:8004 -n audio-ai-platform

# 访问
curl http://localhost:8004/api/v1/health
```

#### NodePort（外部访问）

```bash
# 获取节点 IP
kubectl get nodes -o wide

# 访问（替换 NODE_IP）
curl http://NODE_IP:30004/api/v1/health
```

#### LoadBalancer（云环境）

```bash
# 获取外部 IP
kubectl get svc ai-worker-lb -n audio-ai-platform

# 访问
curl http://EXTERNAL_IP/api/v1/health
```

---

## 监控和日志

### Prometheus 监控

```bash
# 启动 Prometheus
docker-compose up -d prometheus

# 访问 Prometheus UI
# http://localhost:9090

# 查询指标
# ai_worker_inference_latency_seconds
# ai_worker_gpu_utilization
```

### Grafana 可视化

```bash
# 启动 Grafana
docker-compose up -d grafana

# 访问 Grafana
# http://localhost:3000
# 用户名：admin
# 密码：admin

# 添加 Prometheus 数据源
# http://prometheus:9090
```

### 查看日志

```bash
# Docker 日志
docker logs -f ai-worker

# K8s 日志
kubectl logs -f deployment/ai-worker -n audio-ai-platform

# 查看最近 100 行
docker logs --tail 100 ai-worker
```

---

## 弹性伸缩配置

### HPA 配置说明

```yaml
# k8s/hpa.yaml
spec:
  minReplicas: 1      # 最小副本数
  maxReplicas: 10     # 最大副本数
  
  # CPU 使用率超过 70% 时扩容
  - type: Resource
    resource:
      name: cpu
      target:
        averageUtilization: 70
  
  # 内存使用率超过 80% 时扩容
  - type: Resource
    resource:
      name: memory
      target:
        averageUtilization: 80
  
  # GPU 使用率超过 70% 时扩容
  - type: Pods
    pods:
      metric:
        name: nvidia_gpu_utilization
      target:
        averageValue: "70"
```

### 手动扩缩容

```bash
# 手动设置副本数
kubectl scale deployment ai-worker --replicas=3 -n audio-ai-platform

# 查看副本状态
kubectl get pods -n audio-ai-platform
```

---

## 故障排查

### 问题 1: 容器无法启动

```bash
# 查看容器日志
docker logs ai-worker

# 检查 GPU 支持
docker run --rm --gpus all nvidia/cuda:12.1.0-base nvidia-smi

# 检查配置文件
docker exec ai-worker cat /app/etc/ai-worker.yaml
```

### 问题 2: GPU 不可用

**错误**: `could not select device driver`

**解决**:
```bash
# 重新安装 NVIDIA Container Toolkit
sudo apt-get install --reinstall nvidia-container-toolkit

# 重启 Docker
sudo systemctl restart docker

# 验证
docker run --rm --gpus all nvidia/cuda:12.1.0-base nvidia-smi
```

### 问题 3: K8s Pod 无法调度

**错误**: `0/3 nodes are available: 3 Insufficient nvidia.com/gpu`

**解决**:
```bash
# 检查 NVIDIA Device Plugin
kubectl get pods -n kube-system | grep nvidia

# 重新安装 Device Plugin
kubectl apply -f https://raw.githubusercontent.com/NVIDIA/k8s-device-plugin/v0.14.0/deployments/static/nvidia-device-plugin.yml
```

### 问题 4: 健康检查失败

```bash
# 检查服务是否运行
docker exec ai-worker curl http://localhost:8004/api/v1/health

# 查看应用日志
docker logs ai-worker | grep ERROR

# 检查端口占用
docker exec ai-worker netstat -tlnp | grep 8004
```

### 问题 5: 模型加载失败

```bash
# 检查模型文件
docker exec ai-worker ls -lh /app/models/

# 重新下载模型
docker exec -it ai-worker bash
cd /app/models
# 手动下载模型
```

---

## 性能优化

### Docker 优化

```bash
# 使用 host 网络（减少网络开销）
docker run --network host ...

# 设置共享内存（提升性能）
docker run --shm-size=1g ...

# 设置 CPU 绑定
docker run --cpuset-cpus="0-3" ...
```

### K8s 优化

```yaml
# 资源请求和限制
resources:
  requests:
    cpu: "1000m"
    memory: "2Gi"
    nvidia.com/gpu: "1"
  limits:
    cpu: "2000m"
    memory: "4Gi"
    nvidia.com/gpu: "1"

# 节点亲和性
affinity:
  nodeAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      nodeSelectorTerms:
      - matchExpressions:
        - key: nvidia.com/gpu
          operator: Exists
```

---

## 更新和回滚

### Docker 更新

```bash
# 构建新镜像
docker build -t ai-worker:v2 .

# 停止旧容器
docker stop ai-worker
docker rm ai-worker

# 启动新容器
docker run -d --name ai-worker ... ai-worker:v2
```

### K8s 更新

```bash
# 更新镜像
kubectl set image deployment/ai-worker ai-worker=ai-worker:v2 -n audio-ai-platform

# 查看更新状态
kubectl rollout status deployment/ai-worker -n audio-ai-platform

# 回滚
kubectl rollout undo deployment/ai-worker -n audio-ai-platform
```

---

## 常用命令速查

### Docker

```bash
# 构建镜像
docker build -t ai-worker:v1 .

# 启动容器
docker run -d --name ai-worker --gpus all -p 8004:8004 ai-worker:v1

# 查看日志
docker logs -f ai-worker

# 进入容器
docker exec -it ai-worker bash

# 查看 GPU
docker exec ai-worker nvidia-smi

# 停止/启动
docker stop/start/restart ai-worker

# 删除
docker rm -f ai-worker
```

### Kubernetes

```bash
# 部署
kubectl apply -f k8s/

# 查看状态
kubectl get pods/svc/hpa -n audio-ai-platform

# 查看日志
kubectl logs -f deployment/ai-worker -n audio-ai-platform

# 扩缩容
kubectl scale deployment ai-worker --replicas=3 -n audio-ai-platform

# 更新
kubectl set image deployment/ai-worker ai-worker=ai-worker:v2 -n audio-ai-platform

# 回滚
kubectl rollout undo deployment/ai-worker -n audio-ai-platform

# 删除
kubectl delete -f k8s/
```

---

## 下一步

- [性能调优指南](docs/PERFORMANCE_OPTIMIZATION.md)
- [监控指标说明](docs/METRICS.md)
- [安全配置](docs/SECURITY.md)
