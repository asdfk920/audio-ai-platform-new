# Windows 停止脚本：按 pid 文件停服务，并可选 docker compose down
# 用法：
#   .\scripts\stop-windows.ps1
#   .\scripts\stop-windows.ps1 -DockerDown

param(
    [switch]$DockerDown
)

$ErrorActionPreference = "Continue"
$ProjectRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$LogDir = Join-Path $ProjectRoot "logs"

function Stop-PidFile($path, $name) {
    if (-not (Test-Path $path)) { return }
    try {
        $pid = (Get-Content -LiteralPath $path -Raw).Trim()
        if ($pid -match '^\d+$') {
            $p = Get-Process -Id ([int]$pid) -ErrorAction SilentlyContinue
            if ($p) {
                Write-Host "Stopping $name (pid=$pid)..." -ForegroundColor Cyan
                Stop-Process -Id ([int]$pid) -Force -ErrorAction SilentlyContinue
            }
        }
    } catch {
    }
    Remove-Item -LiteralPath $path -Force -ErrorAction SilentlyContinue
}

Write-Host "==========================================" 
Write-Host "  Stop Audio AI Platform (Windows)" 
Write-Host "=========================================="
Write-Host ""

if (Test-Path $LogDir) {
    Stop-PidFile (Join-Path $LogDir "admin.pid") "go-admin"
    Stop-PidFile (Join-Path $LogDir "user-service.pid") "user service"
    Stop-PidFile (Join-Path $LogDir "device-service.pid") "device service"
    Stop-PidFile (Join-Path $LogDir "content-service.pid") "content service"
}

if ($DockerDown) {
    Write-Host ""
    Write-Host "docker compose down..." -ForegroundColor Cyan
    try {
        Set-Location $ProjectRoot
        docker compose down | Out-Null
    } catch {
        Write-Host "docker compose down failed: $($_.Exception.Message)" -ForegroundColor Yellow
    }
}

Write-Host ""
Write-Host "Done."
