# 用 UTF-8 将中文注释重新写入 PostgreSQL，修复 Navicat 中「注释」列显示为 ??? 的问题。
# 前提: Docker 容器 audio-platform-postgres 已启动（默认连库 audio_platform）。
# 用法（仓库根目录）: .\scripts\db\fix-comments-encoding.ps1
#
# Navicat: 连接属性 -> 高级 -> 编码选 UTF-8（或 Automatic），勿用 Latin1。

$ErrorActionPreference = 'Stop'
$root = Resolve-Path (Join-Path $PSScriptRoot '..\..')
$sql = Join-Path $root 'scripts\db\reapply_comments_utf8.sql'
if (-not (Test-Path $sql)) { throw "Missing $sql" }

# Windows 管道到 docker exec 常把非 ASCII 破坏成 ?，改为把 UTF-8 文件拷进容器再 psql -f（字节不经过 PowerShell）
Write-Host 'Re-applying COMMENT ... (docker cp + psql -f inside container)'
$inContainer = '/tmp/reapply_comments_utf8.sql'
docker cp -- "$sql" "audio-platform-postgres:${inContainer}"
if ($LASTEXITCODE -ne 0) { throw "docker cp failed" }
docker exec -e PGCLIENTENCODING=UTF8 -e LANG=C.UTF-8 -e LC_ALL=C.UTF-8 audio-platform-postgres `
    psql -U admin -d audio_platform -v ON_ERROR_STOP=1 -f $inContainer
if ($LASTEXITCODE -ne 0) { throw "psql failed: $LASTEXITCODE" }
Write-Host 'fix-comments-encoding: OK' -ForegroundColor Green
