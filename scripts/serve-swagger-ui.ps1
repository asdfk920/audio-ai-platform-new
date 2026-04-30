# 生成 Swagger JSON 并启动本地 Swagger UI（浏览器查看接口文档）。
# 在项目根目录执行: .\scripts\serve-swagger-ui.ps1
# 默认 http://127.0.0.1:8090/  （需本机 Node.js，用于 npx http-server）

param(
    [int]$Port = 8090,
    [switch]$SkipGen
)

$ErrorActionPreference = 'Stop'
$root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
Set-Location $root

if (-not $SkipGen) {
    & (Join-Path $PSScriptRoot 'gen-swagger.ps1')
}

$swaggerDir = Join-Path $root 'doc\swagger'
$index = Join-Path $swaggerDir 'index.html'
if (-not (Test-Path $index)) {
    throw "Missing doc/swagger/index.html"
}

$base = "http://127.0.0.1:$Port"
Write-Host ""
Write-Host "Swagger UI: $base/?spec=user   (device | content)"
Write-Host "Press Ctrl+C to stop."
Write-Host ""

try {
    Start-Process $base
} catch {
    # headless / no GUI
}

Set-Location $swaggerDir
npx --yes http-server . -p $Port -c-1 --cors
