# 将 doc/swagger 下已生成的 OpenAPI/Swagger 2.0 JSON 导出为 ReDoc 单页 HTML / PDF（与 redoc-cli bundle 思路一致）。
# 仓库根目录执行: .\scripts\export-swagger-redoc.ps1
# 依赖：Node.js（使用 npx，无需全局安装 redoc-cli）
#
# 参数:
#   -SkipGen     跳过 gen-swagger.ps1（默认会先重新生成 JSON）
#   -HtmlOnly    只导出 .html
#   -PdfOnly     只导出 .pdf

param(
    [switch]$SkipGen,
    [switch]$HtmlOnly,
    [switch]$PdfOnly
)

$ErrorActionPreference = 'Stop'
$root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
Set-Location $root

if (-not $SkipGen) {
    & (Join-Path $PSScriptRoot 'gen-swagger.ps1')
}

$exportDir = Join-Path $root 'doc\swagger\export'
New-Item -ItemType Directory -Force -Path $exportDir | Out-Null

$specs = @(
    @{ JsonRel = 'doc\swagger\user\user.json';    Base = 'user-api' },
    @{ JsonRel = 'doc\swagger\device\device.json'; Base = 'device-api' },
    @{ JsonRel = 'doc\swagger\content\content.json'; Base = 'content-api' },
    @{ JsonRel = 'doc\swagger\admin\admin.json';   Base = 'admin-api' }
)

foreach ($s in $specs) {
    $in = Join-Path $root $s.JsonRel
    if (-not (Test-Path $in)) {
        Write-Warning "Skip missing spec: $in"
        continue
    }
    if (-not $PdfOnly) {
        $html = Join-Path $exportDir ($s.Base + '.html')
        Write-Host "redoc-cli bundle -> $html"
        npx --yes redoc-cli bundle $in -o $html
        if ($LASTEXITCODE -ne 0) { throw "redoc-cli html failed for $($s.Base)" }
    }
    if (-not $HtmlOnly) {
        $pdf = Join-Path $exportDir ($s.Base + '.pdf')
        Write-Host "redoc-cli bundle -> $pdf"
        npx --yes redoc-cli bundle $in -o $pdf
        if ($LASTEXITCODE -ne 0) { throw "redoc-cli pdf failed for $($s.Base)" }
    }
}

Write-Host ""
Write-Host "Done. Output: doc/swagger/export/{user-api,device-api,content-api,admin-api}.{html,pdf}"
