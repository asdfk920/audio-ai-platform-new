# AI Worker Docker 部署脚本 (PowerShell)

Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "AI Worker Docker 部署脚本" -ForegroundColor Cyan
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host ""

# 检查 Docker
try {
    $dockerVersion = docker --version 2>&1
    Write-Host "✓ Docker 已安装：$dockerVersion" -ForegroundColor Green
} catch {
    Write-Host "✗ Docker 未安装，请先安装 Docker Desktop" -ForegroundColor Red
    exit 1
}

# 检查 NVIDIA Container Toolkit
Write-Host "`n 检查 GPU 支持..." -ForegroundColor Yellow
$testGpu = docker run --rm --gpus all nvidia/cuda:12.1.0-base-ubuntu22.04 nvidia-smi 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "⚠ NVIDIA Container Toolkit 可能未正确配置" -ForegroundColor Yellow
    Write-Host "  请确保已安装 NVIDIA Container Toolkit" -ForegroundColor Gray
} else {
    Write-Host "✓ GPU 支持正常" -ForegroundColor Green
}

Write-Host ""

# 构建镜像
Write-Host "[步骤 1/3] 构建 Docker 镜像..." -ForegroundColor Yellow
docker build -t ai-worker:v1 .

if ($LASTEXITCODE -eq 0) {
    Write-Host "✓ 镜像构建成功" -ForegroundColor Green
} else {
    Write-Host "✗ 镜像构建失败" -ForegroundColor Red
    exit 1
}

Write-Host ""

# 查看镜像
Write-Host "[步骤 2/3] 查看镜像..." -ForegroundColor Yellow
docker images | Select-String "ai-worker"

Write-Host ""

# 启动容器
Write-Host "[步骤 3/3] 启动容器..." -ForegroundColor Yellow

# 停止旧容器
Write-Host "停止旧容器..." -ForegroundColor Gray
docker stop ai-worker 2>$null | Out-Null
docker rm ai-worker 2>$null | Out-Null

# 启动新容器
Write-Host "启动新容器..." -ForegroundColor Gray
docker run -d `
    --name ai-worker `
    --gpus all `
    -p 8004:8004 `
    -p 9090:9090 `
    -v ${PWD}\models:/app/models `
    -v ${PWD}\data\output:/app/data/output `
    -v ${PWD}\data\cache:/app/data/cache `
    -v ${PWD}\etc\ai-worker.yaml:/app/etc/ai-worker.yaml:ro `
    --restart unless-stopped `
    ai-worker:v1

Write-Host ""

# 等待服务启动
Write-Host "等待服务启动..." -ForegroundColor Gray
Start-Sleep -Seconds 10

# 检查容器状态
Write-Host "`n 容器状态:" -ForegroundColor Yellow
docker ps | Select-String "ai-worker"

# 查看日志
Write-Host "`n 最近日志:" -ForegroundColor Yellow
docker logs --tail 20 ai-worker

# 测试健康检查
Write-Host "`n 测试健康检查:" -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8004/api/v1/health" -TimeoutSec 5 -UseBasicParsing
    if ($response.StatusCode -eq 200) {
        Write-Host "✓ 服务健康检查通过" -ForegroundColor Green
    }
} catch {
    Write-Host "⚠ 服务健康检查失败，请查看日志" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "=========================================" -ForegroundColor Green
Write-Host "部署完成！" -ForegroundColor Green
Write-Host "=========================================" -ForegroundColor Green
Write-Host ""
Write-Host "服务信息:" -ForegroundColor Cyan
Write-Host "  容器名称：ai-worker"
Write-Host "  服务端口：http://localhost:8004"
Write-Host "  监控端口：http://localhost:9090"
Write-Host ""
Write-Host "常用命令:" -ForegroundColor Cyan
Write-Host "  查看日志：docker logs -f ai-worker"
Write-Host "  停止服务：docker stop ai-worker"
Write-Host "  启动服务：docker start ai-worker"
Write-Host "  删除容器：docker rm -f ai-worker"
Write-Host ""
