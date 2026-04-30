# 一键：超级用户授予 public CREATE + 再跑 001–008 迁移。
# 适用：迁移报「对模式 public 权限不够」且应用用户为 admin。
#
# 用法（先设置 postgres 密码，或 URL 里带密码）:
#   $env:PGPASSWORD = 'postgres的密码'
#   .\scripts\db\bootstrap-migrations.ps1
#
# 或指定超级用户连接串（须能连到目标库）:
#   .\scripts\db\bootstrap-migrations.ps1 -SuperuserUrl "postgresql://postgres:xxx@127.0.0.1:5432/audio_platform"
#
# 应用库连接（跑迁移用）可用环境变量覆盖:
#   $env:DATABASE_URL = "postgresql://admin:admin123@127.0.0.1:5432/audio_platform"

param(
    [string]$SuperuserUrl = "",
    [string]$AppDatabaseUrl = ""
)

$ErrorActionPreference = "Stop"
$here = $PSScriptRoot
$grantFile = Join-Path $here "grant_public_schema_to_admin.sql"
$applyScript = Join-Path $here "apply-all-migrations.ps1"

if (-not $SuperuserUrl) {
    $SuperuserUrl = if ($env:SUPERUSER_DATABASE_URL) { $env:SUPERUSER_DATABASE_URL } else { "postgresql://postgres@127.0.0.1:5432/audio_platform" }
}

if (-not (Test-Path $grantFile)) { throw "missing $grantFile" }
if (-not (Test-Path $applyScript)) { throw "missing $applyScript" }

Write-Host "==> [1/2] GRANT USAGE,CREATE ON SCHEMA public TO admin (superuser)" -ForegroundColor Cyan
Write-Host "    using: $SuperuserUrl" -ForegroundColor DarkGray
& psql $SuperuserUrl -v ON_ERROR_STOP=1 -f $grantFile
if ($LASTEXITCODE -ne 0) {
    Write-Host ""
    Write-Host "授权失败。请确认: 1) postgres 密码已设 `$env:PGPASSWORD 或 URL 内含密码  2) 库名在连接串里为 audio_platform（或已改 grant_public_schema_to_admin.sql 中的用户）" -ForegroundColor Red
    exit $LASTEXITCODE
}

if ($AppDatabaseUrl) {
    $env:DATABASE_URL = $AppDatabaseUrl
}

Write-Host "==> [2/2] apply-all-migrations (001-008)" -ForegroundColor Cyan
& $applyScript
exit $LASTEXITCODE
