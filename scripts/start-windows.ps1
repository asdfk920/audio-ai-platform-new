# Audio AI Platform - Windows 一键启动脚本
# 在项目根目录执行: .\scripts\start-windows.ps1  或  powershell -ExecutionPolicy Bypass -File scripts\start-windows.ps1

$ErrorActionPreference = "Stop"
$ProjectRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
Set-Location $ProjectRoot

Write-Host "=========================================="
Write-Host "  启动 Audio AI Platform (Windows)"
Write-Host "=========================================="
Write-Host ""

# 1. 启动 Docker
Write-Host "[1/4] 启动 Docker 服务 (PostgreSQL, Redis, LocalStack)..."
docker-compose up -d
if ($LASTEXITCODE -ne 0) {
    Write-Host "错误: docker-compose 启动失败，请确认 Docker Desktop 已运行。" -ForegroundColor Red
    exit 1
}
Write-Host "      等待服务就绪..."
Start-Sleep -Seconds 5
Write-Host "      PostgreSQL(Docker): localhost:5433 | Redis: localhost:6379 | LocalStack: localhost:4566"
Write-Host ""

# 2. 数据库自检 + 全量迁移（通过 Docker 执行，无需本机安装 psql）
Write-Host "[2/4] 数据库自检 + 执行数据库迁移(001-019)..."
$env:POSTGRES_PORT = if ($env:POSTGRES_PORT) { $env:POSTGRES_PORT } else { "5433" }
$env:DOCKER_POSTGRES_CONTAINER = if ($env:DOCKER_POSTGRES_CONTAINER) { $env:DOCKER_POSTGRES_CONTAINER } else { "audio-platform-postgres" }
try {
    & (Join-Path $ProjectRoot "scripts\db\doctor.ps1")
} catch {
    Write-Host "      doctor 脚本执行失败：$($_.Exception.Message)" -ForegroundColor Yellow
}
& (Join-Path $ProjectRoot "scripts\db\apply-all-migrations.ps1")
if ($LASTEXITCODE -ne 0) {
    Write-Host "      迁移失败，详见上方输出；可先执行 scripts/db/grant_public_schema_to_admin.sql 再重试。" -ForegroundColor Red
    exit $LASTEXITCODE
}

# Casbin gorm-adapter 在部分环境下可能因历史表结构导致 panic（insufficient arguments）；
# 启动前清理 casbin_rule，让服务自动重建（策略会为空，需要在后台重新配置菜单/API 权限）。
Write-Host "      清理 casbin_rule（若存在）..."
docker exec -i audio-platform-postgres psql -U admin -d audio_platform -c "DROP TABLE IF EXISTS casbin_rule CASCADE;" 2>$null | Out-Null
Write-Host ""

# 3. 创建日志目录并启动 go-admin
$LogDir = Join-Path $ProjectRoot "logs"
New-Item -ItemType Directory -Force -Path $LogDir | Out-Null

Write-Host "[3/4] 启动 go-admin 后台..."
$adminLog = Join-Path $LogDir "admin.log"
$adminPid = Join-Path $LogDir "admin.pid"
$adminProc = Start-Process -FilePath "go" -ArgumentList "run", "main.go", "server", "-c", "config/settings.yml" `
    -WorkingDirectory (Join-Path $ProjectRoot "admin") `
    -PassThru -WindowStyle Hidden `
    -RedirectStandardOutput $adminLog -RedirectStandardError (Join-Path $LogDir "admin.err.log")
$adminProc.Id | Out-File $adminPid -Encoding utf8
Start-Sleep -Seconds 3
Write-Host "      go-admin 后台: http://localhost:8000"
Write-Host ""

# 4. 启动微服务
Write-Host "[4/4] 启动微服务..."
$services = @(
    @{ Name = "user";   Port = 8001; Log = "user-service.log" },
    @{ Name = "device"; Port = 8002; Log = "device-service.log" },
    @{ Name = "content"; Port = 8003; Log = "content-service.log" }
)
foreach ($svc in $services) {
    $workDir = Join-Path $ProjectRoot "services\$($svc.Name)"
    $outLog = Join-Path $LogDir $svc.Log
    $pidFile = Join-Path $LogDir ($svc.Log -replace '\.log$','.pid')
    $goFile = switch ($svc.Name) { "user" { "user.go" }; "device" { "device.go" }; "content" { "content.go" }; default { "$($svc.Name).go" } }
    $p = Start-Process -FilePath "go" -ArgumentList "run", $goFile `
        -WorkingDirectory $workDir `
        -PassThru -WindowStyle Hidden `
        -RedirectStandardOutput $outLog -RedirectStandardError (Join-Path $LogDir ($svc.Log -replace '\.log$','.err.log'))
    $p.Id | Out-File $pidFile -Encoding utf8
    Write-Host "      $($svc.Name) 服务: http://localhost:$($svc.Port)"
    Start-Sleep -Seconds 2
}

Write-Host ""
Write-Host "探测服务状态..."
function Probe-Http($url, $name) {
    try {
        $r = Invoke-WebRequest -Uri $url -UseBasicParsing -TimeoutSec 2
        Write-Host "      OK  $name $url ($($r.StatusCode))" -ForegroundColor Green
        return $true
    } catch {
        Write-Host "      ERR $name $url ($($_.Exception.Message))" -ForegroundColor Yellow
        return $false
    }
}
# 后端健康检查
Probe-Http "http://127.0.0.1:8000/api/v1/health" "go-admin"
# 前端界面需手动启动；这里仅做提示与探测
Probe-Http "http://127.0.0.1:9527/" "admin-ui(可选)"

Write-Host ""
Write-Host "=========================================="
Write-Host "  所有服务已启动"
Write-Host "=========================================="
Write-Host ""
Write-Host "访问地址:"
Write-Host "  - go-admin 后台:  http://localhost:8000"
Write-Host "  - go-admin 前端: http://localhost:9527 (需在 admin-ui 目录执行 npm run dev)"
Write-Host "  - 用户服务:       http://localhost:8001"
Write-Host "  - 设备服务:       http://localhost:8002"
Write-Host "  - 内容服务:       http://localhost:8003"
Write-Host ""
Write-Host "日志目录: $LogDir"
Write-Host "停止服务: .\scripts\stop-windows.ps1"
Write-Host ""
