# BSRoformer AI Worker - 本地编译和 Docker 构建脚本

Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "AI Worker - 本地编译 + Docker 构建" -ForegroundColor Cyan
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host ""

# 步骤 1: 本地编译 Go 服务
Write-Host "[步骤 1/4] 本地编译 Go 服务..." -ForegroundColor Yellow

# 设置 Go 代理
$env:GOPROXY = "https://goproxy.cn,direct"

# 编译 Linux 版本
Write-Host "  编译 Linux 版本 (CGO_ENABLED=0 GOOS=linux)..." -ForegroundColor Gray
$env:CGO_ENABLED = "0"
$env:GOOS = "linux"
$env:GOARCH = "amd64"

# 执行编译
go build -o ai-worker .

if ($LASTEXITCODE -ne 0) {
    Write-Host "✗ 编译失败" -ForegroundColor Red
    exit 1
}

Write-Host "✓ 编译成功：ai-worker" -ForegroundColor Green
Write-Host ""

# 步骤 2: 验证编译结果
Write-Host "[步骤 2/4] 验证编译结果..." -ForegroundColor Yellow
if (Test-Path "ai-worker") {
    $fileSize = (Get-Item "ai-worker").Length / 1MB
    Write-Host "  文件大小：{0:N2} MB" -f $fileSize -ForegroundColor Gray
    Write-Host "✓ 编译文件存在" -ForegroundColor Green
} else {
    Write-Host "✗ 编译文件不存在" -ForegroundColor Red
    exit 1
}
Write-Host ""

# 步骤 3: 构建 Docker 镜像
Write-Host "[步骤 3/4] 构建 Docker 镜像..." -ForegroundColor Yellow
docker build -t ai-worker:v1 .

if ($LASTEXITCODE -ne 0) {
    Write-Host "✗ Docker 构建失败" -ForegroundColor Red
    exit 1
}

Write-Host "✓ Docker 镜像构建成功" -ForegroundColor Green
Write-Host ""

# 步骤 4: 查看镜像信息
Write-Host "[步骤 4/4] 查看镜像信息..." -ForegroundColor Yellow
docker images ai-worker:v1

Write-Host ""
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "部署完成！" -ForegroundColor Green
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "下一步操作:" -ForegroundColor Cyan
Write-Host "  1. 启动容器:" -ForegroundColor Gray
Write-Host "     docker run -d --name ai-worker --gpus all -p 8004:8004 ai-worker:v1" -ForegroundColor Gray
Write-Host ""
Write-Host "  2. 查看日志:" -ForegroundColor Gray
Write-Host "     docker logs -f ai-worker" -ForegroundColor Gray
Write-Host ""
Write-Host "  3. 测试健康检查:" -ForegroundColor Gray
Write-Host "     curl http://localhost:8004/api/v1/health" -ForegroundColor Gray
Write-Host ""
