# 从 doc/swagger/*/*.json 用 OpenAPI Generator 生成前端客户端与 Markdown 接口说明。
# 默认先执行 gen-swagger.ps1（goctl 从 *.api 产出 JSON）。需要本机已安装 Node.js（npx）。
#
# 用法（仓库根目录）:
#   .\scripts\gen-openapi-client.ps1
#   .\scripts\gen-openapi-client.ps1 -Service user
#   .\scripts\gen-openapi-client.ps1 -Generator javascript
#   .\scripts\gen-openapi-client.ps1 -SkipSwagger   # 已有最新 JSON 时跳过 goctl

param(
    [ValidateSet('all', 'user', 'device', 'content')]
    [string]$Service = 'all',
    [ValidateSet('typescript-axios', 'javascript')]
    [string]$Generator = 'typescript-axios',
    [switch]$SkipSwagger
)

$ErrorActionPreference = 'Stop'
$root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
Set-Location $root

if (-not $SkipSwagger) {
    & (Join-Path $PSScriptRoot 'gen-swagger.ps1')
}

$outRoot = Join-Path $root 'admin-ui\src\api\openapi-generated'
$additional = switch ($Generator) {
    'typescript-axios' { 'supportsES6=true,stringEnums=true' }
    'javascript' { 'useES6=true,emitModelMethods=true' }
    default { 'supportsES6=true,stringEnums=true' }
}

$specs = @(
    @{ Name = 'user';    JsonRel = 'doc\swagger\user\user.json';       Out = 'user' },
    @{ Name = 'device';  JsonRel = 'doc\swagger\device\device.json';   Out = 'device' },
    @{ Name = 'content'; JsonRel = 'doc\swagger\content\content.json'; Out = 'content' }
)

foreach ($s in $specs) {
    if ($Service -ne 'all' -and $s.Name -ne $Service) { continue }

    $specPath = Join-Path $root $s.JsonRel
    if (-not (Test-Path $specPath)) {
        throw "Missing spec: $specPath (run .\scripts\gen-swagger.ps1 first)"
    }

    $outDir = Join-Path $outRoot $s.Out
    New-Item -ItemType Directory -Force -Path $outDir | Out-Null
    Write-Host "openapi: $($s.Name) -> $outDir ($Generator)"

    npx --yes @openapitools/openapi-generator-cli generate `
        -i $specPath `
        -g $Generator `
        -o $outDir `
        --additional-properties=$additional

    if ($LASTEXITCODE -ne 0) {
        throw "openapi-generator failed for $($s.Name)"
    }
}

Write-Host ''
Write-Host 'Done. Output: admin-ui/src/api/openapi-generated/<service>/'
Write-Host '  typescript-axios: api.ts, models, docs/*.md'
Write-Host '  javascript: src/api, src/model, docs/*.md (uses superagent; admin-ui uses axios — use TS client or adapt)'
