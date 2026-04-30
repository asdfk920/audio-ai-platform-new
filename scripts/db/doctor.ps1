# Windows 本地 DB 自检脚本：端口/权限/缺表诊断
# 用法：
#   .\scripts\db\doctor.ps1
# 可选覆盖：
#   $env:POSTGRES_PORT=5432
#   $env:DATABASE_URL="postgresql://admin:xxx@localhost:5432/audio_platform"

$ErrorActionPreference = "Stop"

$root = Resolve-Path (Join-Path $PSScriptRoot "..\..")

function Write-Step($msg) { Write-Host "==> $msg" -ForegroundColor Cyan }
function Write-Ok($msg) { Write-Host "OK  $msg" -ForegroundColor Green }
function Write-Warn($msg) { Write-Host "WARN $msg" -ForegroundColor Yellow }
function Write-Bad($msg) { Write-Host "ERR  $msg" -ForegroundColor Red }

$port = if ($env:POSTGRES_PORT) { $env:POSTGRES_PORT } else { "5433" }
$dbUrl = if ($env:DATABASE_URL) { $env:DATABASE_URL } else { "postgresql://admin:admin123@127.0.0.1:$port/audio_platform" }

Write-Step "Target DATABASE_URL = $dbUrl"

Write-Step "Check TCP port $port"
try {
    $r = Test-NetConnection -ComputerName 127.0.0.1 -Port ([int]$port) -WarningAction SilentlyContinue
    if (-not $r.TcpTestSucceeded) {
        Write-Bad "127.0.0.1:$port not reachable. If using docker-compose, run: docker compose up -d"
        return
    }
    Write-Ok "Port reachable"
} catch {
    Write-Warn "Test-NetConnection failed: $($_.Exception.Message)"
}

Write-Step "Check psql availability"
$psql = (Get-Command psql -ErrorAction SilentlyContinue)
if (-not $psql) {
    Write-Warn "psql not found in PATH. You can still migrate via DOCKER_POSTGRES_CONTAINER=audio-platform-postgres"
} else {
    Write-Ok "psql found: $($psql.Path)"
}

if ($psql) {
    Write-Step "Check DB connectivity (admin)"
    $env:PGCLIENTENCODING = 'UTF8'
    & psql $dbUrl -v ON_ERROR_STOP=1 -t -c "SELECT 1;" | Out-Null
    if ($LASTEXITCODE -ne 0) {
        Write-Bad "Cannot connect with admin. Verify user/password/dbname/port."
        Write-Warn "Tip: If you are using local PostgreSQL (5432), set: `$env:POSTGRES_PORT=5432"
        return
    }
    Write-Ok "Connected"

    Write-Step "Check admin CREATE privilege on schema public"
    $canCreate = & psql $dbUrl -t -c "SELECT has_schema_privilege('admin','public','CREATE');"
    $canCreate = ($canCreate | Out-String).Trim()
    if ($canCreate -ne "t") {
        Write-Warn "admin has no CREATE on public. Fix (superuser): scripts/db/grant_public_schema_to_admin.sql"
    } else {
        Write-Ok "admin can CREATE on public"
    }

    Write-Step "Check key tables"
    $q = @"
SELECT
  (SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name='users') AS users,
  (SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name='roles') AS roles,
  (SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name='sys_menu') AS sys_menu,
  (SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name='casbin_rule') AS casbin_rule;
"@
    $counts = & psql $dbUrl -t -A -F "," -c $q
    Write-Host "tables(users,roles,sys_menu,casbin_rule)=$counts"
    Write-Ok "Doctor done"
}

