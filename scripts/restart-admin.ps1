# 仅重启 go-admin（Windows）
$ErrorActionPreference = "Stop"
$ProjectRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$LogDir = Join-Path $ProjectRoot "logs"
$pidFile = Join-Path $LogDir "admin.pid"

if (Test-Path $pidFile) {
    $raw = (Get-Content -LiteralPath $pidFile -Raw).Trim()
    if ($raw -match '^\d+$') {
        $id = [int]$raw
        $p = Get-Process -Id $id -ErrorAction SilentlyContinue
        if ($p) {
            Write-Host "Stopping go-admin (pid=$id)..."
            Stop-Process -Id $id -Force -ErrorAction SilentlyContinue
        }
    }
    Remove-Item -LiteralPath $pidFile -Force -ErrorAction SilentlyContinue
}

# pid 文件可能过期或曾手动启动：释放 8000 端口（与 config 默认一致）
$conns = Get-NetTCPConnection -LocalPort 8000 -State Listen -ErrorAction SilentlyContinue
foreach ($c in $conns) {
    $op = [int]$c.OwningProcess
    if ($op -gt 0) {
        Write-Host "Stopping process on :8000 (pid=$op)..."
        Stop-Process -Id $op -Force -ErrorAction SilentlyContinue
    }
}
Start-Sleep -Seconds 1

New-Item -ItemType Directory -Force -Path $LogDir | Out-Null
$adminDir = Join-Path $ProjectRoot "admin"
$adminLog = Join-Path $LogDir "admin.log"
$adminErr = Join-Path $LogDir "admin.err.log"

Write-Host "Starting go-admin..."
$proc = Start-Process -FilePath "go" -ArgumentList @("run", "main.go", "server", "-c", "config/settings.yml") `
    -WorkingDirectory $adminDir -PassThru -WindowStyle Hidden `
    -RedirectStandardOutput $adminLog -RedirectStandardError $adminErr
$proc.Id | Out-File -FilePath $pidFile -Encoding utf8
Write-Host "go-admin started: pid=$($proc.Id)  http://127.0.0.1:8000"
Write-Host "Logs: $adminLog | $adminErr"
