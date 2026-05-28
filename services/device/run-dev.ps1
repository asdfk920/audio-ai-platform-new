# Local dev launcher for device service (Windows; avoids go run temp exe issues)
$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot

$gotmp = Join-Path $PSScriptRoot ".go-tmp"
if (-not (Test-Path $gotmp)) { New-Item -ItemType Directory -Path $gotmp | Out-Null }
$env:GOTMPDIR = $gotmp

Write-Host ">> go build -o device-api.exe ."
go build -o device-api.exe .
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

$listen = Get-NetTCPConnection -LocalPort 8002 -State Listen -ErrorAction SilentlyContinue | Select-Object -First 1
if ($listen) {
    Write-Host ">> Port 8002 already in use (PID=$($listen.OwningProcess)). Service running; skip start."
    exit 0
}

Write-Host ">> .\device-api.exe -f etc/device.yaml"
& .\device-api.exe -f etc/device.yaml
